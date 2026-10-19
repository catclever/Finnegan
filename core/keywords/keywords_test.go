package keywords

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/module"
)

// mockLLMClient simulates Gemini response for testing gating and extraction.
type mockLLMClient struct {
	responseJSON string
	lastPrompt   string
	lastModel    string
}

func (m *mockLLMClient) GenerateJSON(ctx context.Context, modelName string, prompt string) (string, error) {
	m.lastModel = modelName
	m.lastPrompt = prompt
	return m.responseJSON, nil
}

func TestExtractKeywords_SalienceGating(t *testing.T) {
	mockResp := `{
  "keywords": {
    "entity": ["WebKitGTK 4.1", "Tauri v2"],
    "symptom": ["undefined symbol: g_once_init_enter_pointer"],
    "event": [],
    "judgment": [],
    "emotion": [],
    "question": [],
    "metaphor": [],
    "unrelated_dim": ["should be filtered out"]
  }
}`

	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := Config{
		EnabledDimensions: []string{
			DimensionEntity,
			DimensionSymptom,
			DimensionEvent,
			DimensionJudgment,
			DimensionEmotion,
			DimensionQuestion,
		},
	}
	llmCfg := module.LLMConfig{
		DefaultModel: "gemini-3.8-flash",
		APIKey:       "test-key",
	}

	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	cardText := `Compiling WebKitGTK 4.1 with Tauri v2 fails with undefined symbol: g_once_init_enter_pointer`
	profile, err := engine.ExtractKeywords(context.Background(), cardText)
	if err != nil {
		t.Fatalf("ExtractKeywords failed: %v", err)
	}

	// 1. Check prompt includes active dimensions and gating prompts
	if !strings.Contains(mock.lastPrompt, DimensionEntity) || !strings.Contains(mock.lastPrompt, DimensionSymptom) {
		t.Errorf("Prompt should include active dimensions: %s", mock.lastPrompt)
	}
	if !strings.Contains(mock.lastPrompt, "WebKitGTK 4.1") {
		t.Errorf("Prompt should contain card text: %s", mock.lastPrompt)
	}

	// 2. Check triggered dimensions
	if len(profile["entity"]) != 2 || profile["entity"][0] != "WebKitGTK 4.1" {
		t.Errorf("Expected entity keywords, got %v", profile["entity"])
	}
	if len(profile["symptom"]) != 1 || profile["symptom"][0] != "undefined symbol: g_once_init_enter_pointer" {
		t.Errorf("Expected symptom keywords, got %v", profile["symptom"])
	}

	// 3. Check gated-out empty dimensions
	if len(profile["event"]) != 0 {
		t.Errorf("Expected event to be empty, got %v", profile["event"])
	}
	if len(profile["emotion"]) != 0 {
		t.Errorf("Expected emotion to be empty, got %v", profile["emotion"])
	}

	// 4. Check inactive/unrelated dimensions filtered
	if _, exists := profile["unrelated_dim"]; exists {
		t.Errorf("Unrelated dimension should have been filtered out")
	}
}

func TestProcessCard_SuccessAndEventPublishing(t *testing.T) {
	tempDir := t.TempDir()
	cardFile := filepath.Join(tempDir, "20261017-1500_test.md")
	docID := "01J9TESTDOCID12345"

	initialContent := `---
id: 01J9TESTDOCID12345
type: thought
created: 2026-10-17T15:00:00+08:00
updated: 2026-10-17T15:00:00+08:00
source: local
tags: []
publish: false
---

# Local Sync Architecture

Exploring CRDT and local-first architecture trade-offs.
`
	if err := os.WriteFile(cardFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial card: %v", err)
	}

	mockResp := `{
  "keywords": {
    "entity": ["CRDT", "Local-First"],
    "judgment": ["同步复杂性与数据所有权权衡"],
    "question": [],
    "emotion": []
  }
}`
	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	// Subscribe to EventKeywordsExtracted on EventBus
	extractedCh := bus.GetBus().Subscribe(bus.EventKeywordsExtracted, "test-subscriber")
	defer bus.GetBus().Unsubscribe(bus.EventKeywordsExtracted, "test-subscriber")

	// Run ProcessCardSync
	err := engine.ProcessCardSync(context.Background(), cardFile, docID, "Exploring CRDT and local-first architecture trade-offs.")
	if err != nil {
		t.Fatalf("ProcessCardSync failed: %v", err)
	}

	// 1. Verify EventKeywordsExtracted event received
	select {
	case event := <-extractedCh:
		data, ok := event.Data.(bus.EventKeywordsExtractedData)
		if !ok {
			t.Fatalf("Event data is not EventKeywordsExtractedData: %T", event.Data)
		}
		if data.DocID != docID {
			t.Errorf("Expected DocID %s, got %s", docID, data.DocID)
		}
		if data.FilePath != cardFile {
			t.Errorf("Expected FilePath %s, got %s", cardFile, data.FilePath)
		}
		if len(data.Keywords["entity"]) != 2 {
			t.Errorf("Expected 2 entity keywords in event, got %v", data.Keywords["entity"])
		}
		if len(data.Keywords["judgment"]) != 1 {
			t.Errorf("Expected 1 judgment keyword in event, got %v", data.Keywords["judgment"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for EventKeywordsExtracted event")
	}

	// 2. Verify file content on disk has updated frontmatter
	updatedBytes, err := os.ReadFile(cardFile)
	if err != nil {
		t.Fatalf("Failed to read updated card file: %v", err)
	}
	updatedStr := string(updatedBytes)

	if !strings.Contains(updatedStr, "keywords:") {
		t.Errorf("Updated file missing keywords: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "CRDT") || !strings.Contains(updatedStr, "Local-First") {
		t.Errorf("Updated file missing entity keywords: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "同步复杂性与数据所有权权衡") {
		t.Errorf("Updated file missing judgment keyword: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "Exploring CRDT and local-first architecture trade-offs.") {
		t.Errorf("Original body lost in updated file: %s", updatedStr)
	}
}
