package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Finnegan/core/module"
	"Finnegan/modules/bugu/config"
)

func TestDistillerPromptScheme1(t *testing.T) {
	var capturedPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 {
			capturedPrompt = req.Messages[0].Content
		}

		response := `{
			"choices": [{"message": {"content": "{\"source_url\":\"http://test.com\",\"matched_anchor\":\"TestAnchor*\",\"original_text\":\"some text\"}"}}]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	llmCfg := module.LLMConfig{
		Provider:     "openai",
		BaseURL:      server.URL,
		APIKey:       "dummy",
		DefaultModel: "gpt-test",
	}

	router := NewRouter(&config.Config{}, llmCfg, llmCfg)

	keywords := map[string][]string{
		"entity": {"AsteriskTag *"},
	}

	card, err := router.DistillMarkdown("test intent", "test md", "http://test.com", "Chinese", keywords)
	if err != nil {
		t.Fatalf("DistillMarkdown failed: %v", err)
	}

	if card == nil {
		t.Fatalf("expected card to not be nil")
	}

	// 1. Verify prompt contains Scheme 1 rules
	if !strings.Contains(capturedPrompt, "ANY of the 4 dimensions (object, motif, mood, concept) is eligible to pass gating independently") {
		t.Error("prompt missing Scheme 1 dimension eligibility rule")
	}
	if !strings.Contains(capturedPrompt, "If matching an 'object' anchor") {
		t.Error("prompt missing Scheme 1 object anchor rules")
	}
	if !strings.Contains(capturedPrompt, "If CARD ANCHORS is empty/null, evaluate if the text substantively addresses the USER INTENT") {
		t.Error("prompt missing empty anchor fallback rule")
	}
	if !strings.Contains(capturedPrompt, "\"title\": \"article title or concise heading") {
		t.Error("prompt missing title field in JSON schema")
	}

	// 2. Verify prompt strips '*' from keywords
	if strings.Contains(capturedPrompt, "AsteriskTag *") {
		t.Error("prompt leaked trailing '*' in keyword")
	}
	if !strings.Contains(capturedPrompt, "AsteriskTag") {
		t.Error("prompt missing sanitized keyword")
	}

	// 3. Verify gating/stripping '*' from MatchedAnchor
	if card.MatchedAnchor != "TestAnchor" {
		t.Errorf("expected MatchedAnchor to be 'TestAnchor', got '%s'", card.MatchedAnchor)
	}
}
