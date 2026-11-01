package babel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Finnegan/core/models"
)

var (
	PublicAPIEndpoint   = "https://api.bsky.app/xrpc"
	FallbackAPIEndpoint = "https://public.api.bsky.app/xrpc"
	HTTPClient          = &http.Client{Timeout: 10 * time.Second}
)

type bskyThreadResponse struct {
	Thread struct {
		Type    string `json:"$type"`
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

func extractRkey(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// NormalizeToAtURI converts web bsky URLs (https://bsky.app/profile/{handle}/post/{rkey})
// to AT URIs (at://{handle}/app.bsky.feed.post/{rkey}) so getPostThread accepts it.
func NormalizeToAtURI(uri string) string {
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

func executeGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	client := HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
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

// FetchBskyPostComments retrieves the thread of replies for a given Bluesky Post URI or web URL
// and returns them as normalized core/models.BabelCard structs.
func FetchBskyPostComments(postURI string, parentCardID string) ([]models.BabelCard, error) {
	targetURI := NormalizeToAtURI(postURI)
	path := fmt.Sprintf("/app.bsky.feed.getPostThread?uri=%s&depth=1", url.QueryEscape(targetURI))

	body, err := executeGet(PublicAPIEndpoint + path)
	if err != nil && FallbackAPIEndpoint != "" && FallbackAPIEndpoint != PublicAPIEndpoint {
		if fbBody, fbErr := executeGet(FallbackAPIEndpoint + path); fbErr == nil {
			body = fbBody
			err = nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thread: %w", err)
	}

	var threadResp bskyThreadResponse
	if err := json.Unmarshal(body, &threadResp); err != nil {
		return nil, fmt.Errorf("failed to parse thread: %w", err)
	}

	var comments []models.BabelCard
	for _, reply := range threadResp.Thread.Replies {
		if reply.Type != "app.bsky.feed.defs#threadViewPost" {
			continue
		}
		rkey := extractRkey(reply.Post.URI)
		comments = append(comments, models.BabelCard{
			Origin:       "atproto",
			Title:        "Bluesky Echo",
			Content:      fmt.Sprintf("> Reply from @%s\n\n%s", reply.Post.Author.Handle, reply.Post.Record.Text),
			SourceURL:    fmt.Sprintf("https://bsky.app/profile/%s/post/%s", reply.Post.Author.Handle, rkey),
			AuthorName:   reply.Post.Author.Handle,
			AuthorAvatar: reply.Post.Author.Avatar,
			OriginalText: reply.Post.Record.Text,
			EchoOf:       parentCardID,
			ReplyCount:   reply.Post.ReplyCount,
			RepostCount:  reply.Post.RepostCount,
			LikeCount:    reply.Post.LikeCount,
			CreatedAt:    time.Now(),
		})
	}

	return comments, nil
}
