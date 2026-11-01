package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreadReference_CaptureAndLoad(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_thread_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}

	// 1. Create a root card that will be referenced
	noteID, err := app.CaptureNote("# 架构设计\n这是关于Finnegan架构设计的说明")
	if err != nil {
		t.Fatalf("CaptureNote failed: %v", err)
	}
	if noteID == "" {
		t.Fatal("expected non-empty noteID")
	}

	// Verify root note is in LoadLocalCards
	loaded := app.LoadLocalCards()
	if len(loaded) == 0 {
		t.Fatal("expected at least 1 card in LoadLocalCards")
	}

	// 2. Capture a thought referencing @架构设计
	thoughtID, err := app.CaptureThought("今天深化了关于 @架构设计 的思考，增加了3D Peek Carousel")
	if err != nil {
		t.Fatalf("CaptureThought failed: %v", err)
	}

	// 3. Load cards and check that thought has Threads pointing to noteID
	loadedAfter := app.LoadLocalCards()
	var thoughtCard *Card
	for i := range loadedAfter {
		if loadedAfter[i].ID == thoughtID {
			thoughtCard = &loadedAfter[i]
			break
		}
	}
	if thoughtCard == nil {
		t.Fatalf("thought card %s not found in LoadLocalCards", thoughtID)
	}

	if len(thoughtCard.Threads) != 1 {
		t.Fatalf("expected 1 thread in thoughtCard, got %d: %+v", len(thoughtCard.Threads), thoughtCard.Threads)
	}
	if thoughtCard.Threads[0].ID != noteID {
		t.Fatalf("expected thread ID to be %s, got %s", noteID, thoughtCard.Threads[0].ID)
	}
	if thoughtCard.Threads[0].Title != "架构设计" {
		t.Fatalf("expected thread Title to be '架构设计', got %s", thoughtCard.Threads[0].Title)
	}

	// Check file content on disk has threads: frontmatter
	files, _ := filepath.Glob(filepath.Join(tempHome, "Scattering", "*.epiphany.md"))
	if len(files) == 0 {
		t.Fatal("no epiphany.md files found")
	}
	rawBytes, _ := os.ReadFile(files[0])
	rawStr := string(rawBytes)
	if !strings.Contains(rawStr, "threads:") {
		t.Fatalf("expected markdown file to contain 'threads:', got:\n%s", rawStr)
	}
	if !strings.Contains(rawStr, noteID) {
		t.Fatalf("expected markdown file to contain noteID %s, got:\n%s", noteID, rawStr)
	}
}
