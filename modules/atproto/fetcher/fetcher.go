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

var PublicAPIEndpoint = "https://public.api.bsky.app/xrpc"

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

// SearchSimilar fetches posts matching a semantic keyword query
// Returns them mapped to Finnegan's internal Post model for saving to Inbox
func (f *Fetcher) SearchSimilar(keywords []string, limit int, echoOfDocID string) ([]models.Post, error) {
	query := strings.Join(keywords, " ")
	reqURL := fmt.Sprintf("%s/app.bsky.feed.searchPosts?q=%s&limit=%d", PublicAPIEndpoint, url.QueryEscape(query), limit)
	
	resp, err := f.HTTPClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var searchResp BSkySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %v", err)
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
	reqURL := fmt.Sprintf("%s/app.bsky.feed.getPostThread?uri=%s", PublicAPIEndpoint, url.QueryEscape(targetURI))
	
	resp, err := f.HTTPClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thread: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var threadResp BSkyThreadResponse
	if err := json.Unmarshal(body, &threadResp); err != nil {
		return nil, fmt.Errorf("failed to parse thread: %v", err)
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
