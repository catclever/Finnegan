package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
)

func TestCaptureCurio_TextOnly(t *testing.T) {
	tmpHome := t.TempDir()

	// Create required directories
	os.MkdirAll(filepath.Join(tmpHome, "Scattering"), 0755)

	app := &App{
		ctx:  context.Background(),
		Home: tmpHome,
	}

	eventChan := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_capture_curio")

	text := "My New Curio Request #v1.0 #wishlist\nI want to buy this item."
	docID, err := app.CaptureCurio(text, nil)
	if err != nil {
		t.Fatalf("Unexpected error from CaptureCurio: %v", err)
	}
	if docID == "" {
		t.Fatal("Expected docID, got empty string")
	}

	select {
	case evt := <-eventChan:
		data, ok := evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("Expected EventThoughtCapturedData, got %T", evt.Data)
		}
		if data.Type != "curio" {
			t.Errorf("Expected Type 'curio', got %q", data.Type)
		}
		if data.Title != "My New Curio Request" {
			t.Errorf("Expected Title 'My New Curio Request', got %q", data.Title)
		}
		if len(data.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d: %v", len(data.Tags), data.Tags)
		}

		// Verify file
		contentBytes, err := os.ReadFile(data.FilePath)
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}
		content := string(contentBytes)

		expectedLines := []string{
			"type: curio",
			"intent: wish",
			"availability: available",
			"title: My New Curio Request",
		}
		for _, line := range expectedLines {
			if !strings.Contains(content, line) {
				t.Errorf("Expected to find line %q in file content:\n%s", line, content)
			}
		}

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for EventThoughtCaptured")
	}
}

func TestCaptureCurio_ImageEmpty(t *testing.T) {
	app := &App{
		ctx:  context.Background(),
		Home: t.TempDir(),
	}

	images := []ImagePayload{}

	_, err := app.CaptureCurio("", images)
	if err == nil {
		t.Fatal("Expected error with empty text and empty images data")
	}
}

func TestCaptureCurio_ImageHappyPath(t *testing.T) {
	app := &App{
		ctx:  context.Background(),
		Home: t.TempDir(),
	}

	// Because we don't have a valid Gemini API key in the test environment,
	// IngestCurioPayload will fail during genai client creation or execution.
	// We expect an error, but we want to make sure it enters the image processing path.
	// Providing a dummy base64 string
	images := []ImagePayload{
		{
			Data: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
			Ext:  ".png",
		},
	}

	_, err := app.CaptureCurio("Test Image Ingestion", images)
	if err == nil {
		t.Fatal("Expected error because of missing Gemini API Key, got nil")
	}

	if !strings.Contains(err.Error(), "gemini api key is required") {
		t.Errorf("Expected error to mention gemini api key, got: %v", err)
	}
}
