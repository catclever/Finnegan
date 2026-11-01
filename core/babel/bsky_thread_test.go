package babel

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNormalizeToAtURI(t *testing.T) {
	webURL := "https://bsky.app/profile/alice.bsky.social/post/3kxyz123"
	got := NormalizeToAtURI(webURL)
	want := "at://alice.bsky.social/app.bsky.feed.post/3kxyz123"
	if got != want {
		t.Errorf("NormalizeToAtURI(%q) = %q, want %q", webURL, got, want)
	}
}

func TestFetchBskyPostComments(t *testing.T) {
	origClient := HTTPClient
	HTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json")
			_, _ = rec.WriteString(`{
				"thread": {
					"$type": "app.bsky.feed.defs#threadViewPost",
					"replies": [
						{
							"$type": "app.bsky.feed.defs#threadViewPost",
							"post": {
								"uri": "at://did:plc:bob/app.bsky.feed.post/rkey999",
								"author": {"handle": "bob.bsky.social", "avatar": "https://cdn.bsky.app/bob.png"},
								"record": {"text": "Great point!"},
								"replyCount": 1,
								"repostCount": 2,
								"likeCount": 5
							}
						}
					]
				}
			}`)
			return rec.Result(), nil
		}),
	}
	defer func() { HTTPClient = origClient }()

	comments, err := FetchBskyPostComments("https://bsky.app/profile/alice.bsky.social/post/3kxyz123", "parent_001")
	if err != nil {
		t.Fatalf("FetchBskyPostComments failed: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	c := comments[0]
	if c.Origin != "atproto" || c.AuthorName != "bob.bsky.social" || c.EchoOf != "parent_001" || c.LikeCount != 5 {
		t.Errorf("unexpected comment card: %+v", c)
	}
}
