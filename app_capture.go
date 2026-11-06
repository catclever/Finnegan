package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"Finnegan/core/module"
	"context"

	"Finnegan/core/bus"
	"Finnegan/core/chatlog"
	"Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/watcher"
	"Finnegan/modules/curio"

	"github.com/oklog/ulid/v2"
)

// StripThreadMentions strips leading @<th.Title> mentions from card body
// so that theme associations in frontmatter do not duplicate at the beginning of the card text.
func StripThreadMentions(text string, threads []ThreadRef) string {
	res := strings.TrimSpace(text)
	for _, th := range threads {
		if th.Title == "" {
			continue
		}
		targetToken := "@" + th.Title
		if strings.HasPrefix(res, targetToken) {
			res = strings.TrimPrefix(res, targetToken)
			res = strings.TrimLeft(res, " \t\r\n-_:，。！？,. ")
		}
	}
	trimmed := strings.TrimSpace(res)
	if trimmed == "" {
		return text
	}
	return trimmed
}

func sanitizeSlug(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "_")
	s = strings.ReplaceAll(s, "\t", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "@", "")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.Trim(s, "._-@ ")
	if len([]rune(s)) > 20 {
		runes := []rune(s)
		return strings.TrimRight(string(runes[:20]), "._-@ ")
	}
	return s
}

func deriveCaptureSlugSource(body string, threads []ThreadRef, fallback string) string {
	slugSource := body
	for _, th := range threads {
		targetToken := "@" + th.Title
		if strings.HasPrefix(slugSource, targetToken) {
			slugSource = strings.TrimPrefix(slugSource, targetToken)
			slugSource = strings.TrimLeft(slugSource, " \t\r\n-_:，。！？,. ")
		}
	}
	if slugSource == "" {
		if len(threads) > 0 {
			slugSource = threads[0].Title
		} else {
			slugSource = fallback
		}
	}
	return slugSource
}

// StripAndExtractHashtags strips trailing/standalone hashtags completely from card body,
// and strips '#' from inline hashtags to preserve grammatical fluency.
func StripAndExtractHashtags(text string) (string, []string) {
	return keywords.StripAndExtractHashtags(text)
}

// ExtractHashtags extracts clean hashtags from text.
// Supports Chinese characters, letters, digits, underscores, hyphens, and dots.
// Matches tokens like #tag, #WebKitGTK, #跨平台UI, #v1.0
func ExtractHashtags(text string) []string {
	_, tags := keywords.StripAndExtractHashtags(text)
	return tags
}

// ExtractThreadRefs extracts @<cardTitle> references from text and resolves them to ThreadRef
func (a *App) ExtractThreadRefs(text string) []ThreadRef {
	if !strings.Contains(text, "@") {
		return nil
	}

	var allCards []Card
	if a != nil {
		allCards = append(allCards, a.LoadLocalCards()...)
		if a.CardManager != nil {
			allCards = append(allCards, a.CardManager.GetTempCards()...)
		}
	}

	seenIDs := make(map[string]bool)
	var threads []ThreadRef

	// Sort cards by Title length descending so longer titles match before substrings
	sortCards := make([]Card, len(allCards))
	copy(sortCards, allCards)
	sort.SliceStable(sortCards, func(i, j int) bool {
		return len(sortCards[i].Title) > len(sortCards[j].Title)
	})

	for _, c := range sortCards {
		trimmedTitle := strings.TrimSpace(c.Title)
		if trimmedTitle == "" || c.ID == "" {
			continue
		}
		if seenIDs[c.ID] {
			continue
		}
		targetToken := "@" + trimmedTitle
		if strings.Contains(text, targetToken) {
			seenIDs[c.ID] = true
			threads = append(threads, ThreadRef{
				ID:    c.ID,
				Title: trimmedTitle,
			})
		}
	}

	// Also extract generic @token patterns and match against card titles / IDs
	atRegex := regexp.MustCompile(`@([^\s@#,\n]+)`)
	matches := atRegex.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		token := strings.TrimSpace(m[1])
		token = strings.TrimRight(token, ".,!?;:，。！？")
		if token == "" {
			continue
		}
		for _, c := range allCards {
			if seenIDs[c.ID] {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(c.Title), token) || c.ID == token {
				seenIDs[c.ID] = true
				threads = append(threads, ThreadRef{
					ID:    c.ID,
					Title: c.Title,
				})
				break
			}
		}
	}

	return threads
}

