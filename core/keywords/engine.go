package keywords

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/module"
	"Finnegan/core/watcher"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

// LLMClient represents a JSON-generating LLM client for keywords extraction.
type LLMClient interface {
	GenerateJSON(ctx context.Context, modelName string, prompt string, images []ImageAttachment, webContext string) (string, error)
}

// GeminiLLMClient is the default client invoking Google Gemini models.
type GeminiLLMClient struct {
	APIKey string
}

func (c *GeminiLLMClient) GenerateJSON(ctx context.Context, modelName string, prompt string, images []ImageAttachment, webContext string) (string, error) {
	if c.APIKey == "" {
		return "", errors.New("gemini api key is not configured")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(c.APIKey))
	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	if modelName == "" {
		modelName = "gemini-3.8-flash"
	}
	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	parts := []genai.Part{genai.Text(prompt)}
	for _, img := range images {
		if len(img.Data) > 0 {
			mime := img.MIME
			if mime == "" {
				mime = "image/png"
			}
			parts = append(parts, genai.ImageData(mime, img.Data))
		}
	}

	resp, err := model.GenerateContent(ctx, parts...)
	if err != nil {
		return "", fmt.Errorf("gemini generate content failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("empty response from gemini")
	}

	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		raw := string(textPart)
		raw = strings.TrimPrefix(raw, "```json\n")
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimSuffix(raw, "\n```")
		raw = strings.TrimSuffix(raw, "```")
		return strings.TrimSpace(raw), nil
	}

	return "", errors.New("unexpected gemini response part format")
}

// KeywordEngine provides adaptive 4-anchor cognitive keyword extraction.
type KeywordEngine struct {
	LLMConfig   module.LLMConfig
	Config      Config
	HomeDir     string
	client      LLMClient
	linkFetcher LinkFetcher

	processing sync.Map
}

// NewKeywordEngine creates a new KeywordEngine instance.
func NewKeywordEngine(llmCfg module.LLMConfig, cfg Config) *KeywordEngine {
	return &KeywordEngine{
		LLMConfig: llmCfg,
		Config:    cfg,
		client:    &GeminiLLMClient{APIKey: llmCfg.APIKey},
	}
}

// WithClient overrides the LLMClient (primarily for unit testing and mocking).
func (e *KeywordEngine) WithClient(client LLMClient) *KeywordEngine {
	e.client = client
	return e
}

// WithLinkFetcher overrides the LinkFetcher (useful for unit testing and mocking).
func (e *KeywordEngine) WithLinkFetcher(fetcher LinkFetcher) *KeywordEngine {
	e.linkFetcher = fetcher
	return e
}

// WithHomeDir sets the home directory for resolving asset files.
func (e *KeywordEngine) WithHomeDir(dir string) *KeywordEngine {
	e.HomeDir = dir
	return e
}

func (e *KeywordEngine) getHomeDir() string {
	if e.HomeDir != "" {
		return e.HomeDir
	}
	_, home, err := config.LoadConfig()
	if err == nil && home != "" {
		return home
	}
	uHome, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(uHome, "Documents", "Finnegan")
	}
	return ""
}

func (e *KeywordEngine) getJinaAPIKey() string {
	if e.Config.JinaAPIKey != "" {
		return e.Config.JinaAPIKey
	}
	if envKey := os.Getenv("JINA_API_KEY"); envKey != "" {
		return envKey
	}
	cfg, _, err := config.LoadConfig()
	if err == nil && cfg != nil {
		if node, exists := cfg.RawModules["bugu"]; exists {
			var buguMeta struct {
				JinaAPIKey string `yaml:"jina_api_key"`
			}
			if err := node.Decode(&buguMeta); err == nil && buguMeta.JinaAPIKey != "" {
				return buguMeta.JinaAPIKey
			}
		}
	}
	return ""
}

