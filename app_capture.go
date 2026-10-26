package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/chatlog"
	"Finnegan/core/config"
	"Finnegan/core/watcher"

	"github.com/oklog/ulid/v2"
)

func sanitizeSlug(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "_")
	s = strings.ReplaceAll(s, "\t", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.Trim(s, "._ ")
	if len([]rune(s)) > 20 {
		runes := []rune(s)
		return strings.TrimRight(string(runes[:20]), "._ ")
	}
	return s
}

func (a *App) CaptureThought(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty text")
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
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(text))
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

	contentStr := fmt.Sprintf("---\nid: %s\ntype: thought\ncreated: %s\nupdated: %s\nsource: local\ntags: []\npublish: false\n---\n\n%s\n",
		docID, now.Format(time.RFC3339), now.Format(time.RFC3339), text)

	err := watcher.GetWatcher().WriteFile(targetPath, []byte(contentStr), "capture")
	if err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   targetPath,
			Text:       text,
			CapturedAt: now,
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
	slugSrc := text
	if slugSrc == "" {
		slugSrc = "thought"
	}
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
	sb.WriteString("tags: []\n")
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
	sb.WriteString(text)
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
			Text:       text,
			CapturedAt: now,
			Images:     savedImageNames,
		},
	})

	return docID, nil
}