func (a *App) CaptureCurio(text string, images []ImagePayload) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" && len(images) == 0 {
		return "", errors.New("empty text and images")
	}

	home := a.Home
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			a.Home = h
		}
	}
	if home == "" {
		return "", errors.New("home directory not configured")
	}

	docID := ulid.Make().String()
	now := time.Now()
	cleanBody, tags := StripAndExtractHashtags(text)

	if len(images) > 0 {
		img := images[0]
		raw := strings.TrimSpace(img.Data)
		if raw != "" {
			ext := ""
			if strings.HasPrefix(raw, "data:") {
				idx := strings.Index(raw, ",")
				if idx != -1 {
					header := raw[:idx]
					raw = raw[idx+1:]
					if strings.HasPrefix(header, "data:image/") {
						mimeType := strings.TrimPrefix(header, "data:image/")
						if semi := strings.Index(mimeType, ";"); semi != -1 {
							mimeType = mimeType[:semi]
						}
						switch strings.ToLower(mimeType) {
						case "jpeg", "jpg":
							ext = ".jpg"
						case "png":
							ext = ".png"
						case "gif":
							ext = ".gif"
						case "webp":
							ext = ".webp"
						case "svg+xml", "svg":
							ext = ".svg"
						case "bmp":
							ext = ".bmp"
						case "avif":
							ext = ".avif"
						default:
							ext = "." + mimeType
						}
					}
				}
			}
			if ext == "" {
				if img.Ext != "" {
					ext = img.Ext
					if !strings.HasPrefix(ext, ".") {
						ext = "." + ext
					}
				} else if img.Name != "" {
					ext = filepath.Ext(img.Name)
				}
				if ext == "" {
					ext = ".png"
				}
			}

			cleanB64 := strings.Map(func(r rune) rune {
				if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
					return -1
				}
				return r
			}, raw)

			decoded, err := base64.StdEncoding.DecodeString(cleanB64)
			if err != nil {
				decoded, err = base64.RawStdEncoding.DecodeString(cleanB64)
				if err != nil {
					decoded, err = base64.URLEncoding.DecodeString(cleanB64)
					if err != nil {
						return "", fmt.Errorf("failed to decode base64 image: %w", err)
					}
				}
			}

			llmCfg := a.getLLMConfig()
			return curio.IngestCurioPayload(a.ctx, llmCfg, decoded, ext, cleanBody, tags)
		}
	}

	// Text-only
	title := ""
	lines := strings.Split(cleanBody, "\n")
	if len(lines) > 0 {
		title = strings.TrimSpace(lines[0])
	}
	if len(title) > 50 {
		title = title[:50]
	}

	slugSrc := sanitizeSlug(title)
	if slugSrc == "" {
		slugSrc = docID
	}
	targetDir := filepath.Join(home, "Scattering")
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), slugSrc)
	var targetPath string

	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.curio.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", docID))
	sb.WriteString("type: curio\n")
	if title != "" {
		sb.WriteString(fmt.Sprintf("title: %s\n", title))
	}
	sb.WriteString("intent: wish\n")
	sb.WriteString("availability: available\n")
	if len(tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	sb.WriteString("publish: false\n")
	sb.WriteString("---\n\n")
	if cleanBody != "" {
		sb.WriteString(cleanBody)
		sb.WriteString("\n")
	}

	err := watcher.GetWatcher().WriteFile(targetPath, []byte(sb.String()), "curio")
	if err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   targetPath,
			Title:      title,
			Type:       "curio",
			CapturedAt: now,
			Tags:       tags,
		},
	})

	return docID, nil
}

