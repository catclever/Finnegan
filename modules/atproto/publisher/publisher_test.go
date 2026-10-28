package publisher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/models"
)

func TestParseFacets(t *testing.T) {
	text := "Hello @user.bsky.social check this out: https://example.com and #test tag"
	facets := ParseFacets(text)

	if len(facets) != 3 {
		t.Fatalf("expected 3 facets, got %d", len(facets))
	}

	// Order might vary since we iterate over distinct matches, let's map them
	foundMention := false
	foundLink := false
	foundTag := false

	for _, f := range facets {
		for _, feat := range f.Features {
			switch feat.Type {
			case "app.bsky.richtext.facet#mention":
				foundMention = true
				if feat.DID != "did:mock:user.bsky.social" {
					t.Errorf("expected DID did:mock:user.bsky.social, got %s", feat.DID)
				}
			case "app.bsky.richtext.facet#link":
				foundLink = true
				if feat.URI != "https://example.com" {
					t.Errorf("expected URI https://example.com, got %s", feat.URI)
				}
			case "app.bsky.richtext.facet#tag":
				foundTag = true
				if feat.Tag != "test" {
					t.Errorf("expected Tag test, got %s", feat.Tag)
				}
			}
		}
	}

	if !foundMention || !foundLink || !foundTag {
		t.Errorf("missing expected facets")
	}
}

func TestUpdateFrontmatter(t *testing.T) {
	content := `---
title: Test Note
publish: true
# A useful comment
---
Some body text
`
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := updateFrontmatter(filePath, "at://did:plc:123/app.bsky.feed.post/456", "bafyreib")
	if err != nil {
		t.Fatal(err)
	}

	newContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	newContentStr := string(newContent)

	if !strings.Contains(newContentStr, "uri: at://did:plc:123/app.bsky.feed.post/456") {
		t.Errorf("missing uri update")
	}
	if !strings.Contains(newContentStr, "cid: bafyreib") {
		t.Errorf("missing cid update")
	}
	if !strings.Contains(newContentStr, "published: true") {
		t.Errorf("missing published status update")
	}
	if !strings.Contains(newContentStr, "# A useful comment") {
		t.Errorf("lost comment in frontmatter")
	}
	if !strings.Contains(newContentStr, "Some body text") {
		t.Errorf("lost body text")
	}
}

func TestPublisher_PublishNew(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(SessionResponse{
			AccessJwt:  "mock-access",
			RefreshJwt: "mock-refresh",
			Did:        "did:plc:123",
		})
	})
	mux.HandleFunc("/xrpc/com.atproto.repo.createRecord", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-access" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"uri": "at://did:plc:123/app.bsky.feed.post/rkey1",
			"cid": "bafyreibmock",
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	cfg := &config.Config{
		Handle:   "test.bsky.social",
		Password: "password",
		PDSUrl:   ts.URL,
	}

	pub := NewPublisher(cfg)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	os.WriteFile(filePath, []byte("---\npublish: true\n---\nHello World"), 0644)

	post := &models.Post{
		Content: "Hello World",
		FilePath: filePath,
	}

	err := pub.PublishNew(post)
	if err != nil {
		t.Fatal(err)
	}

	newContent, _ := os.ReadFile(filePath)
	if !strings.Contains(string(newContent), "uri: at://did:plc:123/app.bsky.feed.post/rkey1") {
		t.Errorf("expected frontmatter to be updated with new uri")
	}
}
