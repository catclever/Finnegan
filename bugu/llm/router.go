package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"Finnegan/bugu/config"
	"Finnegan/bugu/models"
)

type Router struct {
	Config *config.Config
	APIKey string
}

func NewRouter(cfg *config.Config) *Router {
	return &Router{
		Config: cfg,
		APIKey: os.Getenv("GEMINI_API_KEY"),
	}
}

func (r *Router) GenerateSearchPlan(intent string) (*models.LLMSearchPlan, error) {
	if r.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	catalog := r.Config.GetTargetCatalog()

	systemPrompt := `You are an elite cyber-ethnographer and search architect.
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
}`

	prompt := fmt.Sprintf(systemPrompt, intent, catalog)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json",
			"thinkingConfig": map[string]interface{}{
				"thinkingLevel": "low",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.8-flash:generateContent?key=" + r.APIKey
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LLM API error: %s", string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	rawJSON := geminiResp.Candidates[0].Content.Parts[0].Text
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	var plan models.LLMSearchPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nRaw string: %s", err, rawJSON)
	}

	return &plan, nil
}
