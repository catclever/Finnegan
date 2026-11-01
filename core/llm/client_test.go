package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"Finnegan/core/module"
)

func TestCleanJSONText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "markdown json fence",
			input:    "```json\n{\"hello\":\"world\"}\n```",
			expected: "{\"hello\":\"world\"}",
		},
		{
			name:     "markdown plain fence",
			input:    "```\n[1, 2, 3]\n```",
			expected: "[1, 2, 3]",
		},
		{
			name:     "raw json",
			input:    "  {\"key\": 42}  ",
			expected: "{\"key\": 42}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CleanJSONText(tc.input)
			if got != tc.expected {
				t.Errorf("CleanJSONText(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGenerateCompletion_OpenAICompatible(t *testing.T) {
	var receivedAuth string
	var receivedModel string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if m, ok := payload["model"].(string); ok {
			receivedModel = m
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + "```json\\n{\\\"ok\\\":true}\\n```" + `"}}]}`))
	})

	origClient := HTTPClient
	HTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}
	defer func() { HTTPClient = origClient }()

	cfg := module.LLMConfig{
		Provider:     "openai",
		APIKey:       "test-secret-key",
		BaseURL:      "http://mock-llm.local",
		DefaultModel: "deepseek-chat",
	}

	res, err := GenerateCompletion(context.Background(), cfg, "", "Say ok")
	if err != nil {
		t.Fatalf("GenerateCompletion returned error: %v", err)
	}
	if res != `{"ok":true}` {
		t.Errorf("got %q, want %q", res, `{"ok":true}`)
	}
	if receivedAuth != "Bearer test-secret-key" {
		t.Errorf("Authorization header = %q, want %q", receivedAuth, "Bearer test-secret-key")
	}
	if receivedModel != "deepseek-chat" {
		t.Errorf("model = %q, want %q", receivedModel, "deepseek-chat")
	}
}

func TestGenerateCompletion_GeminiMissingKey(t *testing.T) {
	cfg := module.LLMConfig{
		Provider: "gemini",
		APIKey:   "",
	}
	_, err := GenerateCompletion(context.Background(), cfg, "", "test")
	if err == nil {
		t.Fatal("expected error when Gemini APIKey is empty")
	}
}
