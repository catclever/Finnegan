package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/watcher"
)

func TestBookCard_CaptureAndLoad(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_book_test_*")
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
	defer app.CardManager.Close()

	wStorage := &WatcherStorage{}
	if err := watcher.GetWatcher().Initialize([]string{scatteringDir}, 50, wStorage, tempHome); err != nil {
		t.Fatalf("failed to initialize watcher: %v", err)
	}

	// Empty payload must fail
	if _, err := app.CaptureBook("   ", nil, "Dune", "Frank Herbert"); err == nil {
		t.Fatal("expected error for empty text and images")
	}

	eventCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_book_capture_listener")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_book_capture_listener")

	// 1x1 transparent PNG
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	imgPayload := []ImagePayload{
		{
			Name: "page.png",
			Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes),
		},
	}

	rawText := "# 恐惧与秩序\n> 恐惧是思维杀手。\n\n这段独白精准刻画了高压环境下的心理韧性。"
	docID, err := app.CaptureBook(rawText, imgPayload, " 沙丘 ", " 弗兰克·赫伯特 ")
	if err != nil {
		t.Fatalf("CaptureBook failed: %v", err)
	}
	if docID == "" {
		t.Fatal("expected non-empty docID")
	}

	// Verify EventThoughtCaptured payload
	select {
	case evt := <-eventCh:
		data, ok := evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("unexpected event data type: %T", evt.Data)
		}
		if data.DocID != docID {
			t.Fatalf("expected DocID %s, got %s", docID, data.DocID)
		}
		if data.Type != "book" {
			t.Fatalf("expected Type 'book', got %q", data.Type)
		}
		if data.Title != "恐惧与秩序" {
			t.Fatalf("expected Title '恐惧与秩序', got %q", data.Title)
		}
		if data.BookTitle != "沙丘" {
			t.Fatalf("expected BookTitle '沙丘', got %q", data.BookTitle)
		}
		if data.BookAuthor != "弗兰克·赫伯特" {
			t.Fatalf("expected BookAuthor '弗兰克·赫伯特', got %q", data.BookAuthor)
		}
		if len(data.Images) != 1 {
			t.Fatalf("expected 1 saved image, got %d", len(data.Images))
		}
		if !strings.HasSuffix(data.FilePath, ".book.md") {
			t.Fatalf("expected .book.md file path, got %q", data.FilePath)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}

	// Verify file on disk
	bookFiles, err := filepath.Glob(filepath.Join(scatteringDir, "*.book.md"))
	if err != nil || len(bookFiles) != 1 {
		t.Fatalf("expected 1 .book.md file in Scattering, got %v (err=%v)", bookFiles, err)
	}
	rawBytes, err := os.ReadFile(bookFiles[0])
	if err != nil {
		t.Fatalf("failed to read book file: %v", err)
	}
	fileStr := string(rawBytes)
	for _, needle := range []string{
		"type: book",
		`title: "恐惧与秩序"`,
		`book_title: "沙丘"`,
		`book_author: "弗兰克·赫伯特"`,
		"publish: false",
		"images:",
	} {
		if !strings.Contains(fileStr, needle) {
			t.Fatalf("expected frontmatter to contain %q, got:\n%s", needle, fileStr)
		}
	}

	// Verify LoadLocalCards round-trip
	loaded := app.LoadLocalCards()
	var found *Card
	for i := range loaded {
		if loaded[i].ID == docID {
			found = &loaded[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected LoadLocalCards to contain card %s", docID)
	}
	if found.Type != CardTypeBook {
		t.Fatalf("expected CardTypeBook, got %q", found.Type)
	}
	if found.Title != "恐惧与秩序" {
		t.Fatalf("expected Title '恐惧与秩序', got %q", found.Title)
	}
	if found.BookTitle != "沙丘" {
		t.Fatalf("expected BookTitle '沙丘', got %q", found.BookTitle)
	}
	if found.BookAuthor != "弗兰克·赫伯特" {
		t.Fatalf("expected BookAuthor '弗兰克·赫伯特', got %q", found.BookAuthor)
	}
	if len(found.Images) != 1 {
		t.Fatalf("expected 1 image on loaded card, got %d", len(found.Images))
	}
	if !strings.Contains(found.Content, "> 恐惧是思维杀手。") {
		t.Fatalf("expected blockquote preserved in content, got %q", found.Content)
	}

	// Verify SearchLocalCards matches BookTitle and BookAuthor
	byBookTitle := app.SearchLocalCards("沙丘")
	if len(byBookTitle) == 0 || byBookTitle[0].ID != docID {
		t.Fatalf("expected SearchLocalCards('沙丘') to match %s, got %+v", docID, byBookTitle)
	}
	byBookAuthor := app.SearchLocalCards("弗兰克·赫伯特")
	if len(byBookAuthor) == 0 || byBookAuthor[0].ID != docID {
		t.Fatalf("expected SearchLocalCards('弗兰克·赫伯特') to match %s, got %+v", docID, byBookAuthor)
	}

	// Verify ParseCardFile fallback title derivation for book cards
	blockquotePath := filepath.Join(scatteringDir, "quote-only.book.md")
	blockquoteMD := "---\nid: test-book-quote-1\ntype: book\nbook_title: \"百年孤独\"\nbook_author: \"马尔克斯\"\n---\n\n> 多年以后，面对行刑队。\n"
	if err := os.WriteFile(blockquotePath, []byte(blockquoteMD), 0644); err != nil {
		t.Fatalf("failed to write quote test file: %v", err)
	}
	parsedQuote, err := ParseCardFile(blockquotePath)
	if err != nil {
		t.Fatalf("ParseCardFile(blockquotePath) failed: %v", err)
	}
	if parsedQuote.Title != "多年以后，面对行刑队。" {
		t.Fatalf("expected blockquote-stripped title '多年以后，面对行刑队。', got %q", parsedQuote.Title)
	}

	emptyBodyPath := filepath.Join(scatteringDir, "empty-body.book.md")
	emptyBodyMD := "---\nid: test-book-empty-1\ntype: book\nbook_title: \"三体\"\nbook_author: \"刘慈欣\"\n---\n\n"
	if err := os.WriteFile(emptyBodyPath, []byte(emptyBodyMD), 0644); err != nil {
		t.Fatalf("failed to write empty-body test file: %v", err)
	}
	parsedEmpty, err := ParseCardFile(emptyBodyPath)
	if err != nil {
		t.Fatalf("ParseCardFile(emptyBodyPath) failed: %v", err)
	}
	if parsedEmpty.Title != "《三体》读书笔记" {
		t.Fatalf("expected fallback title '《三体》读书笔记', got %q", parsedEmpty.Title)
	}

	// Verify SearchLocalCards matches book_title/book_author even without frontmatter keywords
	byFallbackBook := app.SearchLocalCards("刘慈欣")
	matchedEmpty := false
	for _, c := range byFallbackBook {
		if c.ID == "test-book-empty-1" {
			matchedEmpty = true
			break
		}
	}
	if !matchedEmpty {
		t.Fatalf("expected SearchLocalCards('刘慈欣') to match card without keywords frontmatter, got %+v", byFallbackBook)
	}

	// Verify PromoteToLocalWithOpts preserves BookTitle and BookAuthor
	app.CardManager.InjectBabelCards([]Card{})
	app.CardManager.mutex.Lock()
	app.CardManager.tempCards["temp-book-1"] = Card{
		ID:         "temp-book-1",
		Type:       CardTypeBook,
		Title:      "认知觉醒摘录",
		Content:    "元认知是人类独有的反思能力。",
		BookTitle:  "认知觉醒",
		BookAuthor: "周岭",
		CreatedAt:  time.Now(),
	}
	app.CardManager.mutex.Unlock()

	promoted, err := app.CardManager.PromoteToLocalWithOpts("temp-book-1", tempHome, "", "")
	if err != nil {
		t.Fatalf("PromoteToLocalWithOpts failed: %v", err)
	}
	reloadedPromoted, err := ParseCardFile(promoted.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile(promoted) failed: %v", err)
	}
	if reloadedPromoted.Type != CardTypeBook || reloadedPromoted.BookTitle != "认知觉醒" || reloadedPromoted.BookAuthor != "周岭" {
		t.Fatalf("promoted book card metadata mismatch: %+v", reloadedPromoted)
	}
}

func TestBookCard_ThreadsAndKeywords(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_book_threads_test_*")
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
	defer app.CardManager.Close()

	// 1. Create a thread target card
	noteID, err := app.CaptureNote("# 复杂系统\n系统论与涌现现象的基础笔记")
	if err != nil {
		t.Fatalf("CaptureNote failed: %v", err)
	}

	eventCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_book_threads_listener")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_book_threads_listener")

	// 2. Capture a book card with @thread mention and hashtags (including duplicate of bookTitle)
	text := "@复杂系统 这本书把涌现机制讲得很透彻 #失控 #系统论"
	bookID, err := app.CaptureBook(text, nil, "失控", "凯文·凯利")
	if err != nil {
		t.Fatalf("CaptureBook failed: %v", err)
	}

	// Verify event data
	select {
	case evt := <-eventCh:
		data, ok := evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("unexpected event data type: %T", evt.Data)
		}
		if data.DocID != bookID {
			t.Fatalf("expected DocID %s, got %s", bookID, data.DocID)
		}
		if len(data.Keywords["object"]) == 0 {
			t.Fatalf("expected object keywords in event, got %+v", data.Keywords)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}

	// 3. Reload via LoadLocalCards and verify Threads, Keywords, and stripped body
	cards := app.LoadLocalCards()
	var bookCard *Card
	for i := range cards {
		if cards[i].ID == bookID {
			bookCard = &cards[i]
			break
		}
	}
	if bookCard == nil {
		t.Fatalf("book card %s not found in LoadLocalCards", bookID)
	}

	if len(bookCard.Threads) != 1 {
		t.Fatalf("expected 1 thread ref, got %d: %+v", len(bookCard.Threads), bookCard.Threads)
	}
	if bookCard.Threads[0].ID != noteID || bookCard.Threads[0].Title != "复杂系统" {
		t.Fatalf("unexpected thread ref: %+v", bookCard.Threads[0])
	}
	if strings.HasPrefix(bookCard.Content, "@复杂系统") {
		t.Fatalf("expected leading @复杂系统 mention to be stripped from body, got %q", bookCard.Content)
	}

	// Check filename does not contain '@'
	if strings.Contains(filepath.Base(bookCard.FilePath), "@") {
		t.Fatalf("expected filename not to contain '@', got %q", filepath.Base(bookCard.FilePath))
	}

	// Check 4D object anchors include bookTitle (deduplicated) and bookAuthor
	objAnchors := bookCard.Keywords["object"]
	titleCount := 0
	hasAuthor := false
	for _, kw := range objAnchors {
		clean := strings.TrimSuffix(strings.TrimSpace(kw), "*")
		if clean == "失控" {
			titleCount++
		}
		if clean == "凯文·凯利" {
			hasAuthor = true
		}
	}
	if titleCount != 1 {
		t.Fatalf("expected '失控' exactly once in object anchors (deduplicated with #失控), got %v", objAnchors)
	}
	if !hasAuthor {
		t.Fatalf("expected '凯文·凯利' in object anchors, got %v", objAnchors)
	}
}
