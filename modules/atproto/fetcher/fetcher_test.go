package fetcher_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"Finnegan/modules/atproto/fetcher"
)

func TestSearchSimilar(t *testing.T) {
	mockResponse := fetcher.BSkySearchResponse{}

	// Create mock post response
	post1 := struct {
		URI    string `json:"uri"`
		CID    string `json:"cid"`
		Author struct {
			Handle string `json:"handle"`
			DID    string `json:"did"`
			Avatar string `json:"avatar"`
		} `json:"author"`
		Record struct {
			Text      string `json:"text"`
			CreatedAt string `json:"createdAt"`
		} `json:"record"`
	}{
		URI: "at://did:plc:foo/app.bsky.feed.post/3kxxxxx",
		CID: "bafyreihxxxx",
		Author: struct {
			Handle string `json:"handle"`
			DID    string `json:"did"`
			Avatar string `json:"avatar"`
		}{
			Handle: "testuser.bsky.social",
			DID:    "did:plc:foo",
			Avatar: "https://avatar.url/avatar.jpg",
		},
		Record: struct {
			Text      string `json:"text"`
			CreatedAt string `json:"createdAt"`
		}{
			Text:      "Test post content",
			CreatedAt: "2023-10-27T10:00:00Z",
		},
	}
	mockResponse.Posts = append(mockResponse.Posts, post1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app.bsky.feed.searchPosts" {
			t.Errorf("Expected path /app.bsky.feed.searchPosts, got %s", r.URL.Path)
		}

		q := r.URL.Query().Get("q")
		if q != "hello world" {
			t.Errorf("Expected query 'hello world', got '%s'", q)
		}

		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("Expected limit '10', got '%s'", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	f := fetcher.NewFetcher()
	f.Endpoint = server.URL

	results, err := f.SearchSimilar([]string{"hello", "world"}, 10, "doc-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.Type != "babel" {
		t.Errorf("Expected Type 'babel', got %s", res.Type)
	}
	if res.Origin != "atproto" {
		t.Errorf("Expected Origin 'atproto', got %s", res.Origin)
	}
	if res.SourceURL != "https://bsky.app/profile/testuser.bsky.social/post/3kxxxxx" {
		t.Errorf("Expected SourceURL 'https://bsky.app/profile/testuser.bsky.social/post/3kxxxxx', got %s", res.SourceURL)
	}
	if res.AuthorName != "testuser.bsky.social" {
		t.Errorf("Expected AuthorName 'testuser.bsky.social', got %s", res.AuthorName)
	}
	if res.AuthorAvatar != "https://avatar.url/avatar.jpg" {
		t.Errorf("Expected AuthorAvatar 'https://avatar.url/avatar.jpg', got %s", res.AuthorAvatar)
	}
	if res.Content != "Test post content" {
		t.Errorf("Expected Content 'Test post content', got %s", res.Content)
	}
	if res.OriginalText != "Test post content" {
		t.Errorf("Expected OriginalText 'Test post content', got %s", res.OriginalText)
	}
	if res.EchoOf != "doc-123" {
		t.Errorf("Expected EchoOf 'doc-123', got %s", res.EchoOf)
	}
}

func TestFetchComments(t *testing.T) {
	mockResponse := fetcher.BSkyThreadResponse{}

	reply1 := struct {
		Type string `json:"$type"`
		Post struct {
			URI    string `json:"uri"`
			Author struct {
				Handle string `json:"handle"`
				Avatar string `json:"avatar"`
			} `json:"author"`
			Record struct {
				Text string `json:"text"`
			} `json:"record"`
		} `json:"post"`
	}{
		Type: "app.bsky.feed.defs#threadViewPost",
		Post: struct {
			URI    string `json:"uri"`
			Author struct {
				Handle string `json:"handle"`
				Avatar string `json:"avatar"`
			} `json:"author"`
			Record struct {
				Text string `json:"text"`
			} `json:"record"`
		}{
			URI: "at://did:plc:bar/app.bsky.feed.post/4lyyyyy",
			Author: struct {
				Handle string `json:"handle"`
				Avatar string `json:"avatar"`
			}{
				Handle: "replyuser.bsky.social",
				Avatar: "https://avatar.url/reply.jpg",
			},
			Record: struct {
				Text string `json:"text"`
			}{
				Text: "This is a reply",
			},
		},
	}
	mockResponse.Thread.Replies = append(mockResponse.Thread.Replies, reply1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app.bsky.feed.getPostThread" {
			t.Errorf("Expected path /app.bsky.feed.getPostThread, got %s", r.URL.Path)
		}

		uri := r.URL.Query().Get("uri")
		if uri != "at://did:plc:foo/app.bsky.feed.post/3kxxxxx" {
			t.Errorf("Expected uri 'at://did:plc:foo/app.bsky.feed.post/3kxxxxx', got '%s'", uri)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	f := fetcher.NewFetcher()
	f.Endpoint = server.URL

	results, err := f.FetchComments("at://did:plc:foo/app.bsky.feed.post/3kxxxxx", "parent-456")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.Type != "babel" {
		t.Errorf("Expected Type 'babel', got %s", res.Type)
	}
	if res.Origin != "atproto" {
		t.Errorf("Expected Origin 'atproto', got %s", res.Origin)
	}
	if res.SourceURL != "https://bsky.app/profile/replyuser.bsky.social/post/4lyyyyy" {
		t.Errorf("Expected SourceURL 'https://bsky.app/profile/replyuser.bsky.social/post/4lyyyyy', got %s", res.SourceURL)
	}
	if res.AuthorName != "replyuser.bsky.social" {
		t.Errorf("Expected AuthorName 'replyuser.bsky.social', got %s", res.AuthorName)
	}
	if res.AuthorAvatar != "https://avatar.url/reply.jpg" {
		t.Errorf("Expected AuthorAvatar 'https://avatar.url/reply.jpg', got %s", res.AuthorAvatar)
	}
	if res.Content != "This is a reply" {
		t.Errorf("Expected Content 'This is a reply', got %s", res.Content)
	}
	if res.OriginalText != "This is a reply" {
		t.Errorf("Expected OriginalText 'This is a reply', got %s", res.OriginalText)
	}
	if res.EchoOf != "parent-456" {
		t.Errorf("Expected EchoOf 'parent-456', got %s", res.EchoOf)
	}
}
