package keywords

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/module"
	"Finnegan/core/watcher"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// LLMClient represents a JSON-generating LLM client for keywords extraction.
type LLMClient interface {
	GenerateJSON(ctx context.Context, modelName string, prompt string) (string, error)
}

// GeminiLLMClient is the default client invoking Google Gemini models.
type GeminiLLMClient struct {
	APIKey string
}

func (c *GeminiLLMClient) GenerateJSON(ctx context.Context, modelName string, prompt string) (string, error) {
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

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
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
	LLMConfig module.LLMConfig
	Config    Config
	client    LLMClient

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

// BuildPrompt constructs the 4-anchor Salience Gating prompt for active dimensions.
func (e *KeywordEngine) BuildPrompt(text string) string {
	activeDefs := e.Config.GetActiveDimensions()
	var dimSb strings.Builder
	for _, def := range activeDefs {
		dimSb.WriteString(fmt.Sprintf("   - [%s] (%s): %s\n", def.ID, def.Name, def.Description))
	}

	var jsonFields strings.Builder
	for i, def := range activeDefs {
		jsonFields.WriteString(fmt.Sprintf("  \"%s\": [\"...\"]", def.ID))
		if i < len(activeDefs)-1 {
			jsonFields.WriteString(",\n")
		} else {
			jsonFields.WriteString("\n")
		}
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
   - For activated dimensions, return 1 to 4 crisp anchor terms.
4. Language:
   - Extract keywords in the natural language matching the card (Chinese for Chinese content, English for English content, etc.), keeping foreign proper nouns and technical terms in their native form.

CARD CONTENT:
%s

Output strictly valid JSON matching this structure:
{
%s}
Only include active dimensions. If a dimension is not triggered, its value MUST be an empty array [].`, dimSb.String(), text, jsonFields.String())
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

// ExtractKeywords performs salience gating and extracts multidimensional keywords for the text.
func (e *KeywordEngine) ExtractKeywords(ctx context.Context, text string) (KeywordsProfile, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return make(KeywordsProfile), nil
	}

	activeDefs := e.Config.GetActiveDimensions()
	if len(activeDefs) == 0 {
		return make(KeywordsProfile), nil
	}

	prompt := e.BuildPrompt(text)

	modelName := "gemini-3.8-flash"
	if e.LLMConfig.DefaultModel != "" {
		modelName = e.LLMConfig.DefaultModel
	}

	if e.client == nil {
		e.client = &GeminiLLMClient{APIKey: e.LLMConfig.APIKey}
	}

	rawJSON, err := e.client.GenerateJSON(ctx, modelName, prompt)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Keywords map[string][]string `json:"keywords"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &wrapper); err == nil && wrapper.Keywords != nil {
		return e.filterActiveProfile(wrapper.Keywords), nil
	}

	var direct map[string][]string
	if err := json.Unmarshal([]byte(rawJSON), &direct); err == nil && direct != nil {
		return e.filterActiveProfile(direct), nil
	}

	return nil, fmt.Errorf("failed to parse keywords JSON: %s", rawJSON)
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

	profile, err := e.ExtractKeywords(ctxTimeout, text)
	if err != nil {
		log.Printf("[KeywordEngine] ExtractKeywords failed for card %s (%s): %v", docID, filePath, err)
		return err
	}

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

	updatedBytes, err := MergeKeywordsToMarkdown(rawBytes, profile)
	if err != nil {
		log.Printf("[KeywordEngine] Failed to merge keywords to markdown %s: %v", filePath, err)
		return err
	}

	err = watcher.GetWatcher().WriteFile(filePath, updatedBytes, "keywords")
	if err != nil {
		log.Printf("[KeywordEngine] Failed to write updated markdown %s: %v", filePath, err)
		return err
	}

	cleanProfile := CleanProfile(profile)

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventKeywordsExtracted,
		Data: bus.EventKeywordsExtractedData{
			DocID:    docID,
			FilePath: filePath,
			Text:     text,
			Keywords: cleanProfile,
		},
	})

	return nil
}
