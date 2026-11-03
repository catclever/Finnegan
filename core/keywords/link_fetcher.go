package keywords

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	urlRegex  = regexp.MustCompile(`https?://[^\s)\]]+`)
	htmlRegex = regexp.MustCompile(`<[^>]*>`)
)

// SniffURLs extracts unique HTTP/HTTPS URLs from the provided text, up to a maximum of 2 URLs.
// Trailing punctuation and adjacent Chinese characters/punctuation are trimmed.
func SniffURLs(text string) []string {
	matches := urlRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var urls []string
	seen := make(map[string]bool)

	for _, m := range matches {
		// Stop at the first Chinese character or Chinese punctuation mark
		// e.g. "https://github.com/dalembert/super-agent，很有启发" -> "https://github.com/dalembert/super-agent"
		idx := strings.IndexFunc(m, func(r rune) bool {
			return unicode.Is(unicode.Han, r) ||
				r == '，' || r == '。' || r == '！' || r == '？' ||
				r == '；' || r == '：' || r == '“' || r == '”' ||
				r == '‘' || r == '’' || r == '（' || r == '）' ||
				r == '【' || r == '】' || r == '《' || r == '》' || r == '、'
		})
		cleaned := m
		if idx != -1 {
			cleaned = m[:idx]
		}
		cleaned = strings.TrimRight(cleaned, ".,;:!?，。！？\"'<>")
		if cleaned == "" || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		urls = append(urls, cleaned)
		if len(urls) >= 2 {
			break
		}
	}

	return urls
}

// LinkFetcher defines the contract for fetching external web content for in-card URLs.
type LinkFetcher interface {
	Fetch(ctx context.Context, targetURL string) (string, error)
	FetchAll(ctx context.Context, urls []string) string
}

// DefaultLinkFetcher performs lightweight web fetches via Jina Reader with direct HTTP GET fallback.
type DefaultLinkFetcher struct {
	JinaAPIKey string
	BaseURL    string
	Client     *http.Client
}

// NewDefaultLinkFetcher creates a new DefaultLinkFetcher instance.
func NewDefaultLinkFetcher(apiKey string) *DefaultLinkFetcher {
	if apiKey == "" {
		apiKey = os.Getenv("JINA_API_KEY")
	}
	return &DefaultLinkFetcher{
		JinaAPIKey: apiKey,
		BaseURL:    "https://r.jina.ai/",
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

// Fetch fetches lightweight content for targetURL, capping timeout strictly at 3 seconds per attempt.
// Tries Jina Reader first, falling back to direct HTTP GET if Jina fails or times out.
func (f *DefaultLinkFetcher) Fetch(ctx context.Context, targetURL string) (string, error) {
	baseURL := f.BaseURL
	if baseURL == "" {
		baseURL = "https://r.jina.ai/"
	}

	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}

	// 1. Primary path: Jina Reader API
	jinaCtx, jinaCancel := context.WithTimeout(ctx, 3*time.Second)
	defer jinaCancel()

	jinaReqURL := baseURL + targetURL
	jinaReq, err := http.NewRequestWithContext(jinaCtx, "GET", jinaReqURL, nil)
	if err == nil {
		jinaReq.Header.Set("Accept", "text/plain")
		jinaReq.Header.Set("User-Agent", "Finnegan/1.0")
		jinaReq.Header.Set("X-Timeout", "3")
		if f.JinaAPIKey != "" {
			jinaReq.Header.Set("Authorization", "Bearer "+f.JinaAPIKey)
		}

		resp, doErr := client.Do(jinaReq)
		if doErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
				if readErr == nil {
					text := strings.TrimSpace(string(bodyBytes))
					if text != "" {
						return TruncateRunes(text, 1500), nil
					}
				}
			}
		}
	}

	// 2. Fallback path: Direct HTTP GET
	directCtx, directCancel := context.WithTimeout(ctx, 3*time.Second)
	defer directCancel()

	directReq, err := http.NewRequestWithContext(directCtx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create direct request: %w", err)
	}
	directReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Finnegan/1.0")
	directReq.Header.Set("Accept", "text/html,text/plain,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(directReq)
	if err != nil {
		return "", fmt.Errorf("direct fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("direct fetch returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	raw := string(bodyBytes)
	clean := cleanHTMLText(raw)
	return TruncateRunes(clean, 1500), nil
}

// FetchAll fetches content for the provided URLs (up to 2) and aggregates them into a single string.
// If any fetch fails or times out, it degrades gracefully without returning an error.
func (f *DefaultLinkFetcher) FetchAll(ctx context.Context, urls []string) string {
	if len(urls) == 0 {
		return ""
	}

	var sb strings.Builder
	limit := len(urls)
	if limit > 2 {
		limit = 2
	}

	for i := 0; i < limit; i++ {
		u := urls[i]
		content, err := f.Fetch(ctx, u)
		if err != nil {
			log.Printf("[LinkFetcher] Notice: failed to fetch web context for %s (graceful degrade): %v", u, err)
			continue
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("[URL: %s]\n%s", u, content))
	}

	return TruncateRunes(sb.String(), 1500)
}

// TruncateRunes truncates string to maxRunes, adding an ellipsis if truncated.
func TruncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// cleanHTMLText strips HTML tags and normalizes consecutive whitespace.
func cleanHTMLText(html string) string {
	stripped := htmlRegex.ReplaceAllString(html, " ")
	fields := strings.Fields(stripped)
	return strings.Join(fields, " ")
}
