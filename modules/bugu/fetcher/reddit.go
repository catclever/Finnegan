package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RedditClient struct {
	Client *http.Client
}

func NewRedditClient() *RedditClient {
	return &RedditClient{
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// IsRedditURL checks if the URL belongs to Reddit.
func IsRedditURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "reddit.com" || strings.HasSuffix(host, ".reddit.com") || host == "redd.it"
}

// FormatRedditJSONURL formats a Reddit URL to point to its .json endpoint with a comment limit.
func FormatRedditJSONURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	p := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(p, ".json") {
		p = p + ".json"
	}
	u.Path = p
	// Request top comments with a reasonable limit
	u.RawQuery = "limit=8"
	return u.String(), nil
}

type redditPostData struct {
	Title       string  `json:"title"`
	Selftext    string  `json:"selftext"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	Subreddit   string  `json:"subreddit"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
}

type redditCommentData struct {
	Author     string  `json:"author"`
	Body       string  `json:"body"`
	Score      int     `json:"score"`
	CreatedUTC float64 `json:"created_utc"`
}

type redditChild struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type redditListingData struct {
	Children []redditChild `json:"children"`
}

type redditListing struct {
	Kind string            `json:"kind"`
	Data redditListingData `json:"data"`
}

// Extract fetches Reddit thread data via .json endpoint from residential IP and converts to rich Markdown.
func (r *RedditClient) Extract(rawURL string) (string, error) {
	jsonURL, err := FormatRedditJSONURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to format Reddit JSON URL: %w", err)
	}

	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return "", err
	}

	// Use realistic desktop browser headers so residential IP is fully recognized
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := r.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit returned HTTP %d for %s", resp.StatusCode, jsonURL)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return ParseRedditJSONToMarkdown(bodyBytes, rawURL)
}

// ParseRedditJSONToMarkdown parses Reddit API JSON bytes into structured Markdown including post and top comments.
func ParseRedditJSONToMarkdown(data []byte, originalURL string) (string, error) {
	var rawArray []json.RawMessage
	if err := json.Unmarshal(data, &rawArray); err != nil {
		// Try parsing as single listing
		var singleListing redditListing
		if sErr := json.Unmarshal(data, &singleListing); sErr == nil && len(singleListing.Data.Children) > 0 {
			return parseSingleListingMarkdown(singleListing, originalURL), nil
		}
		return "", fmt.Errorf("invalid reddit JSON structure: %w", err)
	}

	if len(rawArray) == 0 {
		return "", fmt.Errorf("empty reddit JSON response")
	}

	// 1. Parse Post (Listing 0)
	var postListing redditListing
	if err := json.Unmarshal(rawArray[0], &postListing); err != nil {
		return "", fmt.Errorf("failed to parse reddit post listing: %w", err)
	}

	if len(postListing.Data.Children) == 0 {
		return "", fmt.Errorf("no post found in reddit listing")
	}

	var post redditPostData
	if err := json.Unmarshal(postListing.Data.Children[0].Data, &post); err != nil {
		return "", fmt.Errorf("failed to parse post data: %w", err)
	}

	// 2. Parse Top Comments (Listing 1, if available)
	var comments []redditCommentData
	if len(rawArray) > 1 {
		var commentListing redditListing
		if err := json.Unmarshal(rawArray[1], &commentListing); err == nil {
			for _, child := range commentListing.Data.Children {
				if child.Kind == "t1" {
					var c redditCommentData
					if cErr := json.Unmarshal(child.Data, &c); cErr == nil {
						trimmed := strings.TrimSpace(c.Body)
						if trimmed != "" && trimmed != "[deleted]" && trimmed != "[removed]" {
							comments = append(comments, c)
							if len(comments) >= 5 {
								break
							}
						}
					}
				}
			}
		}
	}

	return formatMarkdownFromPostAndComments(post, comments, originalURL), nil
}

func parseSingleListingMarkdown(listing redditListing, originalURL string) string {
	var sb strings.Builder
	sb.WriteString("# Reddit Community Overview\n\n")
	for i, child := range listing.Data.Children {
		if child.Kind == "t3" {
			var p redditPostData
			if err := json.Unmarshal(child.Data, &p); err == nil {
				sb.WriteString(fmt.Sprintf("## %d. %s\n", i+1, p.Title))
				if p.Author != "" {
					sb.WriteString(fmt.Sprintf("- **Author**: u/%s | **Score**: %d\n", p.Author, p.Score))
				}
				if strings.TrimSpace(p.Selftext) != "" {
					sb.WriteString(p.Selftext + "\n\n")
				}
			}
		}
		if i >= 4 {
			break
		}
	}
	return sb.String()
}

func formatMarkdownFromPostAndComments(post redditPostData, comments []redditCommentData, originalURL string) string {
	var sb strings.Builder

	// Post Title
	sb.WriteString(fmt.Sprintf("# %s\n\n", post.Title))

	// Metadata headers that LLM distiller can easily extract
	if post.CreatedUTC > 0 {
		pubTime := time.Unix(int64(post.CreatedUTC), 0).UTC().Format("2006-01-02 15:04:05")
		sb.WriteString(fmt.Sprintf("- **Published**: %s\n", pubTime))
	}
	if post.Author != "" {
		sb.WriteString(fmt.Sprintf("- **Author**: u/%s\n", post.Author))
	}
	if post.Subreddit != "" {
		sb.WriteString(fmt.Sprintf("- **Subreddit**: r/%s (Upvotes: %d, Comments: %d)\n", post.Subreddit, post.Score, post.NumComments))
	}
	sb.WriteString(fmt.Sprintf("- **Source**: %s\n\n", originalURL))

	// Main Post Body
	if strings.TrimSpace(post.Selftext) != "" {
		sb.WriteString("## Post Content\n\n")
		sb.WriteString(post.Selftext)
		sb.WriteString("\n\n")
	} else if post.URL != "" && post.URL != originalURL {
		sb.WriteString(fmt.Sprintf("## Link Shared\n\n%s\n\n", post.URL))
	}

	// Community Comments
	if len(comments) > 0 {
		sb.WriteString("## Top Community Discussions\n\n")
		for i, c := range comments {
			sb.WriteString(fmt.Sprintf("### %d. Comment by u/%s (⬆ %d upvotes)\n\n%s\n\n", i+1, c.Author, c.Score, c.Body))
		}
	}

	return sb.String()
}
