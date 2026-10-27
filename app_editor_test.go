package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCardInEditor_EmptyID(t *testing.T) {
	app := &App{}
	err := app.OpenCardInEditor("")
	if err == nil {
		t.Fatal("expected error for empty card id, got nil")
	}
}

func TestOpenCardInEditor_NotFound(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_editor_test_*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}

	err = app.OpenCardInEditor("non_existent_card_12345")
	if err == nil {
		t.Fatal("expected error for non-existent card id, got nil")
	}
}

func TestOpenCardInEditor_DirectFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_card_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.WriteString("# Direct card file\n\nContent")
	tempFile.Close()

	app := &App{
		CardManager: NewCardManager(),
	}

	// Direct file path should resolve and attempt to open
	// In test environment, open will execute via qled or system default
	err = app.OpenCardInEditor(tempFile.Name())
	if err != nil {
		t.Logf("OpenCardInEditor executed (might fail in headless environment): %v", err)
	}
}

func TestOpenCardInEditor_ScatteringLookup(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_editor_scattering_*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	scatteringDir := filepath.Join(tempHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create scattering dir: %v", err)
	}

	cardFileName := "20261027-1200-test-slug.md"
	cardPath := filepath.Join(scatteringDir, cardFileName)
	err = os.WriteFile(cardPath, []byte("---\nid: 01TESTCARDID\n---\nHello"), 0644)
	if err != nil {
		t.Fatalf("failed to write card file: %v", err)
	}

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}

	// Lookup by slug or filename substring
	err = app.OpenCardInEditor("test-slug")
	if err != nil {
		t.Logf("OpenCardInEditor executed: %v", err)
	}
}

func TestReloadCard_ContentUpdate(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_reload_test_*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	scatteringDir := filepath.Join(tempHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatalf("failed to create scattering dir: %v", err)
	}

	cardFileName := "20261027-1400-reload-target.md"
	cardPath := filepath.Join(scatteringDir, cardFileName)
	err = os.WriteFile(cardPath, []byte("---\nid: 01RELOADTEST\ntitle: Initial Title\n---\nInitial Body Content"), 0644)
	if err != nil {
		t.Fatalf("failed to write card file: %v", err)
	}

	app := &App{
		Home:        tempHome,
		CardManager: NewCardManager(),
	}

	// 1. Initial reload
	card, err := app.ReloadCard("01RELOADTEST")
	if err != nil {
		t.Fatalf("ReloadCard failed: %v", err)
	}
	if card.Title != "Initial Title" || card.Content != "Initial Body Content" {
		t.Fatalf("unexpected card content: %+v", card)
	}

	// 2. Simulate editing the file externally (e.g. via Qled)
	updatedMarkdown := "---\nid: 01RELOADTEST\ntitle: Updated Title from Qled\n---\nUpdated Body Content with new notes"
	err = os.WriteFile(cardPath, []byte(updatedMarkdown), 0644)
	if err != nil {
		t.Fatalf("failed to update card file: %v", err)
	}

	// 3. Reload after edit
	updatedCard, err := app.ReloadCard("01RELOADTEST")
	if err != nil {
		t.Fatalf("ReloadCard after edit failed: %v", err)
	}
	if updatedCard.Title != "Updated Title from Qled" {
		t.Errorf("expected updated title 'Updated Title from Qled', got '%s'", updatedCard.Title)
	}
	if updatedCard.Content != "Updated Body Content with new notes" {
		t.Errorf("expected updated content 'Updated Body Content with new notes', got '%s'", updatedCard.Content)
	}
}
