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
	"Finnegan/core/watcher"
)

type mockLLMUnify struct{}

func (m *mockLLMUnify) GenerateJSON(ctx context.Context, modelName string, prompt string) (string, error) {
	// AI extraction generates overlapping keywords (WebKitGTK 4.1) and new ones (Tauri v2)
	return `{
  "keywords": {
    "object": ["WebKitGTK 4.1", "Tauri v2"],
    "motif": ["体验新玩具"]
  }
}`, nil
}

func TestUnifyKeywordsAndStripHashtags_EndToEnd(t *testing.T) {
	tempHome := t.TempDir()
	scatteringDir := filepath.Join(tempHome, "Scattering")
	os.MkdirAll(scatteringDir, 0755)

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}
	defer app.CardManager.Close()

	wStorage := &WatcherStorage{}
	err := watcher.GetWatcher().Initialize([]string{scatteringDir}, 50, wStorage, tempHome)
	if err != nil {
		t.Fatalf("failed to initialize watcher: %v", err)
	}

	app.KeywordEngine = keywords.NewKeywordEngine(
		module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "mock-key"},
		keywords.DefaultConfig(),
	).WithClient(&mockLLMUnify{})

	kwDoneCh := bus.GetBus().Subscribe(bus.EventKeywordsExtracted, "unify-integration-tester")
	defer bus.GetBus().Unsubscribe(bus.EventKeywordsExtracted, "unify-integration-tester")

	kwThoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "unify-kw-engine")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "unify-kw-engine")
	go func() {
		for event := range kwThoughtCh {
			if data, ok := event.Data.(bus.EventThoughtCapturedData); ok {
				if app.KeywordEngine != nil && data.FilePath != "" && data.DocID != "" {
					app.KeywordEngine.ProcessCard(data.FilePath, data.DocID, data.Text)
				}
			}
		}
	}()

	// Capture thought with inline and trailing hashtags
	rawInput := "今天测试了 #WebKitGTK 4.1 和 #跨平台UI。\n\n#松弛感 #治愈"
	docID, err := app.CaptureThought(rawInput)
	if err != nil {
		t.Fatalf("CaptureThought failed: %v", err)
	}

	// Wait for KeywordEngine to finish and emit EventKeywordsExtracted
	select {
	case event := <-kwDoneCh:
		data, ok := event.Data.(bus.EventKeywordsExtractedData)
		if !ok {
			t.Fatalf("unexpected event data: %T", event.Data)
		}
		if data.DocID != docID {
			t.Errorf("expected docID %s, got %s", docID, data.DocID)
		}

		// Read file on disk
		contentBytes, err := os.ReadFile(data.FilePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		fileStr := string(contentBytes)

		// 1. Deprecate separate tags: [] field -> Must not contain "tags:"
		if strings.Contains(fileStr, "tags:") {
			t.Errorf("frontmatter must not contain deprecated tags field:\n%s", fileStr)
		}

		// 2. Trailing hashtags must be stripped completely from body
		if strings.Contains(fileStr, "#松弛感") || strings.Contains(fileStr, "#治愈") {
			t.Errorf("trailing hashtags were not stripped from body:\n%s", fileStr)
		}

		// 3. Inline '#' must be stripped from inline hashtags to preserve grammatical fluency
		if strings.Contains(fileStr, "#WebKitGTK") || strings.Contains(fileStr, "#跨平台UI") {
			t.Errorf("inline hashtags '#' were not stripped:\n%s", fileStr)
		}
		if !strings.Contains(fileStr, "今天测试了 WebKitGTK 4.1 和 跨平台UI。") {
			t.Errorf("expected body to preserve grammatical fluency:\n%s", fileStr)
		}

		// 4. Frontmatter keywords must have manual '*' entries preserved and not duplicated
		if !strings.Contains(fileStr, "WebKitGTK*") && !strings.Contains(fileStr, "WebKitGTK 4.1*") {
			t.Errorf("manual object keyword with '*' marker missing:\n%s", fileStr)
		}
		if !strings.Contains(fileStr, "松弛感*") {
			t.Errorf("manual mood keyword '松弛感*' missing:\n%s", fileStr)
		}
		if !strings.Contains(fileStr, "治愈*") {
			t.Errorf("manual mood keyword '治愈*' missing:\n%s", fileStr)
		}

		// 5. New AI keyword 'Tauri v2' and '体验新玩具' must also be present
		if !strings.Contains(fileStr, "Tauri v2") {
			t.Errorf("AI keyword 'Tauri v2' missing:\n%s", fileStr)
		}
		if !strings.Contains(fileStr, "体验新玩具") {
			t.Errorf("AI keyword '体验新玩具' missing:\n%s", fileStr)
		}

	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for keywords extracted event")
	}
}