// BuildPrompt constructs the 4-anchor Salience Gating prompt for active dimensions.
// Optionally accepts webContext and hasImages via extra parameters.
func (e *KeywordEngine) BuildPrompt(text string, extra ...any) string {
	var webContext string
	var hasImages bool

	for _, arg := range extra {
		switch v := arg.(type) {
		case string:
			webContext = strings.TrimSpace(v)
		case bool:
			hasImages = v
		case []ImageAttachment:
			hasImages = len(v) > 0
		}
	}

	activeDefs := e.Config.GetActiveDimensions()
	var dimSb strings.Builder
	for _, def := range activeDefs {
		dimSb.WriteString(fmt.Sprintf("   - [%s] (%s): %s\n", def.ID, def.Name, def.Description))
	}

	var jsonFields strings.Builder
	for i, def := range activeDefs {
		jsonFields.WriteString(fmt.Sprintf("      \"%s\": [\"...\"]", def.ID))
		if i < len(activeDefs)-1 {
			jsonFields.WriteString(",\n")
		} else {
			jsonFields.WriteString("\n")
		}
	}

	var multimodalPrompt string
	if hasImages {
		multimodalPrompt = `
4. MULTIMODAL IMAGE ANALYSIS:
   - The card has attached images. Inspect visual scene, characters, meme elements, text in images, and atmosphere to extract concrete anchors (object, motif, mood, concept). Also provide a 1-2 sentence visual_summary.`
	}

	var linkPrompt string
	if webContext != "" {
		linkPrompt = `
5. REFERENCED WEB CONTENT ANALYSIS:
   - The card references external web content. Inspect the referenced content to ground understanding and extract concrete anchors. Also provide a 1-2 sentence link_summary.`
	}

	var webContentSection string
	if webContext != "" {
		webContentSection = fmt.Sprintf("\nREFERENCED WEB CONTENT:\n%s\n", webContext)
	}

	var schemaSummaries strings.Builder
	if hasImages {
		schemaSummaries.WriteString(",\n  \"visual_summary\": \"1-2 sentence visual summary of attached images\"")
	}
	if webContext != "" {
		schemaSummaries.WriteString(",\n  \"link_summary\": \"1-2 sentence summary of referenced web links\"")
	}

	return fmt.Sprintf(`You are an advanced cognitive extraction engine and ontology architect.
Your goal is to extract CONCRETE, COMPACT ANCHORS (实心锚点) from the given thought or dialogue card.

CRITICAL PRINCIPLES:
1. ALL KEYWORDS MUST BE "CONCRETE ANCHORS" (实心锚点):
   - Every extracted item MUST be a compact noun or discrete concept/motif phrase (2 to 5 Chinese characters, or 1 to 3 English words).
   - STRICTLY FORBIDDEN: Long explanatory sentences, rambling literary phrases, abstract psychological descriptions (e.g. NEVER output "被具体生活之美唤醒后的抽离感" or "为了提高效率而做无用功"). No term may exceed 8 characters.
2. 4 CONCRETE ANCHOR DIMENSIONS:
%s
3. SALIENCE GATING (两步合一门控):
   - Only activate dimensions that genuinely have high weight in this card.
   - If a dimension does not genuinely apply, return [] for that dimension. Strictly NEVER force or hallucinate keywords where the dimension does not clearly apply! Most cards should only activate 1 to 3 relevant dimensions.
   - For activated dimensions, return 1 to 4 crisp anchor terms.%s%s
6. Language:
   - Extract keywords in the natural language matching the card (Chinese for Chinese content, English for English content, etc.), keeping foreign proper nouns and technical terms in their native form.

CARD CONTENT:
%s%s
Output strictly valid JSON matching this structure:
{
  "keywords": {
%s  }%s
}
Only include active dimensions in keywords. If a dimension is not triggered, its value MUST be an empty array [].`, dimSb.String(), multimodalPrompt, linkPrompt, text, webContentSection, jsonFields.String(), schemaSummaries.String())
}

func (e *KeywordEngine) filterActiveProfile(raw map[string][]string) KeywordsProfile {
	activeDefs := e.Config.GetActiveDimensions()
	activeMap := make(map[string]bool, len(activeDefs))
	profile := make(KeywordsProfile, len(activeDefs))
	for _, def := range activeDefs {
		activeMap[def.ID] = true
		profile[def.ID] = []string{}
	}

	for rawKey, list := range raw {
		canonicalKey := NormalizeDimension(rawKey)
		if !activeMap[canonicalKey] {
			continue
		}
		var clean []string
		for _, item := range list {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				clean = append(clean, trimmed)
			}
		}
		if len(clean) > 0 {
			profile[canonicalKey] = append(profile[canonicalKey], clean...)
			// Also retain raw legacy key if different from canonical key for backward compatibility
			if rawKey != canonicalKey {
				profile[rawKey] = append(profile[rawKey], clean...)
			}
		}
	}
	return profile
}