func (a *App) CaptureThought(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty text")
	}

	cleanBody, tags := StripAndExtractHashtags(text)
	kwProfile := keywords.RouteTagsToKeywords(tags)
	threads := a.ExtractThreadRefs(text)
	slugSource := deriveCaptureSlugSource(cleanBody, threads, text)
	cleanBody = StripThreadMentions(cleanBody, threads)

	home := a.Home
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			a.Home = h
		}
	}
	if home == "" {
		return "", errors.New("home directory not configured")
	}
	targetDir := filepath.Join(home, "Scattering")

	now := time.Now()
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(slugSource))
	var targetPath string

	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.epiphany.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}

	docID := ulid.Make().String()

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", docID))
	sb.WriteString("type: thought\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	if kwYAML := keywords.FormatKeywordsYAML(kwProfile); kwYAML != "" {
		sb.WriteString(kwYAML)
	}
	if len(threads) > 0 {
		sb.WriteString("threads:\n")
		for _, th := range threads {
			sb.WriteString(fmt.Sprintf("  - id: %q\n    title: %q\n", th.ID, th.Title))
		}
	}
	sb.WriteString("publish: false\n")
	sb.WriteString("---\n\n")
	sb.WriteString(cleanBody)
	sb.WriteString("\n")

	contentStr := sb.String()

	err := watcher.GetWatcher().WriteFile(targetPath, []byte(contentStr), "capture")
	if err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   targetPath,
			Text:       cleanBody,
			CapturedAt: now,
			Tags:       tags,
			Keywords:   kwProfile,
			Threads:    threads,
		},
	})

	return docID, nil
}

// CaptureNote saves a factual note or command card to Scattering without emotional anchors.
func (a *App) CaptureNote(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty text")
	}

	title, cleanText := chatlog.ExtractHashTitle(text)
	body := text
	if title != "" && strings.TrimSpace(cleanText) != "" {
		body = strings.TrimSpace(cleanText)
	}

	home := a.Home
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			a.Home = h
		}
	}
	if home == "" {
		return "", errors.New("home directory not configured")
	}
	targetDir := filepath.Join(home, "Scattering")

	now := time.Now()
	slugSource := title
	if slugSource == "" {
		slugSource = body
	}
	threads := a.ExtractThreadRefs(text)
	slugSource = deriveCaptureSlugSource(slugSource, threads, text)
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(slugSource))
	var targetPath string

	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.note.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}

	docID := ulid.Make().String()
	tags := ExtractHashtags(text)
	body = StripThreadMentions(body, threads)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", docID))
	sb.WriteString("type: note\n")
	if title != "" {
		sb.WriteString(fmt.Sprintf("title: %s\n", title))
	}
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	if len(threads) > 0 {
		sb.WriteString("threads:\n")
		for _, th := range threads {
			sb.WriteString(fmt.Sprintf("  - id: %q\n    title: %q\n", th.ID, th.Title))
		}
	}
	sb.WriteString("publish: false\n")
	sb.WriteString("---\n\n")
	sb.WriteString(body)
	sb.WriteString("\n")

	err := watcher.GetWatcher().WriteFile(targetPath, []byte(sb.String()), "capture")
	if err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   targetPath,
			Title:      title,
			Text:       body,
			Type:       "note",
			CapturedAt: now,
			Tags:       tags,
			Threads:    threads,
		},
	})

	return docID, nil
}

