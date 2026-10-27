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

const PublicAPIEndpoint = "https://public.api.bsky.app/xrpc"

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
			} `json:"post"`
		} `json:"replies"`
	} `json:"thread"`
}

// Fetcher handles read-only (Discovery) requests to the AT Protocol
type Fetcher struct {
	HTTPClient *http.Client
	Endpoint   string
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Endpoint:   PublicAPIEndpoint,
	}
}

// SearchSimilar fetches posts matching a semantic keyword query
// Returns them mapped to Finnegan's internal Post model for saving to Inbox
func (f *Fetcher) SearchSimilar(keywords []string, limit int, echoOfDocID string) ([]models.Post, error) {
	query := strings.Join(keywords, " ")
	reqURL := fmt.Sprintf("%s/app.bsky.feed.searchPosts?q=%s&limit=%d", f.Endpoint, url.QueryEscape(query), limit)
	
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
		uriParts := strings.Split(p.URI, "/")
		rkey := ""
		if len(uriParts) > 0 {
			rkey = uriParts[len(uriParts)-1]
		}
		sourceURL := fmt.Sprintf("https://bsky.app/profile/%s/post/%s", p.Author.Handle, rkey)

		// Convert external post to our local model
		post := models.Post{
			Metadata: models.PostMetadata{
				URI:  p.URI,
				CID:  p.CID,
				Tags: []string{"#External", "#Discovery"},
			},
			Content:      p.Record.Text,
			Type:         "babel",
			Origin:       "atproto",
			SourceURL:    sourceURL,
			AuthorName:   p.Author.Handle,
			AuthorAvatar: p.Author.Avatar,
			OriginalText: p.Record.Text,
			EchoOf:       echoOfDocID,
			// FilePath will be determined by the caller (Inbox logic)
		}
		results = append(results, post)
	}

	return results, nil
}

// FetchComments retrieves the thread of replies for a given Post URI
func (f *Fetcher) FetchComments(postURI string, parentCardID string) ([]models.Post, error) {
	reqURL := fmt.Sprintf("%s/app.bsky.feed.getPostThread?uri=%s", f.Endpoint, url.QueryEscape(postURI))
	
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
		
		uriParts := strings.Split(reply.Post.URI, "/")
		rkey := ""
		if len(uriParts) > 0 {
			rkey = uriParts[len(uriParts)-1]
		}
		sourceURL := fmt.Sprintf("https://bsky.app/profile/%s/post/%s", reply.Post.Author.Handle, rkey)

		comment := models.Post{
			Metadata: models.PostMetadata{
				URI:  reply.Post.URI,
				Tags: []string{"#Comment", "#Interaction"},
			},
			Content:      reply.Post.Record.Text,
			Type:         "babel",
			Origin:       "atproto",
			SourceURL:    sourceURL,
			AuthorName:   reply.Post.Author.Handle,
			AuthorAvatar: reply.Post.Author.Avatar,
			OriginalText: reply.Post.Record.Text,
			EchoOf:       parentCardID,
		}
		comments = append(comments, comment)
	}

	return comments, nil
}
