package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"Finnegan/core/module"
	"Finnegan/modules/bugu/config"
)

func TestCleanJSONText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{\"key\": \"val\"}", "{\"key\": \"val\"}"},
		{"```json\n{\"key\": \"val\"}\n```", "{\"key\": \"val\"}"},
		{"```\n{\"key\": \"val\"}\n```", "{\"key\": \"val\"}"},
		{"  ```json{\"key\": \"val\"}```  ", "{\"key\": \"val\"}"},
	}
	for _, tc := range tests {
		if got := CleanJSONText(tc.input); got != tc.expected {
			t.Errorf("CleanJSONText(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestGenerateCompletion_OpenAICompatible(t *testing.T) {
	var receivedAuth string
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("expected URL ending in /chat/completions, got %s", r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		responseJSON := `{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "{\"status\": \"ok\", \"model_used\": \"test-model\"}"
					}
				}
			]
		}`
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	ctx := context.Background()
	llmCfg := module.LLMConfig{
		Provider:     "openai",
		BaseURL:      server.URL,
		APIKey:       "sk-test-secret-123",
		DefaultModel: "test-model-default",
	}

	result, err := GenerateCompletion(ctx, llmCfg, "custom-model", "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedResult := `{"status": "ok", "model_used": "test-model"}`
	if result != expectedResult {
		t.Errorf("expected %q, got %q", expectedResult, result)
	}

	if receivedAuth != "Bearer sk-test-secret-123" {
		t.Errorf("expected Bearer header, got %q", receivedAuth)
	}

	if receivedBody["model"] != "custom-model" {
		t.Errorf("expected model 'custom-model', got %v", receivedBody["model"])
	}
}

func TestNewRouter_ModelSeparation(t *testing.T) {
	cfg := &config.Config{
		RouterModel:    "deepseek-v4.1-flash",
		DistillerModel: "gemini-3.8-flash",
	}
	routerLLM := module.LLMConfig{
		Provider:     "openai",
		BaseURL:      "http://172.16.51.7:48787/v1",
		DefaultModel: "deepseek-v4.1-flash",
	}
	distillerLLM := module.LLMConfig{
		Provider:     "gemini",
		APIKey:       "AIzaTestKey",
		DefaultModel: "gemini-3.8-flash",
	}

	// 1. Both separate
	r1 := NewRouter(cfg, routerLLM, distillerLLM)
	if r1.LLM.Provider != "openai" {
		t.Errorf("expected router provider 'openai', got %s", r1.LLM.Provider)
	}
	if r1.GetDistillerLLM().Provider != "gemini" {
		t.Errorf("expected distiller provider 'gemini', got %s", r1.GetDistillerLLM().Provider)
	}

	// 2. Single LLM fallback
	r2 := NewRouter(cfg, routerLLM)
	if r2.GetDistillerLLM().Provider != "openai" {
		t.Errorf("expected distiller fallback to router LLM, got %s", r2.GetDistillerLLM().Provider)
	}
}

func TestGenerateCompletion_LiveDeepSeek(t *testing.T) {
	if os.Getenv("LIVE_DEEPSEEK_TEST") == "" {
		t.Skip("skipping live test without LIVE_DEEPSEEK_TEST=1")
	}
	ctx := context.Background()
	llmCfg := module.LLMConfig{
		Provider:     "openai",
		BaseURL:      "http://172.16.51.7:48787/v1",
		DefaultModel: "deepseek-v4.1-flash",
	}
	res, err := GenerateCompletion(ctx, llmCfg, "", "Say 'OK' in JSON format: {\"status\": \"OK\"}")
	if err != nil {
		t.Fatalf("Live DeepSeek call failed: %v", err)
	}
	if !strings.Contains(res, "OK") {
		t.Errorf("expected 'OK' in response, got %s", res)
	}
}
