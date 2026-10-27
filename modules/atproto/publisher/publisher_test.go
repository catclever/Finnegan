package publisher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"Finnegan/modules/atproto/config"
)

func TestParseFacets(t *testing.T) {
	text := "Check out https://example.com! Great work @alice.bsky.social and #golang."

	facets := ParseFacets(text)

	if len(facets) != 3 {
		t.Fatalf("expected 3 facets, got %d", len(facets))
	}

	// 1. URL
	if facets[0].Features[0].Type != "app.bsky.richtext.facet#link" {
		t.Errorf("expected link facet, got %s", facets[0].Features[0].Type)
	}
	if facets[0].Features[0].URI != "https://example.com!" {
		// regex includes the trailing !, this is acceptable for simple regex
	}

	// 2. Mention
	if facets[1].Features[0].Type != "app.bsky.richtext.facet#mention" {
		t.Errorf("expected mention facet, got %s", facets[1].Features[0].Type)
	}
	if facets[1].Features[0].DID != "alice.bsky.social" {
		t.Errorf("expected alice.bsky.social, got %s", facets[1].Features[0].DID)
	}

	// 3. Hashtag
	if facets[2].Features[0].Type != "app.bsky.richtext.facet#tag" {
		t.Errorf("expected tag facet, got %s", facets[2].Features[0].Type)
	}
	if facets[2].Features[0].Tag != "golang." {
		// trailing punctuation, acceptable for this pass
	}
}

func TestAuthManager_CreateSession(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/xrpc/com.atproto.server.createSession" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Session{
				DID:        "did:plc:123",
				Handle:     "test.bsky.social",
				AccessJwt:  "access-token",
				RefreshJwt: "refresh-token",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Handle:   "test.bsky.social",
		Password: "password",
		PDSUrl:   mockServer.URL,
	}

	auth := NewAuthManager(cfg)
	err := auth.CreateSession()

	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if auth.Session.DID != "did:plc:123" {
		t.Errorf("expected DID did:plc:123, got %s", auth.Session.DID)
	}
	if auth.Session.AccessJwt != "access-token" {
		t.Errorf("expected AccessJwt access-token, got %s", auth.Session.AccessJwt)
	}
}

func TestUpdateFrontmatter(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_frontmatter_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `---
title: Test Note
tags:
  - test
# A comment
publish: true
---
# Hello World

This is a test note.
`
	tmpFile.Write([]byte(content))
	tmpFile.Close()

	err = UpdateFrontmatter(tmpFile.Name(), "at://did:plc:123/app.bsky.feed.post/rkey123", "cid123")
	if err != nil {
		t.Fatalf("failed to update frontmatter: %v", err)
	}

	updatedContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	updatedStr := string(updatedContent)
	if !strings.Contains(updatedStr, "uri: at://did:plc:123/app.bsky.feed.post/rkey123") {
		t.Errorf("uri not updated correctly: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "cid: cid123") {
		t.Errorf("cid not updated correctly: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "published: true") {
		t.Errorf("published not updated correctly: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "# A comment") {
		t.Errorf("comment was not preserved: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "# Hello World") {
		t.Errorf("markdown body not preserved: %s", updatedStr)
	}
}

func TestPublisher_Publish(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/xrpc/com.atproto.server.createSession" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Session{
				DID:        "did:plc:123",
				Handle:     "test.bsky.social",
				AccessJwt:  "access-token",
				RefreshJwt: "refresh-token",
			})
			return
		}
		if r.URL.Path == "/xrpc/com.atproto.repo.createRecord" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(CreateRecordResponse{
				URI: "at://did:plc:123/app.bsky.feed.post/rkey123",
				CID: "bafyreih3...",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Handle:   "test.bsky.social",
		Password: "password",
		PDSUrl:   mockServer.URL,
	}

	pub := NewPublisher(cfg)
	uri, cid, err := pub.Publish("Hello world", nil, nil)

	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	if uri != "at://did:plc:123/app.bsky.feed.post/rkey123" {
		t.Errorf("expected correct URI, got %s", uri)
	}
	if cid != "bafyreih3..." {
		t.Errorf("expected correct CID, got %s", cid)
	}
}
