package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/keywords"
	"Finnegan/core/module"
)

type mockLLM struct {
	jsonResp string
}

func (m *mockLLM) GenerateJSON(ctx context.Context, modelName string, prompt string) (string, error) {
	return m.jsonResp, nil
}

func TestApp_KeywordEngine_PromoteAndCaptureIntegration(t *testing.T) {
	tempHome := t.TempDir()
	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}
	defer app.CardManager.Close()

	os.MkdirAll(filepath.Join(tempHome, "Scattering"), 0755)

	mock := &mockLLM{
		jsonResp: `{
  "keywords": {
    "entity": ["Finnegan", "Wails"],
    "judgment": ["本地优先消除网络依赖"]
  }
}`,
	}

	app.KeywordEngine = keywords.NewKeywordEngine(
		module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "mock-key"},
		keywords.DefaultConfig(),
	).WithClient(mock)

	// Subscribe to EventKeywordsExtracted
	kwDoneCh := bus.GetBus().Subscribe(bus.EventKeywordsExtracted, "integration-tester")
	defer bus.GetBus().Unsubscribe(bus.EventKeywordsExtracted, "integration-tester")

	// Subscribe to EventThoughtCaptured to simulate startup() listener
	kwThoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "integration-kw-engine")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "integration-kw-engine")
	go func() {
		for event := range kwThoughtCh {
			if data, ok := event.Data.(bus.EventThoughtCapturedData); ok {
				if app.KeywordEngine != nil && data.FilePath != "" && data.DocID != "" {
					app.KeywordEngine.ProcessCard(data.FilePath, data.DocID, data.Text)
				}
			}
		}
	}()

	// 1. Test CaptureThought
	docID, err := app.CaptureThought("测试 Finnegan Wails 本地优先卡片创建")
	if err != nil {
		t.Fatalf("CaptureThought failed: %v", err)
	}
	if docID == "" {
		t.Fatal("empty docID returned")
	}

	// Wait for keywords extracted event
	select {
	case event := <-kwDoneCh:
		data, ok := event.Data.(bus.EventKeywordsExtractedData)
		if !ok {
			t.Fatalf("unexpected event data: %T", event.Data)
		}
		if data.DocID != docID {
			t.Errorf("expected docID %s, got %s", docID, data.DocID)
		}
		if len(data.Keywords["entity"]) != 2 {
			t.Errorf("expected 2 entities, got %v", data.Keywords["entity"])
		}

		// Verify on-disk file
		contentBytes, err := os.ReadFile(data.FilePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		contentStr := string(contentBytes)
		if !strings.Contains(contentStr, "keywords:") || !strings.Contains(contentStr, "Finnegan") {
			t.Errorf("file missing keywords block:\n%s", contentStr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for keywords extracted event from CaptureThought")
	}

	// 2. Test PromoteTempCard
	app.CardManager.InjectBabelCards([]Card{
		{
			ID:      "babel-temp-1",
			Title:   "Temporary Babel Thought",
			Content: "Babel card to promote",
		},
	})

	promoted, err := app.PromoteTempCard("babel-temp-1")
	if err != nil {
		t.Fatalf("PromoteTempCard failed: %v", err)
	}
	if promoted.FilePath == "" {
		t.Fatal("expected non-empty FilePath on promoted card")
	}

	select {
	case event := <-kwDoneCh:
		data := event.Data.(bus.EventKeywordsExtractedData)
		if data.DocID != promoted.ID {
			t.Errorf("expected promoted ID %s, got %s", promoted.ID, data.DocID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for keywords extracted event from PromoteTempCard")
	}
}
