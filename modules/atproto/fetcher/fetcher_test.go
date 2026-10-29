package fetcher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchSimilar(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app.bsky.feed.searchPosts" {
			t.Errorf("expected path /app.bsky.feed.searchPosts, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("q")
		if query != "hello | world" {
			t.Errorf("expected query 'hello | world', got %s", query)
		}
		if !strings.Contains(r.URL.RawQuery, "hello+%7C+world") {
			t.Errorf("expected raw query to contain 'hello+%%7C+world', got %s", r.URL.RawQuery)
		}

		resp := BSkySearchResponse{}
		resp.Posts = []struct {
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
			ReplyCount  int `json:"replyCount"`
			RepostCount int `json:"repostCount"`
			LikeCount   int `json:"likeCount"`
		}{
			{
				URI: "at://did:plc:foo/app.bsky.feed.post/123",
				CID: "bafyreihere",
				Author: struct {
					Handle string `json:"handle"`
					DID    string `json:"did"`
					Avatar string `json:"avatar"`
				}{
					Handle: "bob.bsky.social",
					DID:    "did:plc:foo",
					Avatar: "http://example.com/avatar.jpg",
				},
				Record: struct {
					Text      string `json:"text"`
					CreatedAt string `json:"createdAt"`
				}{
					Text:      "Test post hello world",
					CreatedAt: "2023-01-01T00:00:00Z",
				},
				ReplyCount:  3,
				RepostCount: 1,
				LikeCount:   8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// Replace the global endpoint with the mock server URL
	originalEndpoint := PublicAPIEndpoint
	originalFallback := FallbackAPIEndpoint
	PublicAPIEndpoint = mockServer.URL
	FallbackAPIEndpoint = ""
	defer func() {
		PublicAPIEndpoint = originalEndpoint
		FallbackAPIEndpoint = originalFallback
	}()

	f := NewFetcher()
	f.HTTPClient = mockServer.Client()

	results, err := f.SearchSimilar([]string{"hello", "world"}, 10, "doc123")
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	post := results[0]
	if post.Type != "babel" {
		t.Errorf("Expected Type 'babel', got %s", post.Type)
	}
	if post.Origin != "atproto" {
		t.Errorf("Expected Origin 'atproto', got %s", post.Origin)
	}
	if post.AuthorName != "bob.bsky.social" {
		t.Errorf("Expected AuthorName 'bob.bsky.social', got %s", post.AuthorName)
	}
	if post.AuthorAvatar != "http://example.com/avatar.jpg" {
		t.Errorf("Expected AuthorAvatar 'http://example.com/avatar.jpg', got %s", post.AuthorAvatar)
	}
	if post.OriginalText != "Test post hello world" {
		t.Errorf("Expected OriginalText 'Test post hello world', got %s", post.OriginalText)
	}
	if post.EchoOf != "doc123" {
		t.Errorf("Expected EchoOf 'doc123', got %s", post.EchoOf)
	}
	expectedSourceURL := "https://bsky.app/profile/bob.bsky.social/post/123"
	if post.SourceURL != expectedSourceURL {
		t.Errorf("Expected SourceURL '%s', got %s", expectedSourceURL, post.SourceURL)
	}
	if post.ReplyCount != 3 || post.RepostCount != 1 || post.LikeCount != 8 {
		t.Errorf("unexpected counts: reply=%d repost=%d like=%d", post.ReplyCount, post.RepostCount, post.LikeCount)
	}
}

func TestFetchComments(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app.bsky.feed.getPostThread" {
			t.Errorf("expected path /app.bsky.feed.getPostThread, got %s", r.URL.Path)
		}

		resp := BSkyThreadResponse{}
		resp.Thread.Replies = []struct {
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
				ReplyCount  int `json:"replyCount"`
				RepostCount int `json:"repostCount"`
				LikeCount   int `json:"likeCount"`
			} `json:"post"`
		}{
			{
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
					ReplyCount  int `json:"replyCount"`
					RepostCount int `json:"repostCount"`
					LikeCount   int `json:"likeCount"`
				}{
					URI: "at://did:plc:bar/app.bsky.feed.post/456",
					Author: struct {
						Handle string `json:"handle"`
						Avatar string `json:"avatar"`
					}{
						Handle: "alice.bsky.social",
						Avatar: "http://example.com/alice.jpg",
					},
					Record: struct {
						Text string `json:"text"`
					}{
						Text: "Reply text",
					},
					ReplyCount:  2,
					RepostCount: 4,
					LikeCount:   9,
				},
			},
			{
				Type: "app.bsky.feed.defs#blockedPost", // Should be ignored
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	originalEndpoint := PublicAPIEndpoint
	originalFallback := FallbackAPIEndpoint
	PublicAPIEndpoint = mockServer.URL
	FallbackAPIEndpoint = ""
	defer func() {
		PublicAPIEndpoint = originalEndpoint
		FallbackAPIEndpoint = originalFallback
	}()

	f := NewFetcher()
	f.HTTPClient = mockServer.Client()

	comments, err := f.FetchComments("at://someuri", "parent123")
	if err != nil {
		t.Fatalf("FetchComments failed: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("Expected 1 comment (ignoring blockedPost), got %d", len(comments))
	}

	comment := comments[0]
	if comment.Type != "babel" {
		t.Errorf("Expected Type 'babel', got %s", comment.Type)
	}
	if comment.Origin != "atproto" {
		t.Errorf("Expected Origin 'atproto', got %s", comment.Origin)
	}
	if comment.EchoOf != "parent123" {
		t.Errorf("Expected EchoOf 'parent123', got %s", comment.EchoOf)
	}
	if comment.AuthorName != "alice.bsky.social" {
		t.Errorf("Expected AuthorName 'alice.bsky.social', got %s", comment.AuthorName)
	}
	if comment.OriginalText != "Reply text" {
		t.Errorf("Expected OriginalText 'Reply text', got %s", comment.OriginalText)
	}
	expectedSourceURL := "https://bsky.app/profile/alice.bsky.social/post/456"
	if comment.SourceURL != expectedSourceURL {
		t.Errorf("Expected SourceURL '%s', got %s", expectedSourceURL, comment.SourceURL)
	}
	if comment.ReplyCount != 2 || comment.RepostCount != 4 || comment.LikeCount != 9 {
		t.Errorf("unexpected comment counts: reply=%d repost=%d like=%d", comment.ReplyCount, comment.RepostCount, comment.LikeCount)
	}
}

func TestSearchSimilar_HTTPErrorHandling(t *testing.T) {
	// Simulate CDN 403 Forbidden with HTML body
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html><head><title>403 Forbidden</title></head><body><h1>403</h1></body></html>"))
	}))
	defer mockServer.Close()

	originalEndpoint := PublicAPIEndpoint
	originalFallback := FallbackAPIEndpoint
	PublicAPIEndpoint = mockServer.URL
	FallbackAPIEndpoint = ""
	defer func() {
		PublicAPIEndpoint = originalEndpoint
		FallbackAPIEndpoint = originalFallback
	}()

	f := NewFetcher()
	f.HTTPClient = mockServer.Client()

	_, err := f.SearchSimilar([]string{"test"}, 5, "doc123")
	if err == nil {
		t.Fatalf("expected error on 403 response, got nil")
	}

	// Should report clean HTTP error rather than JSON unmarshal `<` syntax error
	errMsg := err.Error()
	if !strings.Contains(errMsg, "HTTP error 403") {
		t.Errorf("expected 'HTTP error 403' in error message, got: %s", errMsg)
	}
}

