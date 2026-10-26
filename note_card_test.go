package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/watcher"
	"Finnegan/modules/bugu/api"
	buguconfig "Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/llm"
	bugumodels "Finnegan/modules/bugu/models"
)

func TestCaptureNote_And_LoadLocalCards(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_note_test_*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	scatteringDir := filepath.Join(tempHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create Scattering dir: %v", err)
	}

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}

	// Initialize watcher
	wStorage := &WatcherStorage{}
	err = watcher.GetWatcher().Initialize([]string{scatteringDir}, 50, wStorage, tempHome)
	if err != nil {
		t.Fatalf("failed to initialize watcher: %v", err)
	}

	// Track bus events
	eventCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_note_listener")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_note_listener")

	noteText := "docker run -d --name test-redis -p 6379:6379 redis:alpine"
	docID, err := app.CaptureNote(noteText)
	if err != nil {
		t.Fatalf("CaptureNote failed: %v", err)
	}
	if docID == "" {
		t.Fatal("expected non-empty docID")
	}

	// Verify EventThoughtCaptured has Type: "note"
	select {
	case evt := <-eventCh:
		data, ok := evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("unexpected event data: %T", evt.Data)
		}
		if data.DocID != docID {
			t.Fatalf("expected docID %s, got %s", docID, data.DocID)
		}
		if data.Type != "note" {
			t.Fatalf("expected Type 'note', got %q", data.Type)
		}
		if data.Text != noteText {
			t.Fatalf("expected Text %q, got %q", noteText, data.Text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}

	// Verify file exists on disk with .note.md suffix
	files, err := os.ReadDir(scatteringDir)
	if err != nil {
		t.Fatalf("failed to read scattering dir: %v", err)
	}
	var noteFilePath string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".note.md") {
			noteFilePath = filepath.Join(scatteringDir, f.Name())
			break
		}
	}
	if noteFilePath == "" {
		t.Fatal("expected .note.md file to be created in Scattering")
	}

	fileBytes, err := os.ReadFile(noteFilePath)
	if err != nil {
		t.Fatalf("failed to read note file: %v", err)
	}
	fileContent := string(fileBytes)
	if !strings.Contains(fileContent, "type: note") {
		t.Fatalf("expected frontmatter to contain 'type: note', got: %s", fileContent)
	}
	if strings.Contains(fileContent, "keywords:") {
		t.Fatalf("note card must not contain keywords frontmatter: %s", fileContent)
	}

	// Verify LoadLocalCards reads it back as CardTypeNote
	loadedCards := app.LoadLocalCards()
	if len(loadedCards) == 0 {
		t.Fatal("LoadLocalCards returned 0 cards")
	}
	var loadedNote *Card
	for i := range loadedCards {
		if loadedCards[i].ID == docID {
			loadedNote = &loadedCards[i]
			break
		}
	}
	if loadedNote == nil {
		t.Fatalf("expected to find note card with id %s, got: %+v", docID, loadedCards)
	}
	if loadedNote.Type != CardTypeNote {
		t.Fatalf("expected card type %q, got %q", CardTypeNote, loadedNote.Type)
	}
	if loadedNote.Content != noteText {
		t.Fatalf("expected content %q, got %q", noteText, loadedNote.Content)
	}

	// Verify Bugu radar rejects note cards
	time.Sleep(100 * time.Millisecond)
	cfg := &buguconfig.Config{
		Targets:   []bugumodels.Target{},
		TargetMap: make(map[string]bugumodels.Target),
	}
	router := &llm.Router{}
	radarResp := api.ExecuteRadar(cfg, router, "", "", 1, "test intent", "en", noteFilePath)
	if radarResp.Status != "error" || !strings.Contains(radarResp.Error, "excludes note cards") {
		t.Fatalf("expected Bugu to exclude note card with error, got: %+v", radarResp)
	}

	// Also verify by docID
	radarRespDoc := api.ExecuteRadar(cfg, router, "", "", 1, "test intent", "en", docID)
	if radarRespDoc.Status != "error" || !strings.Contains(radarRespDoc.Error, "excludes note cards") {
		t.Fatalf("expected Bugu to exclude note card by docID with error, got: %+v", radarRespDoc)
	}
}