func (a *App) saveAssetImages(images []ImagePayload) ([]string, error) {
	if len(images) == 0 {
		return nil, nil
	}

	home := a.Home
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			a.Home = h
		}
	}
	if home == "" {
		return nil, errors.New("home directory not configured")
	}

	assetsDir := filepath.Join(home, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create assets directory: %w", err)
	}

	var savedImageNames []string
	for _, img := range images {
		raw := strings.TrimSpace(img.Data)
		if raw == "" {
			continue
		}

		ext := ""
		if strings.HasPrefix(raw, "data:") {
			idx := strings.Index(raw, ",")
			if idx != -1 {
				header := raw[:idx]
				raw = raw[idx+1:]
				if strings.HasPrefix(header, "data:image/") {
					mimeType := strings.TrimPrefix(header, "data:image/")
					if semi := strings.Index(mimeType, ";"); semi != -1 {
						mimeType = mimeType[:semi]
					}
					switch strings.ToLower(mimeType) {
					case "jpeg", "jpg":
						ext = "jpg"
					case "png":
						ext = "png"
					case "gif":
						ext = "gif"
					case "webp":
						ext = "webp"
					case "svg+xml", "svg":
						ext = "svg"
					case "bmp":
						ext = "bmp"
					case "avif":
						ext = "avif"
					default:
						ext = mimeType
					}
				}
			}
		}

		if img.Ext != "" {
			ext = strings.TrimPrefix(strings.TrimSpace(img.Ext), ".")
		} else if img.Name != "" {
			if e := filepath.Ext(img.Name); e != "" {
				ext = strings.TrimPrefix(e, ".")
			}
		}
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			ext = "png"
		}
		if ext == "jpeg" {
			ext = "jpg"
		}

		cleanB64 := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, raw)

		decoded, err := base64.StdEncoding.DecodeString(cleanB64)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(cleanB64)
			if err != nil {
				decoded, err = base64.URLEncoding.DecodeString(cleanB64)
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64 image: %w", err)
				}
			}
		}

		hashBytes := sha256.Sum256(decoded)
		hashStr := hex.EncodeToString(hashBytes[:])
		fileName := fmt.Sprintf("%d-%s.%s", time.Now().UnixMilli(), hashStr, ext)

		filePath := filepath.Join(assetsDir, fileName)
		if err := os.WriteFile(filePath, decoded, 0644); err != nil {
			return nil, fmt.Errorf("failed to save asset file: %w", err)
		}

		savedImageNames = append(savedImageNames, fileName)
		time.Sleep(time.Millisecond)
	}

	return savedImageNames, nil
}

func (a *App) CaptureThoughtWithImages(text string, images []ImagePayload) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" && len(images) == 0 {
		return "", errors.New("empty text and images")
	}

	savedImageNames, err := a.saveAssetImages(images)
	if err != nil {
		return "", err
	}

	home := a.Home
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			a.Home = h
		}
	}
	if home == "" {
		return "", errors.New("home directory not configured")
	}

	targetDir := filepath.Join(home, "Scattering")
	now := time.Now()
	cleanBody, tags := StripAndExtractHashtags(text)
	kwProfile := keywords.RouteTagsToKeywords(tags)
	threads := a.ExtractThreadRefs(text)
	slugSrc := deriveCaptureSlugSource(cleanBody, threads, "thought")
	cleanBody = StripThreadMentions(cleanBody, threads)
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(slugSrc))
	var targetPath string

	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.epiphany.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}

	docID := ulid.Make().String()

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", docID))
	sb.WriteString("type: thought\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	if kwYAML := keywords.FormatKeywordsYAML(kwProfile); kwYAML != "" {
		sb.WriteString(kwYAML)
	}
	if len(threads) > 0 {
		sb.WriteString("threads:\n")
		for _, th := range threads {
			sb.WriteString(fmt.Sprintf("  - id: %q\n    title: %q\n", th.ID, th.Title))
		}
	}
	sb.WriteString("publish: false\n")
	if len(savedImageNames) > 0 {
		sb.WriteString("images:\n")
		for _, imgName := range savedImageNames {
			sb.WriteString(fmt.Sprintf("  - %s\n", imgName))
		}
	} else {
		sb.WriteString("images: []\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(cleanBody)
	sb.WriteString("\n")

	contentStr := sb.String()

	err = watcher.GetWatcher().WriteFile(targetPath, []byte(contentStr), "capture")
	if err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   targetPath,
			Text:       cleanBody,
			CapturedAt: now,
			Images:     savedImageNames,
			Tags:       tags,
			Keywords:   kwProfile,
			Threads:    threads,
		},
	})

	return docID, nil
}