func TestSearchSimilar_FallbackToSecondaryEndpoint(t *testing.T) {
	// Primary fails with 403, Secondary succeeds with valid JSON
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html>Forbidden</html>"))
	}))
	defer primaryServer.Close()

	secondaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := BSkySearchResponse{}
		resp.Posts = []struct {
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
			ReplyCount  int `json:"replyCount"`
			RepostCount int `json:"repostCount"`
			LikeCount   int `json:"likeCount"`
		}{
			{
				URI: "at://did:plc:fb/app.bsky.feed.post/999",
				CID: "cid-fb",
				Author: struct {
					Handle string `json:"handle"`
					DID    string `json:"did"`
					Avatar string `json:"avatar"`
				}{
					Handle: "fallback.bsky.social",
				},
				Record: struct {
					Text      string `json:"text"`
					CreatedAt string `json:"createdAt"`
				}{
					Text: "Fallback success post",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer secondaryServer.Close()

	originalEndpoint := PublicAPIEndpoint
	originalFallback := FallbackAPIEndpoint
	PublicAPIEndpoint = primaryServer.URL
	FallbackAPIEndpoint = secondaryServer.URL
	defer func() {
		PublicAPIEndpoint = originalEndpoint
		FallbackAPIEndpoint = originalFallback
	}()

	f := NewFetcher()
	f.HTTPClient = &http.Client{}

	results, err := f.SearchSimilar([]string{"fallback"}, 5, "doc123")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}
	if len(results) != 1 || results[0].AuthorName != "fallback.bsky.social" {
		t.Fatalf("unexpected results from fallback: %+v", results)
	}
}
