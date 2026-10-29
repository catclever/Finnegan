package curio

import (
	"context"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/module"
	"Finnegan/core/watcher"

	"github.com/google/generative-ai-go/genai"
	"github.com/oklog/ulid/v2"
	"google.golang.org/api/option"
)

type CurioMetadata struct {
	Title        string   `json:"title"`
	Brand        string   `json:"brand"`
	Price        string   `json:"price"`
	Platform     string   `json:"platform"`
	Intent       string   `json:"intent"`
	Availability string   `json:"availability"`
	Tags         []string `json:"tags"`
}

func sanitizeSlug(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ToLower(s)
	if len(s) > 30 {
		s = s[:30]
	}
	return s
}

// IngestScreenshot takes an image path, note, and tags, analyzes it using Gemini Vision,
// extracts CurioMetadata, simulates hero cropping, saves assets, and creates a .curio.md.
func IngestScreenshot(ctx context.Context, llmCfg module.LLMConfig, imgPath string, note string, extraTags []string) error {
	_, home, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Read image
	imgData, err := os.ReadFile(imgPath)
	if err != nil {
		return fmt.Errorf("failed to read screenshot: %w", err)
	}
	imgExt := strings.ToLower(filepath.Ext(imgPath))
	mimeType := "image/jpeg"
	if imgExt == ".png" {
		mimeType = "image/png"
	}

	// Use Gemini Vision
	if llmCfg.APIKey == "" {
		return fmt.Errorf("gemini api key is required for curio ingestion")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(llmCfg.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	modelName := llmCfg.DefaultModel
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
- "intent": the user's intent (e.g., buy, sell, research, wishlist)
- "availability": product availability (e.g., in stock, sold out, pre-order, unknown)
- "tags": an array of string tags describing the aesthetic, category, or features.`

	if note != "" {
		prompt += fmt.Sprintf("\nUser provided note: %s", note)
	}

	imgBlob := genai.ImageData(mimeType, imgData)
	resp, err := model.GenerateContent(ctx, imgBlob, genai.Text(prompt))
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	var meta CurioMetadata
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if txtPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			err = json.Unmarshal([]byte(txtPart), &meta)
			if err != nil {
				log.Printf("Curio Ingest warning: could not unmarshal json %v", err)
			}
		}
	}

	// Combine extra tags
	tagSet := make(map[string]bool)
	for _, t := range meta.Tags {
		if t != "" {
			tagSet[t] = true
		}
	}
	for _, t := range extraTags {
		if t != "" {
			tagSet[t] = true
		}
	}
	finalTags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		finalTags = append(finalTags, t)
	}

	// Save images
	now := time.Now()
	timestampStr := now.Format("20060102-150405")
	docID := ulid.Make().String()

	archivesDir := filepath.Join(home, "archives")
	os.MkdirAll(archivesDir, 0755)
	snapshotName := fmt.Sprintf("%s-screenshot-%s%s", timestampStr, docID, imgExt)
	snapshotPath := filepath.Join(archivesDir, snapshotName)
	if err := os.WriteFile(snapshotPath, imgData, 0644); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Simulate hero crop by just copying for now
	assetsDir := filepath.Join(home, "assets", "curio")
	os.MkdirAll(assetsDir, 0755)
	heroName := fmt.Sprintf("%s-hero-%s%s", timestampStr, docID, imgExt)
	heroPath := filepath.Join(assetsDir, heroName)
	if err := os.WriteFile(heroPath, imgData, 0644); err != nil {
		return fmt.Errorf("failed to save hero image: %w", err)
	}

	// Generate markdown
	slug := sanitizeSlug(meta.Title)
	if slug == "" {
		slug = docID
	}
	scatteringDir := filepath.Join(home, "Scattering")
	os.MkdirAll(scatteringDir, 0755)
	cardFilename := fmt.Sprintf("%s-%s.curio.md", now.Format("20060102-1504"), slug)
	cardPath := filepath.Join(scatteringDir, cardFilename)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", docID))
	sb.WriteString("type: curio\n")
	sb.WriteString(fmt.Sprintf("title: %s\n", meta.Title))
	sb.WriteString(fmt.Sprintf("brand: %s\n", meta.Brand))
	sb.WriteString(fmt.Sprintf("platform: %s\n", meta.Platform))
	sb.WriteString(fmt.Sprintf("price: %s\n", meta.Price))
	sb.WriteString(fmt.Sprintf("intent: %s\n", meta.Intent))
	sb.WriteString(fmt.Sprintf("availability: %s\n", meta.Availability))
	if len(finalTags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range finalTags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	sb.WriteString("images:\n")
	sb.WriteString(fmt.Sprintf("  - curio/%s\n", heroName)) // assume relative to assets
	sb.WriteString(fmt.Sprintf("snapshotPath: %s\n", snapshotPath))
	sb.WriteString(fmt.Sprintf("screenshotPath: %s\n", heroPath))
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	sb.WriteString("publish: false\n")
	sb.WriteString("---\n\n")
	if note != "" {
		sb.WriteString(note)
		sb.WriteString("\n")
	}

	err = watcher.GetWatcher().WriteFile(cardPath, []byte(sb.String()), "curio")
	if err != nil {
		return fmt.Errorf("failed to write card: %w", err)
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   cardPath,
			Title:      meta.Title,
			Type:       "curio",
			CapturedAt: now,
			Tags:       finalTags,
			Images:     []string{fmt.Sprintf("curio/%s", heroName)},
		},
	})

	return nil
}
