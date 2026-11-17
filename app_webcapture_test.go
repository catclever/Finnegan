package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/webcapture"
)

type mockRoundTrip func(req *http.Request) (*http.Response, error)

func (m mockRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}

func TestCaptureURL_FailFast_NoDirtyFiles(t *testing.T) {
	tmpHome := t.TempDir()
	scatteringDir := filepath.Join(tmpHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create Scattering dir: %v", err)
	}

	app := &App{
		ctx:  context.Background(),
		Home: tmpHome,
	}

	oldFactory := capturePipelineFactory
	defer func() { capturePipelineFactory = oldFactory }()

	// 1. Test 403 Forbidden Fail-fast
	capturePipelineFactory = func(cfg config.JinaConfig) *webcapture.Pipeline {
		p := webcapture.NewPipeline(cfg)
		p.Fetcher.WithClient(&http.Client{
			Transport: mockRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(bytes.NewBufferString("Forbidden access")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		return p
	}

	card, err := app.CaptureURL("https://example.com/protected")
	if err == nil {
		t.Fatalf("expected error for 403, got nil card: %+v", card)
	}
	if !errors.Is(err, webcapture.ErrPageProtected) {
		t.Errorf("expected ErrPageProtected, got %v", err)
	}

	// Verify no files were created in Scattering
	files, err := os.ReadDir(scatteringDir)
	if err != nil {
		t.Fatalf("failed to read Scattering directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files in Scattering after fail-fast, found %d", len(files))
	}

	// 2. Test Anti-bot Fail-fast
	capturePipelineFactory = func(cfg config.JinaConfig) *webcapture.Pipeline {
		p := webcapture.NewPipeline(cfg)
		p.Fetcher.WithClient(&http.Client{
			Transport: mockRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("Please verify you are human - Cloudflare turnstile")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		return p
	}

	card, err = app.CaptureURL("https://example.com/antibot")
	if err == nil {
		t.Fatalf("expected error for anti-bot, got nil card")
	}
	if !errors.Is(err, webcapture.ErrBlocked) {
		t.Errorf("expected ErrBlocked, got %v", err)
	}

	files, _ = os.ReadDir(scatteringDir)
	if len(files) != 0 {
		t.Fatalf("expected 0 files in Scattering after anti-bot fail-fast, found %d", len(files))
	}

	// 3. Test Empty Content Fail-fast (< 30 chars)
	capturePipelineFactory = func(cfg config.JinaConfig) *webcapture.Pipeline {
		p := webcapture.NewPipeline(cfg)
		p.Fetcher.WithClient(&http.Client{
			Transport: mockRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("Too short")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		return p
	}

	card, err = app.CaptureURL("https://example.com/empty")
	if err == nil {
		t.Fatalf("expected error for empty content, got nil card")
	}
	if !errors.Is(err, webcapture.ErrEmptyContent) {
		t.Errorf("expected ErrEmptyContent, got %v", err)
	}

	files, _ = os.ReadDir(scatteringDir)
	if len(files) != 0 {
		t.Fatalf("expected 0 files in Scattering after short body fail-fast, found %d", len(files))
	}
}

func TestCaptureURL_NoteSuccess(t *testing.T) {
	tmpHome := t.TempDir()
	scatteringDir := filepath.Join(tmpHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create Scattering dir: %v", err)
	}

	app := &App{
		ctx:  context.Background(),
		Home: tmpHome,
	}

	oldFactory := capturePipelineFactory
	defer func() { capturePipelineFactory = oldFactory }()

	noteBody := `Title: Architectural Patterns in Distributed Systems
URL Source: https://example.com/arch/patterns
Markdown Content:
# Architectural Patterns in Distributed Systems

In this paper, we explore how event-driven architectures minimize service coupling.
Local message buses allow components to interact without hard dependencies.
`

	capturePipelineFactory = func(cfg config.JinaConfig) *webcapture.Pipeline {
		p := webcapture.NewPipeline(cfg)
		p.Fetcher.WithClient(&http.Client{
			Transport: mockRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(noteBody)),
					Header:     make(http.Header),
				}, nil
			}),
		})
		return p
	}

	eventChan := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_webcapture_note")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_webcapture_note")

	targetURL := "https://example.com/arch/patterns"
	card, err := app.CaptureURL(targetURL)
	if err != nil {
		t.Fatalf("CaptureURL failed: %v", err)
	}

	if card == nil {
		t.Fatal("expected non-nil card")
	}
	if card.Type != CardTypeNote {
		t.Errorf("expected CardTypeNote, got %v", card.Type)
	}
	if card.Title != "Architectural Patterns in Distributed Systems" {
		t.Errorf("expected title 'Architectural Patterns in Distributed Systems', got %q", card.Title)
	}
	if card.SourceURL != targetURL {
		t.Errorf("expected SourceURL %q, got %q", targetURL, card.SourceURL)
	}
	if card.ID == "" {
		t.Errorf("expected valid ID, got empty string")
	}
	if card.CreatedAt.IsZero() {
		t.Errorf("expected non-zero CreatedAt")
	}

	// Verify filename in Scattering directory: <YYYYMMDD-HHMM>-<slug>.note.md
	files, err := os.ReadDir(scatteringDir)
	if err != nil {
		t.Fatalf("failed to read Scattering directory: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file in Scattering, found %d", len(files))
	}

	createdFileName := files[0].Name()
	if !strings.HasSuffix(createdFileName, ".note.md") {
		t.Errorf("expected file to end with '.note.md', got %q", createdFileName)
	}

	// Read and verify file contents
	fileBytes, err := os.ReadFile(filepath.Join(scatteringDir, createdFileName))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	fileStr := string(fileBytes)

	if !strings.Contains(fileStr, "type: note") {
		t.Errorf("file missing 'type: note'")
	}
	if !strings.Contains(fileStr, "source: https://example.com/arch/patterns") {
		t.Errorf("file missing source URL")
	}
	if !strings.Contains(fileStr, "title: Architectural Patterns in Distributed Systems") {
		t.Errorf("file missing title in frontmatter")
	}
	if !strings.Contains(fileStr, "In this paper, we explore how event-driven architectures") {
		t.Errorf("file missing body content")
	}

	// Verify bus event was emitted
	select {
	case evt := <-eventChan:
		data, ok := evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("expected EventThoughtCapturedData, got %T", evt.Data)
		}
		if data.DocID != card.ID {
			t.Errorf("event DocID %q != card ID %q", data.DocID, card.ID)
		}
		if data.Type != "note" {
			t.Errorf("event Type %q != 'note'", data.Type)
		}
		if data.Title != card.Title {
			t.Errorf("event Title %q != card Title %q", data.Title, card.Title)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}
}

func TestCaptureURL_ChatLogSuccess(t *testing.T) {
	tmpHome := t.TempDir()
	scatteringDir := filepath.Join(tmpHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create Scattering dir: %v", err)
	}

	app := &App{
		ctx:  context.Background(),
		Home: tmpHome,
	}

	oldFactory := capturePipelineFactory
	defer func() { capturePipelineFactory = oldFactory }()

	chatLogContent := `# Discussion with Assistant

User 2026-11-06 10:00
Please explain the single responsibility principle.

Assistant 2026-11-06 10:01
The single responsibility principle states that a module should have one, and only one, reason to change.
`

	capturePipelineFactory = func(cfg config.JinaConfig) *webcapture.Pipeline {
		p := webcapture.NewPipeline(cfg)
		p.Fetcher.WithClient(&http.Client{
			Transport: mockRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(chatLogContent)),
					Header:     make(http.Header),
				}, nil
			}),
		})
		return p
	}

	eventChan := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_webcapture_chatlog")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_webcapture_chatlog")

	targetURL := "https://chatgpt.com/share/672a0011"
	card, err := app.CaptureURL(targetURL)
	if err != nil {
		t.Fatalf("CaptureURL failed: %v", err)
	}

	if card == nil {
		t.Fatal("expected non-nil card")
	}
	if card.Type != CardTypeChatLog {
		t.Errorf("expected CardTypeChatLog, got %v", card.Type)
	}
	if card.Title != "Discussion with Assistant" {
		t.Errorf("expected title 'Discussion with Assistant', got %q", card.Title)
	}
	if card.SourceURL != targetURL {
		t.Errorf("expected SourceURL %q, got %q", targetURL, card.SourceURL)
	}

	// Verify filename in Scattering directory: <YYYYMMDD-HHMM>-<slug>.chat_log.md
	files, err := os.ReadDir(scatteringDir)
	if err != nil {
		t.Fatalf("failed to read Scattering directory: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file in Scattering, found %d", len(files))
	}

	createdFileName := files[0].Name()
	if !strings.HasSuffix(createdFileName, ".chat_log.md") {
		t.Errorf("expected file to end with '.chat_log.md', got %q", createdFileName)
	}

	// Read and verify file contents
	fileBytes, err := os.ReadFile(filepath.Join(scatteringDir, createdFileName))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	fileStr := string(fileBytes)

	if !strings.Contains(fileStr, "type: chat_log") {
		t.Errorf("file missing 'type: chat_log'")
	}
	if !strings.Contains(fileStr, "source: https://chatgpt.com/share/672a0011") {
		t.Errorf("file missing source URL")
	}
	if !strings.Contains(fileStr, "participants: [User, Assistant]") {
		t.Errorf("file missing participants")
	}

	// Verify bus event was emitted
	select {
	case evt := <-eventChan:
		data, ok := evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("expected EventThoughtCapturedData, got %T", evt.Data)
		}
		if data.DocID != card.ID {
			t.Errorf("event DocID %q != card ID %q", data.DocID, card.ID)
		}
		if data.Type != "chat_log" {
			t.Errorf("event Type %q != 'chat_log'", data.Type)
		}
		if data.Title != card.Title {
			t.Errorf("event Title %q != card Title %q", data.Title, card.Title)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}
}