func (a *App) CaptureDiary(text string, images []ImagePayload, meta map[string]interface{}) (string, error) {
	createdAt := time.Now()
	if ts, ok := meta["timestamp"].(string); ok && ts != "" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			createdAt = parsed
		}
	} else if ca, ok := meta["createdAt"].(string); ok && ca != "" {
		if parsed, err := time.Parse(time.RFC3339, ca); err == nil {
			createdAt = parsed
		}
	}

	if createdAt.After(time.Now()) {
		return "", errors.New("cannot capture diary with future timestamp")
	}

	if meta["location"] == nil || meta["weather"] == nil {
		reg := module.GetRegistry()
		if geoMod, ok := reg["geo"]; ok {
			if ap, ok := geoMod.(module.ActionProvider); ok {
				res, err := ap.ExecuteAction(context.Background(), "resolve_context", map[string]interface{}{
					"module": "diary",
				})
				if err == nil && res != nil {
					if rMap, ok := res.(map[string]interface{}); ok {
						if meta["location"] == nil && rMap["location"] != nil {
							meta["location"] = rMap["location"]
						}
						if meta["weather"] == nil && rMap["weather"] != nil {
							meta["weather"] = rMap["weather"]
						}
					}
				}
			}
		}
	}

	id := ulid.Make().String()
	filename := fmt.Sprintf("%s-%s.diary.md", createdAt.Format("20060102-1504"), id)

	homeDir, _ := os.UserHomeDir()
	saveDir := filepath.Join(homeDir, "Scattering")
	os.MkdirAll(saveDir, 0755)
	filePath := filepath.Join(saveDir, filename)

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("id: %s\n", id))
	builder.WriteString("type: diary\n")
	builder.WriteString(fmt.Sprintf("created_at: \"%s\"\n", createdAt.Format(time.RFC3339)))

	if meta["location"] != nil {
		if lm, ok := meta["location"].(map[string]interface{}); ok {
			builder.WriteString("location:\n")
			if v, ok := lm["placeId"]; ok && v != "" { builder.WriteString(fmt.Sprintf("  place_id: %s\n", v)) }
			if v, ok := lm["name"]; ok && v != "" { builder.WriteString(fmt.Sprintf("  name: %s\n", v)) }
			if v, ok := lm["landmark"]; ok && v != "" { builder.WriteString(fmt.Sprintf("  landmark: %s\n", v)) }
			if v, ok := lm["lat"]; ok { builder.WriteString(fmt.Sprintf("  lat: %v\n", v)) }
			if v, ok := lm["lng"]; ok { builder.WriteString(fmt.Sprintf("  lng: %v\n", v)) }
		}
	}
	if meta["weather"] != nil {
		if wm, ok := meta["weather"].(map[string]interface{}); ok {
			builder.WriteString("weather:\n")
			if v, ok := wm["condition"]; ok && v != "" { builder.WriteString(fmt.Sprintf("  condition: %s\n", v)) }
			if v, ok := wm["wmoCode"]; ok { builder.WriteString(fmt.Sprintf("  wmo_code: %v\n", v)) }
			if v, ok := wm["icon"]; ok && v != "" { builder.WriteString(fmt.Sprintf("  icon: %s\n", v)) }
			if v, ok := wm["tempC"]; ok { builder.WriteString(fmt.Sprintf("  temp_c: %v\n", v)) }
			if v, ok := wm["observedHour"]; ok && v != "" { builder.WriteString(fmt.Sprintf("  observed_hour: %s\n", v)) }
			if v, ok := wm["isDay"]; ok { builder.WriteString(fmt.Sprintf("  is_day: %v\n", v)) }
		}
	}

	if len(images) > 0 {
		builder.WriteString("images:\n")
		for _, img := range images {
			builder.WriteString(fmt.Sprintf("  - %s\n", img.Name))
		}
	}

	builder.WriteString("---\n\n")
	builder.WriteString(text)

	if err := watcher.GetWatcher().WriteFile(filePath, []byte(builder.String()), "capture"); err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: id,
	})

	return id, nil
}
