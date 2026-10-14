package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
)

func TestCardImagesSerialization(t *testing.T) {
	c := Card{
		ID:     "test-card",
		Type:   CardTypeThought,
		Title:  "Thought with images",
		Images: []string{"img1.png", "img2.jpg"},
	}
	if len(c.Images) != 2 || c.Images[0] != "img1.png" || c.Images[1] != "img2.jpg" {
		t.Fatalf("unexpected card images: %v", c.Images)
	}
}

func TestLoadLocalCardsImages(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_home_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	scatteringDir := filepath.Join(tempHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatal(err)
	}

	mdContent := `---
id: test-card-001
type: thought
created: 2026-10-14T01:00:00Z
updated: 2026-10-14T01:00:00Z
source: local
tags: []
publish: false
images:
  - 1728863456789-abc12345.png
  - 1728863456790-def67890.jpg
---

This is a test thought with images.
`
	if err := os.WriteFile(filepath.Join(scatteringDir, "20261014-0100-test.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	app := &App{Home: tempHome}
	cards := app.LoadLocalCards()
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	card := cards[0]
	if len(card.Images) != 2 {
		t.Fatalf("expected 2 images, got %d: %v", len(card.Images), card.Images)
	}
	if card.Images[0] != "1728863456789-abc12345.png" || card.Images[1] != "1728863456790-def67890.jpg" {
		t.Fatalf("mismatched images: %v", card.Images)
	}
	if card.Content != "This is a test thought with images." {
		t.Fatalf("unexpected content: %q", card.Content)
	}
}

func TestCaptureThoughtWithImages(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_home_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	app := &App{Home: tempHome}

	// 1x1 transparent PNG bytes
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	b64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)

	// Subscribe to event bus
	ch := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_listener")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_listener")

	images := []ImagePayload{
		{
			Data: b64Data,
			Name: "test.png",
			Ext:  "png",
		},
	}

	docID, err := app.CaptureThoughtWithImages("Testing thought with image capture", images)
	if err != nil {
		t.Fatalf("CaptureThoughtWithImages failed: %v", err)
	}
	if docID == "" {
		t.Fatal("expected non-empty docID")
	}

	// Verify asset file was saved
	assetsDir := filepath.Join(tempHome, "assets")
	files, err := os.ReadDir(assetsDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("expected 1 asset file in %s, got %v (err: %v)", assetsDir, files, err)
	}
	savedFileName := files[0].Name()
	if !strings.HasSuffix(savedFileName, ".png") || !strings.Contains(savedFileName, "-") {
		t.Fatalf("unexpected asset filename format: %s", savedFileName)
	}

	savedBytes, err := os.ReadFile(filepath.Join(assetsDir, savedFileName))
	if err != nil || len(savedBytes) != len(pngBytes) {
		t.Fatalf("saved bytes mismatch: len %d vs expected %d", len(savedBytes), len(pngBytes))
	}

	// Verify markdown file in Scattering
	scatteringDir := filepath.Join(tempHome, "Scattering")
	scatFiles, err := os.ReadDir(scatteringDir)
	if err != nil || len(scatFiles) != 1 {
		t.Fatalf("expected 1 scattering file, got %v", scatFiles)
	}
	mdData, err := os.ReadFile(filepath.Join(scatteringDir, scatFiles[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	mdStr := string(mdData)
	if !strings.Contains(mdStr, "images:") || !strings.Contains(mdStr, savedFileName) {
		t.Fatalf("markdown frontmatter does not contain saved image name:\n%s", mdStr)
	}

	// Verify bus event
	select {
	case event := <-ch:
		data, ok := event.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("unexpected event data type: %T", event.Data)
		}
		if data.DocID != docID {
			t.Fatalf("event docID mismatch: %s vs %s", data.DocID, docID)
		}
		if len(data.Images) != 1 || data.Images[0] != savedFileName {
			t.Fatalf("event images mismatch: %v vs %s", data.Images, savedFileName)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}

	// Verify LoadLocalCards reads it back
	loadedCards := app.LoadLocalCards()
	if len(loadedCards) != 1 {
		t.Fatalf("expected 1 loaded card, got %d", len(loadedCards))
	}
	if len(loadedCards[0].Images) != 1 || loadedCards[0].Images[0] != savedFileName {
		t.Fatalf("LoadLocalCards failed to parse saved images: %v", loadedCards[0].Images)
	}
}

func TestAssetServerHTTPHandler(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "finnegan_home_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	assetsDir := filepath.Join(tempHome, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sampleContent := []byte("fake-png-content-data")
	if err := os.WriteFile(filepath.Join(assetsDir, "sample.png"), sampleContent, 0644); err != nil {
		t.Fatal(err)
	}
	sampleWebp := []byte("fake-webp-content-data")
	if err := os.WriteFile(filepath.Join(assetsDir, "sample.webp"), sampleWebp, 0644); err != nil {
		t.Fatal(err)
	}

	app := &App{Home: tempHome}
	handler := createAssetHandler(app)

	// 1. Success request for PNG
	req := httptest.NewRequest(http.MethodGet, "/api/assets/sample.png", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Fatalf("expected image/png content type, got %s", ct)
	}
	if rr.Body.String() != string(sampleContent) {
		t.Fatalf("body mismatch: %s", rr.Body.String())
	}

	// 2. Success request for WEBP
	req = httptest.NewRequest(http.MethodGet, "/api/assets/sample.webp", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/webp") {
		t.Fatalf("expected image/webp content type, got %s", ct)
	}

	// 3. Path traversal attack blocked
	req = httptest.NewRequest(http.MethodGet, "/api/assets/../etc/passwd", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden for path traversal, got %d", rr.Code)
	}

	// 4. Nonexistent file returns 404
	req = httptest.NewRequest(http.MethodGet, "/api/assets/notfound.jpg", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}

	// 5. Non /api/assets/ path returns 404
	req = httptest.NewRequest(http.MethodGet, "/other/path", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}

	// 6. Method not allowed for POST
	req = httptest.NewRequest(http.MethodPost, "/api/assets/sample.png", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", rr.Code)
	}
}
