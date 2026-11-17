package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/oklog/ulid/v2"
	"google.golang.org/api/option"
)

type llmConfig struct {
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	DefaultModel string `json:"default_model"`
}

type curioMetadata struct {
	Title        string   `json:"title"`
	Brand        string   `json:"brand"`
	Price        string   `json:"price"`
	Platform     string   `json:"platform"`
	Intent       string   `json:"intent"`
	Availability string   `json:"availability"`
	Tags         []string `json:"tags"`
}

type captureImageInput struct {
	Data string `json:"data"`
	Name string `json:"name,omitempty"`
	Ext  string `json:"ext,omitempty"`
}

type captureRequest struct {
	CardType  string                 `json:"cardType"`
	Text      string                 `json:"text"`
	CleanText string                 `json:"cleanText,omitempty"`
	Images    []captureImageInput    `json:"images,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	ExtraMeta map[string]interface{} `json:"extraMeta,omitempty"`
	HomeDir   string                 `json:"homeDir,omitempty"`
}

type captureResult struct {
	DocID    string                 `json:"docId,omitempty"`
	FilePath string                 `json:"filePath,omitempty"`
	FileExt  string                 `json:"fileExt,omitempty"`
	Slug     string                 `json:"slug,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Body     string                 `json:"body,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Images   []string               `json:"images,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

type decodedImage struct {
	Data     []byte
	Ext      string
	MimeType string
}

func sanitizeSlug(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "-")
	s = strings.ReplaceAll(s, "\t", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ToLower(strings.Trim(s, "._-@ "))
	runes := []rune(s)
	if len(runes) > 30 {
		s = strings.TrimRight(string(runes[:30]), "._-@ ")
	}
	return s
}

func deriveTitleFromText(text string, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		runes := []rune(trimmed)
		if len(runes) > 50 {
			return string(runes[:50])
		}
		return trimmed
	}
	return fallback
}

func resolveHomeDir(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	return strings.TrimSpace(os.Getenv("FINNEGAN_HOME"))
}

func resolveGeminiLLM(llms map[string]llmConfig) llmConfig {
	if llms != nil {
		if def, ok := llms["default"]; ok && strings.TrimSpace(def.APIKey) != "" {
			if def.Provider == "" || strings.EqualFold(def.Provider, "gemini") {
				return def
			}
		}
		for _, cfg := range llms {
			if strings.EqualFold(cfg.Provider, "gemini") && strings.TrimSpace(cfg.APIKey) != "" {
				return cfg
			}
		}
	}
	return llmConfig{
		Provider:     "gemini",
		APIKey:       strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		DefaultModel: "gemini-3.8-flash",
	}
}

func extToMimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".avif":
		return "image/avif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "image/png"
	}
}

func decodeCaptureImage(img captureImageInput) (decodedImage, error) {
	raw := strings.TrimSpace(img.Data)
	if raw == "" {
		return decodedImage{}, errors.New("empty image payload")
	}

	ext := ""
	if strings.HasPrefix(raw, "data:") {
		if idx := strings.Index(raw, ","); idx != -1 {
			header := raw[:idx]
			raw = raw[idx+1:]
			if strings.HasPrefix(header, "data:image/") {
				mimeSub := strings.TrimPrefix(header, "data:image/")
				if semi := strings.Index(mimeSub, ";"); semi != -1 {
					mimeSub = mimeSub[:semi]
				}
				switch strings.ToLower(mimeSub) {
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
					if mimeSub != "" {
						ext = "." + mimeSub
					}
				}
			}
		}
	}

	if ext == "" {
		if strings.TrimSpace(img.Ext) != "" {
			ext = strings.TrimSpace(img.Ext)
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
		} else if strings.TrimSpace(img.Name) != "" {
			ext = filepath.Ext(strings.TrimSpace(img.Name))
		}
	}
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		ext = ".png"
	}
	if ext == ".jpeg" {
		ext = ".jpg"
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
				return decodedImage{}, fmt.Errorf("failed to decode base64 image: %w", err)
			}
		}
	}

	return decodedImage{
		Data:     decoded,
		Ext:      ext,
		MimeType: extToMimeType(ext),
	}, nil
}

func analyzeWithGeminiVision(ctx context.Context, llmCfg llmConfig, img decodedImage, note string) (*curioMetadata, error) {
	if strings.TrimSpace(llmCfg.APIKey) == "" {
		return nil, errors.New("no gemini api key configured")
	}

	visionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(visionCtx, option.WithAPIKey(llmCfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	modelName := strings.TrimSpace(llmCfg.DefaultModel)
	if modelName == "" {
		modelName = "gemini-3.8-flash"
	}
	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	prompt := `Analyze this product screenshot. Extract the following fields as JSON:
- "title": the product name or listing title
- "brand": the brand of the product (if visible, otherwise empty)
- "price": estimated price or listed price (if visible, otherwise empty)
- "platform": the platform (e.g. 闲鱼, Mercari, Taobao, eBay - guess based on UI if possible, otherwise empty)
- "intent": the user's intent (one of: wish, someday, archive, acquired; default wish)
- "availability": product availability (one of: available, deadstock, delisted; default available)
- "tags": an array of string tags describing the aesthetic, category, or features.`

	if strings.TrimSpace(note) != "" {
		prompt += fmt.Sprintf("\nUser provided note: %s", note)
	}

	resp, err := model.GenerateContent(visionCtx, genai.ImageData(img.MimeType, img.Data), genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty vision response")
	}

	txtPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, errors.New("unexpected non-text vision response")
	}

	var meta curioMetadata
	if err := json.Unmarshal([]byte(txtPart), &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vision json: %w", err)
	}
	return &meta, nil
}

func mergeTags(base []string, extra []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, list := range [][]string{base, extra} {
		for _, t := range list {
			clean := strings.TrimSpace(strings.TrimPrefix(t, "#"))
			if clean != "" && !seen[clean] {
				seen[clean] = true
				out = append(out, clean)
			}
		}
	}
	return out
}

func ingestCurioImages(ctx context.Context, homeDir string, llms map[string]llmConfig, imgs []decodedImage, cleanText string, baseTags []string) (*captureResult, error) {
	if len(imgs) == 0 {
		return nil, errors.New("no valid images provided")
	}
	if homeDir == "" {
		return nil, errors.New("homeDir or FINNEGAN_HOME is required for image capture")
	}

	var extracted *curioMetadata
	llmCfg := resolveGeminiLLM(llms)
	if strings.TrimSpace(llmCfg.APIKey) != "" {
		meta, err := analyzeWithGeminiVision(ctx, llmCfg, imgs[0], cleanText)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[curio.plugin] vision fallback (offline or API error): %v\n", err)
		} else {
			extracted = meta
		}
	}

	var finalMeta curioMetadata
	if extracted != nil {
		finalMeta = *extracted
	}
	if strings.TrimSpace(finalMeta.Title) == "" {
		finalMeta.Title = deriveTitleFromText(cleanText, "Curio Specimen")
	}
	if strings.TrimSpace(finalMeta.Intent) == "" {
		finalMeta.Intent = "wish"
	}
	if strings.TrimSpace(finalMeta.Availability) == "" {
		finalMeta.Availability = "available"
	}

	mergedTags := mergeTags(baseTags, finalMeta.Tags)

	now := time.Now()
	timestampStr := now.Format("20060102-150405")
	docID := ulid.Make().String()

	archivesDir := filepath.Join(homeDir, "archives")
	if err := os.MkdirAll(archivesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archives dir: %w", err)
	}
	snapshotName := fmt.Sprintf("%s-screenshot-%s%s", timestampStr, docID, imgs[0].Ext)
	snapshotPath := filepath.Join(archivesDir, snapshotName)
	if err := os.WriteFile(snapshotPath, imgs[0].Data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	assetsDir := filepath.Join(homeDir, "assets", "curio")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create assets/curio dir: %w", err)
	}

	var savedRelImages []string
	var primaryHeroPath string
	for i, img := range imgs {
		var heroName string
		if i == 0 {
			heroName = fmt.Sprintf("%s-hero-%s%s", timestampStr, docID, img.Ext)
		} else {
			heroName = fmt.Sprintf("%s-hero-%s-%d%s", timestampStr, docID, i+1, img.Ext)
		}
		heroPath := filepath.Join(assetsDir, heroName)
		if err := os.WriteFile(heroPath, img.Data, 0644); err != nil {
			return nil, fmt.Errorf("failed to save hero image: %w", err)
		}
		if i == 0 {
			primaryHeroPath = heroPath
		}
		savedRelImages = append(savedRelImages, "curio/"+heroName)
	}

	slug := sanitizeSlug(finalMeta.Title)
	if slug == "" {
		slug = docID
	}

	metaMap := map[string]interface{}{
		"intent":         finalMeta.Intent,
		"availability":   finalMeta.Availability,
		"snapshotPath":   snapshotPath,
		"screenshotPath": primaryHeroPath,
	}
	if strings.TrimSpace(finalMeta.Brand) != "" {
		metaMap["brand"] = strings.TrimSpace(finalMeta.Brand)
	}
	if strings.TrimSpace(finalMeta.Platform) != "" {
		metaMap["platform"] = strings.TrimSpace(finalMeta.Platform)
	}
	if strings.TrimSpace(finalMeta.Price) != "" {
		metaMap["price"] = strings.TrimSpace(finalMeta.Price)
	}

	return &captureResult{
		DocID:   docID,
		FileExt: ".curio.md",
		Slug:    slug,
		Title:   finalMeta.Title,
		Body:    cleanText,
		Tags:    mergedTags,
		Images:  savedRelImages,
		Meta:    metaMap,
	}, nil
}

func captureTextOnly(cleanText string, tags []string) (*captureResult, error) {
	docID := ulid.Make().String()
	title := deriveTitleFromText(cleanText, "")
	slug := sanitizeSlug(title)
	if slug == "" {
		slug = docID
	}

	return &captureResult{
		DocID:   docID,
		FileExt: ".curio.md",
		Slug:    slug,
		Title:   title,
		Body:    cleanText,
		Tags:    mergeTags(tags, nil),
		Meta: map[string]interface{}{
			"intent":       "wish",
			"availability": "available",
		},
	}, nil
}

func writeCurioMarkdownFile(homeDir string, res *captureResult) (string, error) {
	scatteringDir := filepath.Join(homeDir, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create Scattering dir: %w", err)
	}

	now := time.Now()
	slug := res.Slug
	if slug == "" {
		slug = res.DocID
	}
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), slug)
	var cardPath string
	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(scatteringDir, fmt.Sprintf("%s%s.curio.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			cardPath = candidate
			break
		}
	}
	if cardPath == "" {
		return "", errors.New("could not find unique curio markdown filename")
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", res.DocID))
	sb.WriteString("type: curio\n")
	if res.Title != "" {
		sb.WriteString(fmt.Sprintf("title: %q\n", res.Title))
	}
	for _, k := range []string{"brand", "platform", "price", "intent", "availability", "snapshotPath", "screenshotPath"} {
		if v, ok := res.Meta[k].(string); ok && v != "" {
			sb.WriteString(fmt.Sprintf("%s: %q\n", k, v))
		}
	}
	if len(res.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range res.Tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if len(res.Images) > 0 {
		sb.WriteString("images:\n")
		for _, img := range res.Images {
			sb.WriteString(fmt.Sprintf("  - %s\n", img))
		}
	}
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	sb.WriteString("publish: false\n")
	sb.WriteString("---\n\n")
	if strings.TrimSpace(res.Body) != "" {
		sb.WriteString(strings.TrimSpace(res.Body))
		sb.WriteString("\n")
	}

	if err := os.WriteFile(cardPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write curio markdown: %w", err)
	}
	return cardPath, nil
}