// parseExtractionResponse parses the LLM output into an ExtractionResult, gracefully
// supporting the expected modern nested format as well as legacy flat formats.
func (e *KeywordEngine) parseExtractionResponse(rawJSON string) (*ExtractionResult, error) {
	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	// Format 1: Expected modern nested structure
	// { "keywords": { ... }, "visual_summary": "...", "link_summary": "..." }
	var modern struct {
		Keywords      map[string][]string `json:"keywords"`
		VisualSummary string              `json:"visual_summary"`
		LinkSummary   string              `json:"link_summary"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &modern); err == nil && modern.Keywords != nil {
		return &ExtractionResult{
			Keywords:      e.filterActiveProfile(modern.Keywords),
			VisualSummary: strings.TrimSpace(modern.VisualSummary),
			LinkSummary:   strings.TrimSpace(modern.LinkSummary),
		}, nil
	}

	// Format 2: Flexible fallback supporting top-level keywords or mixed JSON
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &rawMap); err == nil {
		var vs string
		var ls string
		if vsRaw, ok := rawMap["visual_summary"]; ok {
			_ = json.Unmarshal(vsRaw, &vs)
		}
		if lsRaw, ok := rawMap["link_summary"]; ok {
			_ = json.Unmarshal(lsRaw, &ls)
		}

		kwMap := make(map[string][]string)
		if kwRaw, ok := rawMap["keywords"]; ok {
			_ = json.Unmarshal(kwRaw, &kwMap)
		} else {
			// Dimensions are at top-level
			for k, v := range rawMap {
				if k == "visual_summary" || k == "link_summary" {
					continue
				}
				var list []string
				if err := json.Unmarshal(v, &list); err == nil {
					kwMap[k] = list
				}
			}
		}

		return &ExtractionResult{
			Keywords:      e.filterActiveProfile(kwMap),
			VisualSummary: strings.TrimSpace(vs),
			LinkSummary:   strings.TrimSpace(ls),
		}, nil
	}

	return nil, fmt.Errorf("failed to parse keywords JSON: %s", rawJSON)
}

// ExtractKeywordsWithContext extracts multidimensional keywords along with visual and link summaries.
func (e *KeywordEngine) ExtractKeywordsWithContext(ctx context.Context, text string, images []ImageAttachment, webContext string) (*ExtractionResult, error) {
	text = strings.TrimSpace(text)
	webContext = strings.TrimSpace(webContext)

	if text == "" && len(images) == 0 && webContext == "" {
		return &ExtractionResult{Keywords: make(KeywordsProfile)}, nil
	}

	activeDefs := e.Config.GetActiveDimensions()
	if len(activeDefs) == 0 {
		return &ExtractionResult{Keywords: make(KeywordsProfile)}, nil
	}

	prompt := e.BuildPrompt(text, webContext, len(images) > 0)

	modelName := "gemini-3.8-flash"
	if e.LLMConfig.DefaultModel != "" {
		modelName = e.LLMConfig.DefaultModel
	}

	if e.client == nil {
		e.client = &GeminiLLMClient{APIKey: e.LLMConfig.APIKey}
	}

	rawJSON, err := e.client.GenerateJSON(ctx, modelName, prompt, images, webContext)
	if err != nil {
		return nil, err
	}

	return e.parseExtractionResponse(rawJSON)
}

// ExtractKeywords performs salience gating and extracts multidimensional keywords for backwards compatibility.
// Accepts optional arguments (such as []ImageAttachment or webContext string).
func (e *KeywordEngine) ExtractKeywords(ctx context.Context, text string, extra ...any) (KeywordsProfile, error) {
	var images []ImageAttachment
	var webContext string

	for _, arg := range extra {
		switch v := arg.(type) {
		case []ImageAttachment:
			images = v
		case string:
			webContext = v
		}
	}

	res, err := e.ExtractKeywordsWithContext(ctx, text, images, webContext)
	if err != nil {
		return nil, err
	}
	return res.Keywords, nil
}

// ProcessCard asynchronously extracts keywords, merges them into the file's frontmatter,
// writes back via watcher with origin "keywords", and broadcasts EventKeywordsExtracted.
func (e *KeywordEngine) ProcessCard(filePath string, docID string, text string) {
	go func() {
		_ = e.ProcessCardSync(context.Background(), filePath, docID, text)
	}()
}

// ProcessCardSync runs the card processing synchronously (useful for verification and tests).
func (e *KeywordEngine) ProcessCardSync(ctx context.Context, filePath string, docID string, text string) error {
	if filePath == "" || docID == "" {
		return errors.New("empty filePath or docID")
	}

	if _, loaded := e.processing.LoadOrStore(docID, true); loaded {
		log.Printf("[KeywordEngine] Card %s is already being processed, skipping duplicate request", docID)
		return nil
	}
	defer e.processing.Delete(docID)

	ctxTimeout, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 1. Read card markdown file from disk
	var rawBytes []byte
	var readErr error
	for i := 0; i < 5; i++ {
		rawBytes, readErr = os.ReadFile(filePath)
		if readErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if readErr != nil {
		log.Printf("[KeywordEngine] Failed to read markdown file %s: %v", filePath, readErr)
		return readErr
	}

	// 2. Parse frontmatter for attached images and fallback body text
	var meta struct {
		Images []string `yaml:"images"`
	}
	var cardBody string
	strContent := string(rawBytes)
	if strings.HasPrefix(strContent, "---") {
		parts := strings.SplitN(strContent, "---", 3)
		if len(parts) >= 3 {
			_ = yaml.Unmarshal([]byte(parts[1]), &meta)
			cardBody = strings.TrimSpace(parts[2])
		}
	}
	if text == "" && cardBody != "" {
		text = cardBody
	}

	// 3. Resolve and load attached images from <home>/assets/<image_filename>
	homeDir := e.getHomeDir()
	assetsDir := filepath.Join(homeDir, "assets")
	var imageAttachments []ImageAttachment
	for _, imgName := range meta.Images {
		imgName = strings.TrimSpace(imgName)
		if imgName == "" {
			continue
		}
		var imgPath string
		if filepath.IsAbs(imgName) {
			imgPath = imgName
		} else {
			imgPath = filepath.Join(assetsDir, imgName)
		}
		imgBytes, err := os.ReadFile(imgPath)
		if err != nil {
			log.Printf("[KeywordEngine] Warning: failed to read image attachment %s: %v", imgPath, err)
			continue
		}
		mime := detectMIME(imgName, imgBytes)
		imageAttachments = append(imageAttachments, ImageAttachment{
			MIME: mime,
			Data: imgBytes,
		})
	}

	// 4. Sniff URLs and fetch web context
	urls := SniffURLs(text)
	var webContext string
	if len(urls) > 0 {
		fetcher := e.linkFetcher
		if fetcher == nil {
			fetcher = NewDefaultLinkFetcher(e.getJinaAPIKey())
		}
		webContext = fetcher.FetchAll(ctxTimeout, urls)
	}

	// 5. Extract keywords, visual summary, and link summary
	res, err := e.ExtractKeywordsWithContext(ctxTimeout, text, imageAttachments, webContext)
	if err != nil {
		log.Printf("[KeywordEngine] ExtractKeywordsWithContext failed for card %s (%s): %v", docID, filePath, err)
		return err
	}

	// 6. Merge keywords, visual_summary, and link_summary into markdown frontmatter
	updatedBytes, err := MergeKeywordsToMarkdown(rawBytes, res.Keywords, res.VisualSummary, res.LinkSummary)
	if err != nil {
		log.Printf("[KeywordEngine] Failed to merge keywords to markdown %s: %v", filePath, err)
		return err
	}

	// 7. Write back updated markdown file via Watcher
	err = watcher.GetWatcher().WriteFile(filePath, updatedBytes, "keywords")
	if err != nil {
		log.Printf("[KeywordEngine] Failed to write updated markdown %s: %v", filePath, err)
		return err
	}

	finalProfile, err := ExtractKeywordsFromMarkdown(updatedBytes)
	if err != nil {
		finalProfile = CleanProfile(res.Keywords)
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventKeywordsExtracted,
		Data: bus.EventKeywordsExtractedData{
			DocID:    docID,
			FilePath: filePath,
			Text:     text,
			Keywords: finalProfile,
		},
	})

	return nil
}

// detectMIME determines the appropriate image MIME type for a file.
func detectMIME(fileName string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		ct := http.DetectContentType(data)
		if strings.HasPrefix(ct, "image/") {
			return ct
		}
		return "image/png"
	}
}
