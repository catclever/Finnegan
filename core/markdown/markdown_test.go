package markdown

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFrontmatterAndBody(t *testing.T) {
	content := []byte(`---
id: custom-id
title: Custom Title
created: 2026-10-10T10:00:00Z
bugu: true
---
# Actual Heading
Body content`)

	doc, err := Parse("test.md", content, time.Time{})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if doc.ID != "custom-id" {
		t.Errorf("Expected id custom-id, got %s", doc.ID)
	}
	if doc.Title != "Custom Title" {
		t.Errorf("Expected title Custom Title, got %s", doc.Title)
	}
	if doc.CreatedAt.Format(time.RFC3339) != "2026-10-10T10:00:00Z" {
		t.Errorf("Expected created 2026-10-10T10:00:00Z, got %v", doc.CreatedAt)
	}
	if doc.Meta["bugu"] != true {
		t.Errorf("Expected meta bugu=true, got %v", doc.Meta["bugu"])
	}
	if doc.Body != "# Actual Heading\nBody content" {
		t.Errorf("Unexpected body: %s", doc.Body)
	}
}

func TestIDDerivation(t *testing.T) {
	content := []byte(`No frontmatter here
Just a simple text file.`)
	path := "some/path/file.md"
	abs, _ := filepath.Abs(path)
	hash := sha1.Sum([]byte(abs))
	expectedID := hex.EncodeToString(hash[:])

	doc, _ := Parse(path, content, time.Time{})
	if doc.ID != expectedID {
		t.Errorf("Expected derived ID %s, got %s", expectedID, doc.ID)
	}
}