func TestCaptureNote_WithHashTitle(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_note_title_test_*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	scatteringDir := filepath.Join(tempHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create Scattering dir: %v", err)
	}

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}

	wStorage := &WatcherStorage{}
	err = watcher.GetWatcher().Initialize([]string{scatteringDir}, 50, wStorage, tempHome)
	if err != nil {
		t.Fatalf("failed to initialize watcher: %v", err)
	}

	// 1. Test Header # Title
	headerText := "# Redis启动命令\ndocker run -d --name redis -p 6379:6379 redis:alpine"
	docID1, err := app.CaptureNote(headerText)
	if err != nil {
		t.Fatalf("CaptureNote with header title failed: %v", err)
	}

	// 2. Test Footer # Title
	footerText := "kill -9 $(lsof -t -i:8080)\n# 强制释放端口"
	docID2, err := app.CaptureNote(footerText)
	if err != nil {
		t.Fatalf("CaptureNote with footer title failed: %v", err)
	}

	// 3. Test Relaxed Title Without Space (#Redis启动)
	noSpaceText := "#Redis启动\ndocker start redis"
	docID3, err := app.CaptureNote(noSpaceText)
	if err != nil {
		t.Fatalf("CaptureNote with no-space title failed: %v", err)
	}

	// 4. Test Relaxed Title With Indentation (   # 缩进标题)
	indentedText := "   # 缩进标题\necho 'hello'"
	docID4, err := app.CaptureNote(indentedText)
	if err != nil {
		t.Fatalf("CaptureNote with indented title failed: %v", err)
	}

	loadedCards := app.LoadLocalCards()
	var card1, card2, card3, card4 *Card
	for i := range loadedCards {
		if loadedCards[i].ID == docID1 {
			card1 = &loadedCards[i]
		}
		if loadedCards[i].ID == docID2 {
			card2 = &loadedCards[i]
		}
		if loadedCards[i].ID == docID3 {
			card3 = &loadedCards[i]
		}
		if loadedCards[i].ID == docID4 {
			card4 = &loadedCards[i]
		}
	}

	if card1 == nil || card2 == nil || card3 == nil || card4 == nil {
		t.Fatalf("expected all cards to be loaded, got card1=%v, card2=%v, card3=%v, card4=%v", card1, card2, card3, card4)
	}

	// Verify Card 1
	if card1.Title != "Redis启动命令" {
		t.Fatalf("expected card1 title 'Redis启动命令', got %q", card1.Title)
	}
	if strings.Contains(card1.Content, "# Redis启动命令") {
		t.Fatalf("expected # title line to be stripped from content, got: %q", card1.Content)
	}
	if card1.Content != "docker run -d --name redis -p 6379:6379 redis:alpine" {
		t.Fatalf("expected content %q, got %q", "docker run -d --name redis -p 6379:6379 redis:alpine", card1.Content)
	}

	// Verify Card 2
	if card2.Title != "强制释放端口" {
		t.Fatalf("expected card2 title '强制释放端口', got %q", card2.Title)
	}
	if strings.Contains(card2.Content, "# 强制释放端口") {
		t.Fatalf("expected # title line to be stripped from content, got: %q", card2.Content)
	}
	if card2.Content != "kill -9 $(lsof -t -i:8080)" {
		t.Fatalf("expected content %q, got %q", "kill -9 $(lsof -t -i:8080)", card2.Content)
	}

	// Verify Card 3 (#Redis启动)
	if card3.Title != "Redis启动" {
		t.Fatalf("expected card3 title 'Redis启动', got %q", card3.Title)
	}
	if strings.Contains(card3.Content, "#Redis启动") {
		t.Fatalf("expected # title line to be stripped from content, got: %q", card3.Content)
	}
	if card3.Content != "docker start redis" {
		t.Fatalf("expected content %q, got %q", "docker start redis", card3.Content)
	}

	// Verify Card 4 (   # 缩进标题)
	if card4.Title != "缩进标题" {
		t.Fatalf("expected card4 title '缩进标题', got %q", card4.Title)
	}
	if strings.Contains(card4.Content, "# 缩进标题") {
		t.Fatalf("expected # title line to be stripped from content, got: %q", card4.Content)
	}
	if card4.Content != "echo 'hello'" {
		t.Fatalf("expected content %q, got %q", "echo 'hello'", card4.Content)
	}
}
