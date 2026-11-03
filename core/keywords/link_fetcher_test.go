package keywords

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSniffURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "No URLs",
			input:    "今天天气真不错，没有链接。",
			expected: nil,
		},
		{
			name:     "Single clean URL",
			input:    "查看项目详情: https://github.com/dalembert/super-agent 完成了存算分离",
			expected: []string{"https://github.com/dalembert/super-agent"},
		},
		{
			name:     "Markdown link with trailing punctuation",
			input:    "请参考 [Dalembert](https://github.com/dalembert/super-agent)，还有镜像 https://mirror.example.com/repo!",
			expected: []string{"https://github.com/dalembert/super-agent", "https://mirror.example.com/repo"},
		},
		{
			name:     "More than 2 URLs capped at 2",
			input:    "https://url1.com https://url2.com https://url3.com",
			expected: []string{"https://url1.com", "https://url2.com"},
		},
		{
			name:     "Duplicate URLs deduplicated",
			input:    "访问 https://example.com/test 和 https://example.com/test 了解更多",
			expected: []string{"https://example.com/test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SniffURLs(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("SniffURLs() got %d urls, expected %d: %v", len(got), len(tt.expected), got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("SniffURLs()[%d] = %q, expected %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	str := "这是一段包含中文字符的测试文本，用来验证截断逻辑是否正确。"
	truncated := TruncateRunes(str, 10)
	runes := []rune(truncated)
	// 10 runes + 3 for "..."
	if len(runes) != 13 || !strings.HasSuffix(truncated, "...") {
		t.Errorf("Expected truncated string to have 13 runes and end with '...', got %s (len %d)", truncated, len(runes))
	}

	short := "短文本"
	if TruncateRunes(short, 10) != short {
		t.Errorf("Short text should not be truncated")
	}
}

func TestDefaultLinkFetcher_JinaSuccess(t *testing.T) {
	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			auth := req.Header.Get("Authorization")
			if auth != "Bearer test-jina-key" {
				t.Errorf("Expected Bearer test-jina-key, got %s", auth)
			}
			respBody := "# Project Dalembert\n\nSuper-agent decoupling compute and storage."
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	fetcher := &DefaultLinkFetcher{
		JinaAPIKey: "test-jina-key",
		BaseURL:    "https://r.jina.ai/",
		Client:     inMemClient,
	}

	content, err := fetcher.Fetch(context.Background(), "https://github.com/dalembert/super-agent")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !strings.Contains(content, "Super-agent decoupling compute and storage") {
		t.Errorf("Expected fetched markdown content, got: %s", content)
	}
}

func TestDefaultLinkFetcher_JinaFailFallbackToDirect(t *testing.T) {
	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.HasPrefix(req.URL.String(), "https://r.jina.ai/") {
				// Jina fails with 500
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Jina error")),
					Header:     make(http.Header),
				}, nil
			}
			// Direct request succeeds
			htmlBody := "<html><body><h1>Super Agent Repo</h1><p>Direct fetched content.</p></body></html>"
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(htmlBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	fetcher := &DefaultLinkFetcher{
		JinaAPIKey: "test-key",
		BaseURL:    "https://r.jina.ai/",
		Client:     inMemClient,
	}

	content, err := fetcher.Fetch(context.Background(), "https://github.com/dalembert/super-agent")
	if err != nil {
		t.Fatalf("Fetch fallback failed: %v", err)
	}

	if !strings.Contains(content, "Super Agent Repo") || !strings.Contains(content, "Direct fetched content.") {
		t.Errorf("Expected direct fetched HTML stripped text, got: %s", content)
	}
}

func TestDefaultLinkFetcher_TimeoutGracefulDegrade(t *testing.T) {
	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Simulate network timeout or error
			return nil, errors.New("connection timed out")
		}),
	}

	fetcher := &DefaultLinkFetcher{
		BaseURL: "https://r.jina.ai/",
		Client:  inMemClient,
	}

	// FetchAll should gracefully return empty string without failing
	res := fetcher.FetchAll(context.Background(), []string{"https://example.com/slow"})
	if res != "" {
		t.Errorf("Expected empty string on error, got %q", res)
	}
}
