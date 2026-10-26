package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"Finnegan/core/chatlog"
	"Finnegan/core/module"
	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/models"
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

// FormatKeywordsProfile formats a structured keywords profile into a deterministic bulleted list.
func FormatKeywordsProfile(keywords map[string][]string) string {
	if len(keywords) == 0 {
		return ""
	}
	keys := make([]string, 0, len(keywords))
	for k, list := range keywords {
		if len(list) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", k, strings.Join(keywords[k], ", ")))
	}
	return sb.String()
}

// BuildFacetSearchPlanPrompt builds the comprehensive LLM prompt for search plan routing.
func BuildFacetSearchPlanPrompt(intent string, catalog string, keywords map[string][]string) string {
	kwProfile := FormatKeywordsProfile(keywords)

	var facetSection string
	if kwProfile != "" {
		facetSection = fmt.Sprintf(`
KEYWORDS PROFILE:
%s
DIMENSION-BASED FACET MATCHING & TIME RANGE STRATEGY:
When a Keywords Profile is present, match target sites and time_range values based on the 4 concrete anchor dimension facets:
- object (名物与通用指涉): Match technical documentation, code repositories, vertical encyclopedias, and specialized domain sites. Set time_range to "year".
- motif (处境与母题): Prioritize real-time discussion platforms and forums (HN, Reddit, V2EX, GitHub Issues/Discussions). Set time_range to "month" or "week" to capture fresh and active community discourse.
- mood (心境与心绪): Match personal indie blogs, IndieWeb, literary essays, humanities platforms, and cultural forums. Set time_range to "" (unlimited/timeless, cross-temporal resonance).
- concept (观念与议题): Match long-form analytical blogs, Substack, LessWrong, and editorial essays. Set time_range to "" or "year".
`, kwProfile)
	}

	return fmt.Sprintf(`You are an elite cyber-ethnographer and search architect.
Your goal is to translate the user's intent into a highly targeted multilingual search plan.
You have access to a curated catalog of global web targets.

CURRENT YEAR: 2026 (Ensure all queries, temporal reasoning, and freshness expectations are grounded in 2026)

USER INTENT: %s
%s
TARGET CATALOG:
%s

INSTRUCTIONS:
1. Select 8 to 10 highly relevant target IDs from the catalog that match the intent and facet characteristics.
   - Cross-Language Diversity is MANDATORY: Select a balanced mix of targets across multiple global languages (e.g. at least 2-3 English targets, 1-2 Chinese targets, and 2-4 foreign/minor language targets like ru, fr, de, ja, es if matching targets exist).
2. Formulate 1 or 2 high-precision, natural search queries for EACH selected target's language code:
   - For EVERY selected target, you MUST provide search queries in its primary language code (e.g. if you select a Russian target, you MUST provide "ru": ["..."] in native Russian; for French, "fr": ["..."] in native French).
   - ALWAYS provide "en" queries as universal baseline.
   - Use standard ISO 639-1 language codes as keys: "en", "zh", "ru", "fr", "de", "ja", "es", "it", etc.
3. Determine appropriate time_range values:
   - SearxNG native time_range supports "day", "week", "month", "year", or "" (empty string for unlimited/all).
   - Set "default_time_range" as the baseline time range for this search session.
   - Set "target_time_ranges" mapping each selected target ID to its specific time_range based on dimension characteristics (e.g., "month" or "week" for real-time forums, "year" for tech docs, "" for deep essays/timeless thoughts).
4. Do NOT include site: operators in the queries; the system will append them automatically.
5. Output strict JSON matching this exact structure:
{
  "selected_target_ids": ["id1", "id2"],
  "search_queries": {
    "ru": ["запрос 1", "запрос 2"],
    "en": ["query 1"],
    "zh": ["查询 1"]
  },
  "default_time_range": "year",
  "target_time_ranges": {
    "id1": "month",
    "id2": "year"
  }
}`, intent, facetSection, catalog)
}

// ParseSearchPlanJSON parses and validates the LLM JSON response into an LLMSearchPlan.
func ParseSearchPlanJSON(rawJSON string) (*models.LLMSearchPlan, error) {
	cleaned := strings.TrimPrefix(rawJSON, "```json\n")
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var plan models.LLMSearchPlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nRaw string: %s", err, rawJSON)
	}

	return &plan, nil
}

// GenerateSearchPlan provides backward compatibility for intent-only search routing.
func (r *Router) GenerateSearchPlan(intent string) (*models.LLMSearchPlan, error) {
	return r.GenerateFacetSearchPlan(intent, nil)
}

// GenerateFacetSearchPlan generates a search plan with structured keywords and facet routing.
func (r *Router) GenerateFacetSearchPlan(intent string, keywords map[string][]string) (*models.LLMSearchPlan, error) {
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

	catalog := r.Config.GetTargetCatalog()
	prompt := BuildFacetSearchPlanPrompt(intent, catalog, keywords)

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

	return ParseSearchPlanJSON(rawJSON)
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
