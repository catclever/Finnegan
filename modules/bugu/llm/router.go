package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/models"
	"Finnegan/core/chatlog"
	"Finnegan/core/module"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Router struct {
	Config *config.Config
	LLM    module.LLMConfig
}

func NewRouter(cfg *config.Config, llmCfg module.LLMConfig) *Router {
	return &Router{
		Config: cfg,
		LLM:    llmCfg,
	}
}

func (r *Router) GenerateSearchPlan(intent string) (*models.LLMSearchPlan, error) {
	if r.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM API Key is not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(r.LLM.APIKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	modelName := "gemini-3.8-flash"
	if r.LLM.DefaultModel != "" {
		modelName = r.LLM.DefaultModel
	}
	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"
	// Ensure we parse to standard output

	catalog := r.Config.GetTargetCatalog()

	prompt := fmt.Sprintf(`You are an elite cyber-ethnographer and search architect.
Your goal is to translate the user's intent into a highly targeted multilingual search plan.
You have access to a curated catalog of global web targets.

USER INTENT: %s

TARGET CATALOG:
%s

INSTRUCTIONS:
1. Select 8 to 10 highly relevant target IDs from the catalog that match the intent.
   - Cross-Language Diversity is MANDATORY: Select a balanced mix of targets across multiple global languages (e.g. at least 2-3 English targets, 1-2 Chinese targets, and 2-4 foreign/minor language targets like ru, fr, de, ja, es if matching targets exist).
2. Formulate 1 or 2 high-precision, natural search queries for EACH selected target's language code:
   - For EVERY selected target, you MUST provide search queries in its primary language code (e.g. if you select a Russian target, you MUST provide "ru": ["..."] in native Russian; for French, "fr": ["..."] in native French).
   - ALWAYS provide "en" queries as universal baseline.
   - Use standard ISO 639-1 language codes as keys: "en", "zh", "ru", "fr", "de", "ja", "es", "it", etc.
3. Do NOT include site: operators in the queries; the system will append them automatically.
4. Output strict JSON matching this exact structure:
{
  "selected_target_ids": ["id1", "id2"],
  "search_queries": {
    "ru": ["запрос 1", "запрос 2"],
    "en": ["query 1"],
    "zh": ["查询 1"]
  }
}`, intent, catalog)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	var rawJSON string
	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		rawJSON = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected LLM response format")
	}

	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	var plan models.LLMSearchPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nRaw string: %s", err, rawJSON)
	}

	return &plan, nil
}

// TranslateTags uses Gemini to translate a list of tags into 8 global languages for the i18n target catalog.
func (r *Router) TranslateTags(tags []string) (map[string]map[string]string, error) {
	if len(tags) == 0 {
		return make(map[string]map[string]string), nil
	}
	if r.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM API Key is not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(r.LLM.APIKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	modelName := "gemini-3.8-flash"
	if r.LLM.DefaultModel != "" {
		modelName = r.LLM.DefaultModel
	}
	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	tagsList := strings.Join(tags, "\n- ")

	prompt := fmt.Sprintf(`You are an expert multilingual lexicographer and translator.
Translate the following list of domain/category tags from Chinese into 8 target languages:
"en" (English), "ru" (Russian), "de" (German), "fr" (French), "es" (Spanish), "ja" (Japanese), "it" (Italian), "ko" (Korean).

Ensure translations are concise, standard web category and community tags.

TAGS TO TRANSLATE:
- %s

Output strict JSON in this exact structure:
{
  "tag_chinese_1": {
    "en": "...",
    "ru": "...",
    "de": "...",
    "fr": "...",
    "es": "...",
    "ja": "...",
    "it": "...",
    "ko": "..."
  }
}`, tagsList)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM tag translator")
	}

	var rawJSON string
	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		rawJSON = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected LLM response format")
	}

	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	var result map[string]map[string]string
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse tag translation JSON: %w\nRaw string: %s", err, rawJSON)
	}

	return result, nil
}

// DeriveChatLogTitle extracts a concise, vivid title and participants from chat dialogue.
func (r *Router) DeriveChatLogTitle(dialogue string, fallbackParticipants []string) (*chatlog.TitleResult, error) {
	ctx := context.Background()
	return chatlog.DeriveTitle(ctx, r.LLM, dialogue, fallbackParticipants)
}
