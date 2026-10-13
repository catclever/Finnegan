package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/models"
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
1. Select up to 10 highly relevant target IDs from the catalog that match the intent.
2. Formulate 1 or 2 specific search queries for EACH target's language. Use the exact native language of the target (e.g. Russian for ru, French for fr).
3. Do NOT include site: operators in the queries; the system will append them automatically.
4. Output strict JSON matching this exact structure:
{
  "selected_target_ids": ["id1", "id2"],
  "search_queries": {
    "ru": ["запрос 1", "запрос 2"],
    "en": ["query 1"]
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
