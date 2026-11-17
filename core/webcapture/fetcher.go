package webcapture

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"Finnegan/core/config"
)

var (
	// ErrPageProtected indicates access was denied due to auth/protection (e.g., HTTP 401/403).
	ErrPageProtected = errors.New("page is protected or requires authentication")
	// ErrEmptyContent indicates the fetched content was empty or too short (< 30 characters).
	ErrEmptyContent = errors.New("content too short or empty (< 30 characters)")
	// ErrBlocked indicates the request was blocked by anti-bot, captcha, or rate limiting (HTTP 429/5xx).
	ErrBlocked = errors.New("request was blocked or rate limited")
	// ErrFetchFailed indicates network or unexpected HTTP failure.
	ErrFetchFailed = errors.New("failed to fetch content")
)

// FetchError represents a structured error returned by the web capture fetcher.
type FetchError struct {
	Err        error
	StatusCode int
	URL        string
	Reason     string
}

func (e *FetchError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%v: %s (url: %s, status: %d)", e.Err, e.Reason, e.URL, e.StatusCode)
	}
	return fmt.Sprintf("%v (url: %s, status: %d)", e.Err, e.URL, e.StatusCode)
}

func (e *FetchError) Unwrap() error {
	return e.Err
}

// Fetcher fetches web page markdown using Jina Reader.
type Fetcher struct {
	JinaConfig config.JinaConfig
	Client     *http.Client
}

// NewFetcher creates a new Fetcher instance.
func NewFetcher(cfg ...config.JinaConfig) *Fetcher {
	var jCfg config.JinaConfig
	if len(cfg) > 0 {
		jCfg = cfg[0]
	} else {
		loaded, _, err := config.LoadConfig()
		if err == nil && loaded != nil {
			jCfg = loaded.Jina
		}
	}
	if jCfg.BaseURL == "" {
		jCfg.BaseURL = "https://r.jina.ai/"
	}
	return &Fetcher{
		JinaConfig: jCfg,
		Client: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

// WithClient overrides the HTTP client (useful for unit testing with mock servers).
func (f *Fetcher) WithClient(client *http.Client) *Fetcher {
	f.Client = client
	return f
}

// containsAntiBotNotice checks if content contains common anti-bot or captcha indicators.
// Only flags short responses (< 800 runes) so long valid articles or dialogues that mention
// security terms like "recaptcha", "turnstile", or "just a moment..." are never falsely blocked.
func containsAntiBotNotice(body string) bool {
	trimmed := strings.TrimSpace(body)
	if len([]rune(trimmed)) >= 800 {
		return false
	}
	lower := strings.ToLower(trimmed)
	patterns := []string{
		"verify you are human",
		"checking your browser",
		"just a moment...",
		"attention required! | cloudflare",
		"cf-turnstile",
		"turnstile",
		"please complete the security check",
		"sorry, you have been blocked",
		"access denied: security check",
		"bot detection",
		"robot or human",
		"recaptcha",
		"hcaptcha",
		"solve the captcha",
		"security check to continue",
		"enable cookies and reload the page",
		"captcha",
		"access denied",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// Fetch fetches the target URL via Jina Reader, enforcing fast-fail checks.
func (f *Fetcher) Fetch(ctx context.Context, targetURL string) (string, error) {
	cleanURL := strings.TrimSpace(targetURL)
	if cleanURL == "" {
		return "", &FetchError{
			Err:    ErrEmptyContent,
			URL:    targetURL,
			Reason: "target URL is empty",
		}
	}

	baseURL := f.JinaConfig.BaseURL
	if baseURL == "" {
		baseURL = "https://r.jina.ai/"
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	reqURL := baseURL + cleanURL

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", &FetchError{
			Err:    ErrFetchFailed,
			URL:    cleanURL,
			Reason: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", "Finnegan/1.0")
	req.Header.Set("X-Timeout", "30")
	if f.JinaConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+f.JinaConfig.APIKey)
	}

	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", &FetchError{
			Err:    ErrFetchFailed,
			URL:    cleanURL,
			Reason: fmt.Sprintf("network error: %v", err),
		}
	}
	defer resp.Body.Close()

	// Fast-fail check: HTTP status code
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", &FetchError{
			Err:        ErrPageProtected,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     fmt.Sprintf("access denied (status %d)", resp.StatusCode),
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &FetchError{
			Err:        ErrBlocked,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     "rate limit exceeded (status 429)",
		}
	}

	if resp.StatusCode >= 500 {
		return "", &FetchError{
			Err:        ErrBlocked,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     fmt.Sprintf("remote server error (status %d)", resp.StatusCode),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", &FetchError{
			Err:        ErrFetchFailed,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode),
		}
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", &FetchError{
			Err:        ErrFetchFailed,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     fmt.Sprintf("error reading body: %v", err),
		}
	}

	bodyStr := string(bodyBytes)

	// Fast-fail check: Anti-bot / Captcha notice
	if containsAntiBotNotice(bodyStr) {
		return "", &FetchError{
			Err:        ErrBlocked,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     "anti-bot or captcha notice detected in content",
		}
	}

	// Fast-fail check: Trimmed content length < 30
	trimmed := strings.TrimSpace(bodyStr)
	if len([]rune(trimmed)) < 30 {
		return "", &FetchError{
			Err:        ErrEmptyContent,
			StatusCode: resp.StatusCode,
			URL:        cleanURL,
			Reason:     fmt.Sprintf("trimmed content length too short (%d runes)", len([]rune(trimmed))),
		}
	}

	return trimmed, nil
}
