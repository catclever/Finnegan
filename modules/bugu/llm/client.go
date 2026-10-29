package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"Finnegan/core/module"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// CleanJSONText strips any markdown code blocks (```json ... ``` or ``` ... ```) from raw LLM output.
func CleanJSONText(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	return strings.TrimSpace(cleaned)
}

// GenerateCompletion calls either Google Gemini or OpenAI-compatible backend depending on llmCfg.
func GenerateCompletion(ctx context.Context, llmCfg module.LLMConfig, modelName string, prompt string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(llmCfg.Provider))
	isCustomOpenAI := provider == "openai" ||
		provider == "openai_compatible" ||
		provider == "deepseek" ||
		strings.HasPrefix(llmCfg.BaseURL, "http://") ||
		strings.HasPrefix(llmCfg.BaseURL, "https://")

	if isCustomOpenAI {
		return callOpenAICompatible(ctx, llmCfg, modelName, prompt)
	}

	return callGemini(ctx, llmCfg, modelName, prompt)
}

func callOpenAICompatible(ctx context.Context, llmCfg module.LLMConfig, modelName string, prompt string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(llmCfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		if strings.HasSuffix(endpoint, "/v1") {
			endpoint = endpoint + "/chat/completions"
		} else {
			endpoint = endpoint + "/v1/chat/completions"
		}
	}

	mName := strings.TrimSpace(modelName)
	if mName == "" {
		mName = strings.TrimSpace(llmCfg.DefaultModel)
	}
	if mName == "" {
		mName = "gpt-4o"
	}

	reqBody := map[string]interface{}{
		"model": mName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(llmCfg.APIKey); apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai request error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to decode openai response: %w\nBody: %s", err, string(bodyBytes))
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return "", fmt.Errorf("openai response error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices from openai response: %s", string(bodyBytes))
	}

	return CleanJSONText(chatResp.Choices[0].Message.Content), nil
}

func callGemini(ctx context.Context, llmCfg module.LLMConfig, modelName string, prompt string) (string, error) {
	if llmCfg.APIKey == "" {
		return "", fmt.Errorf("gemini API key is not configured")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(llmCfg.APIKey))
	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	mName := strings.TrimSpace(modelName)
	if mName == "" {
		mName = strings.TrimSpace(llmCfg.DefaultModel)
	}
	if mName == "" {
		mName = "gemini-3.8-flash"
	}

	model := client.GenerativeModel(mName)
	model.ResponseMIMEType = "application/json"

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini generate content failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from gemini")
	}

	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		return CleanJSONText(string(textPart)), nil
	}

	return "", fmt.Errorf("unexpected gemini response format")
}
