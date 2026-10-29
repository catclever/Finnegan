package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Finnegan/modules/atproto/models"
)

// Primary and fallback endpoints for public XRPC reads.
// api.bsky.app connects directly to Bluesky AppView OpenResty, bypassing BunnyCDN 403 blocks.
// public.api.bsky.app serves as an alternate fallback.
var (
	PublicAPIEndpoint   = "https://api.bsky.app/xrpc"
	FallbackAPIEndpoint = "https://public.api.bsky.app/xrpc"
)

// BSkySearchResponse represents the minimal JSON response from app.bsky.feed.searchPosts
type BSkySearchResponse struct {
	Posts []struct {
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
	} `json:"posts"`
}

// BSkyThreadResponse represents the minimal JSON response from app.bsky.feed.getPostThread
type BSkyThreadResponse struct {
	Thread struct {
		Type   string `json:"$type"`
		Post   struct {
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
		Replies []struct {
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
		} `json:"replies"`
	} `json:"thread"`
}

// Fetcher handles read-only (Discovery) requests to the AT Protocol
type Fetcher struct {
	HTTPClient *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// extractRkey extracts the record key from an AT URI (e.g. at://did/collection/rkey)
func extractRkey(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// executeGet sends an HTTP GET request with standard headers and validates status 200 OK.
func (f *Fetcher) executeGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, snippet)
	}

	return body, nil
}

// SearchSimilar fetches posts matching a semantic keyword query
// Returns them mapped to Finnegan's internal Post model for saving to Inbox
func (f *Fetcher) SearchSimilar(keywords []string, limit int, echoOfDocID string) ([]models.Post, error) {
	var query string
	if len(keywords) == 1 {
		query = keywords[0]
	} else if len(keywords) > 1 {
		query = strings.Join(keywords, " | ")
	}
	path := fmt.Sprintf("/app.bsky.feed.searchPosts?q=%s&limit=%d", url.QueryEscape(query), limit)

	body, err := f.executeGet(PublicAPIEndpoint + path)
	if err != nil && FallbackAPIEndpoint != "" && FallbackAPIEndpoint != PublicAPIEndpoint {
		if fbBody, fbErr := f.executeGet(FallbackAPIEndpoint + path); fbErr == nil {
			body = fbBody
			err = nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}

	var searchResp BSkySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	var results []models.Post
	for _, p := range searchResp.Posts {
		rkey := extractRkey(p.URI)
		// Convert external post to our local model
		post := models.Post{
			Metadata: models.PostMetadata{
				URI:  p.URI,
				CID:  p.CID,
				Tags: []string{"#External", "#Discovery"},
			},
			Content:      fmt.Sprintf("> [@%s](https://bsky.app/profile/%s)\n\n%s", p.Author.Handle, p.Author.Handle, p.Record.Text),
			Type:         "babel",
			Origin:       "atproto",
			SourceURL:    fmt.Sprintf("https://bsky.app/profile/%s/post/%s", p.Author.Handle, rkey),
			AuthorName:   p.Author.Handle,
			AuthorAvatar: p.Author.Avatar,
			OriginalText: p.Record.Text,
			EchoOf:       echoOfDocID,
			ReplyCount:   p.ReplyCount,
			RepostCount:  p.RepostCount,
			LikeCount:    p.LikeCount,
			// FilePath will be determined by the caller (Inbox logic)
		}
		results = append(results, post)
	}

	return results, nil
}

// normalizeToAtURI converts web bsky URLs (https://bsky.app/profile/{handle}/post/{rkey})
// to AT URIs (at://{handle}/app.bsky.feed.post/{rkey}) so getPostThread accepts it.
func normalizeToAtURI(uri string) string {
	if strings.HasPrefix(uri, "at://") {
		return uri
	}
	if strings.Contains(uri, "bsky.app/profile/") {
		trimmed := strings.TrimPrefix(uri, "https://")
		trimmed = strings.TrimPrefix(trimmed, "http://")
		parts := strings.Split(trimmed, "/")
		for i, part := range parts {
			if part == "profile" && i+3 < len(parts) && parts[i+2] == "post" {
				handle := parts[i+1]
				rkey := parts[i+3]
				return fmt.Sprintf("at://%s/app.bsky.feed.post/%s", handle, rkey)
			}
		}
	}
	return uri
}

// FetchComments retrieves the thread of replies for a given Post URI
func (f *Fetcher) FetchComments(postURI string, parentCardID string) ([]models.Post, error) {
	targetURI := normalizeToAtURI(postURI)
	path := fmt.Sprintf("/app.bsky.feed.getPostThread?uri=%s", url.QueryEscape(targetURI))

	body, err := f.executeGet(PublicAPIEndpoint + path)
	if err != nil && FallbackAPIEndpoint != "" && FallbackAPIEndpoint != PublicAPIEndpoint {
		if fbBody, fbErr := f.executeGet(FallbackAPIEndpoint + path); fbErr == nil {
			body = fbBody
			err = nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thread: %w", err)
	}

	var threadResp BSkyThreadResponse
	if err := json.Unmarshal(body, &threadResp); err != nil {
		return nil, fmt.Errorf("failed to parse thread: %w", err)
	}

	var comments []models.Post
	for _, reply := range threadResp.Thread.Replies {
		// Only parse valid post nodes
		if reply.Type != "app.bsky.feed.defs#threadViewPost" {
			continue
		}
		
		rkey := extractRkey(reply.Post.URI)

		comment := models.Post{
			Metadata: models.PostMetadata{
				URI:  reply.Post.URI,
				Tags: []string{"#Comment", "#Interaction"},
			},
			Content:      fmt.Sprintf("> Reply from @%s\n\n%s", reply.Post.Author.Handle, reply.Post.Record.Text),
			Type:         "babel",
			Origin:       "atproto",
			SourceURL:    fmt.Sprintf("https://bsky.app/profile/%s/post/%s", reply.Post.Author.Handle, rkey),
			AuthorName:   reply.Post.Author.Handle,
			AuthorAvatar: reply.Post.Author.Avatar,
			OriginalText: reply.Post.Record.Text,
			EchoOf:       parentCardID,
			ReplyCount:   reply.Post.ReplyCount,
			RepostCount:  reply.Post.RepostCount,
			LikeCount:    reply.Post.LikeCount,
		}
		comments = append(comments, comment)
	}

	return comments, nil
}
