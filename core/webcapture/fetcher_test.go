package webcapture

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"Finnegan/core/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetcher_Success(t *testing.T) {
	mockBody := "# Sample Article Title\n\nThis is a long article content exceeding thirty characters easily."
	var capturedAccept, capturedUA, capturedTimeout, capturedAuth string

	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedAccept = req.Header.Get("Accept")
			capturedUA = req.Header.Get("User-Agent")
			capturedTimeout = req.Header.Get("X-Timeout")
			capturedAuth = req.Header.Get("Authorization")

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := config.JinaConfig{
		BaseURL: "https://r.jina.ai/",
		APIKey:  "test-secret-token",
	}

	fetcher := NewFetcher(cfg).WithClient(inMemClient)
	res, err := fetcher.Fetch(context.Background(), "https://example.com/sample")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if res != mockBody {
		t.Errorf("expected body %q, got %q", mockBody, res)
	}
	if capturedAccept != "text/plain" {
		t.Errorf("expected Accept text/plain, got %q", capturedAccept)
	}
	if capturedUA != "Finnegan/1.0" {
		t.Errorf("expected User-Agent Finnegan/1.0, got %q", capturedUA)
	}
	if capturedTimeout != "30" {
		t.Errorf("expected X-Timeout 30, got %q", capturedTimeout)
	}
	if capturedAuth != "Bearer test-secret-token" {
		t.Errorf("expected Authorization Bearer test-secret-token, got %q", capturedAuth)
	}

	defaultFetcher := NewFetcher(cfg)
	if defaultFetcher.Client.Timeout != 45*time.Second {
		t.Errorf("expected default client Timeout 45s, got %v", defaultFetcher.Client.Timeout)
	}
}

func TestFetcher_FailFast_Statuses(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{
			name:    "401 Unauthorized",
			status:  http.StatusUnauthorized,
			body:    "Unauthorized access",
			wantErr: ErrPageProtected,
		},
		{
			name:    "403 Forbidden",
			status:  http.StatusForbidden,
			body:    "Forbidden page content",
			wantErr: ErrPageProtected,
		},
		{
			name:    "429 Too Many Requests",
			status:  http.StatusTooManyRequests,
			body:    "Rate limited",
			wantErr: ErrBlocked,
		},
		{
			name:    "500 Internal Server Error",
			status:  http.StatusInternalServerError,
			body:    "Server crashed",
			wantErr: ErrBlocked,
		},
		{
			name:    "502 Bad Gateway",
			status:  http.StatusBadGateway,
			body:    "Bad gateway",
			wantErr: ErrBlocked,
		},
		{
			name:    "503 Service Unavailable",
			status:  http.StatusServiceUnavailable,
			body:    "Service unavailable",
			wantErr: ErrBlocked,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inMemClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.status,
						Body:       io.NopCloser(bytes.NewBufferString(tc.body)),
						Header:     make(http.Header),
					}, nil
				}),
			}

			fetcher := NewFetcher(config.JinaConfig{BaseURL: "https://r.jina.ai/"}).WithClient(inMemClient)
			_, err := fetcher.Fetch(context.Background(), "https://example.com")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestFetcher_FailFast_AntiBotAndCaptcha(t *testing.T) {
	notices := []string{
		"Attention Required! | Cloudflare - Please verify you are human",
		"Just a moment... Enable JavaScript and cookies to continue",
		"Checking your browser before accessing the website",
		"Please complete the security check to continue (cf-turnstile)",
		"Access denied: security check required",
		"Solve the captcha below to proceed to the page",
		"Bot detection has triggered. Robot or human?",
	}

	for _, notice := range notices {
		t.Run(notice[:20], func(t *testing.T) {
			inMemClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(notice + " and extra padding to exceed thirty characters.")),
						Header:     make(http.Header),
					}, nil
				}),
			}

			fetcher := NewFetcher(config.JinaConfig{BaseURL: "https://r.jina.ai/"}).WithClient(inMemClient)
			_, err := fetcher.Fetch(context.Background(), "https://example.com/protected")
			if err == nil {
				t.Fatalf("expected error for anti-bot body, got nil")
			}

			if !errors.Is(err, ErrBlocked) {
				t.Errorf("expected ErrBlocked, got %v", err)
			}
		})
	}

	t.Run("LongValidArticleWithSecurityKeywordsNotBlocked", func(t *testing.T) {
		longBody := "# Security Architecture Review\n\n" +
			"In this article we discuss how modern web applications integrate recaptcha, hcaptcha, and Cloudflare turnstile " +
			"without degrading user experience with 'just a moment...' screens. " +
			strings.Repeat("This is a detailed engineering paragraph discussing distributed systems and agent architectures. ", 12)

		inMemClient := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(longBody)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		fetcher := NewFetcher(config.JinaConfig{BaseURL: "https://r.jina.ai/"}).WithClient(inMemClient)
		got, err := fetcher.Fetch(context.Background(), "https://example.com/security-article")
		if err != nil {
			t.Fatalf("expected long valid article not to be blocked, got error: %v", err)
		}
		if got == "" {
			t.Errorf("expected non-empty body")
		}
	})
}

func TestFetcher_FailFast_EmptyOrTooShort(t *testing.T) {
	cases := []string{
		"",
		"   \n\t  ",
		"Short text",
		"12345678901234567890123456789", // 29 chars
	}

	for _, c := range cases {
		t.Run("Length_"+strings.TrimSpace(c), func(t *testing.T) {
			inMemClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(c)),
						Header:     make(http.Header),
					}, nil
				}),
			}

			fetcher := NewFetcher(config.JinaConfig{BaseURL: "https://r.jina.ai/"}).WithClient(inMemClient)
			_, err := fetcher.Fetch(context.Background(), "https://example.com")
			if err == nil {
				t.Fatalf("expected error for short/empty content, got nil")
			}

			if !errors.Is(err, ErrEmptyContent) {
				t.Errorf("expected ErrEmptyContent, got %v", err)
			}
		})
	}
}

func TestFetcher_Timeout(t *testing.T) {
	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			select {
			case <-time.After(50 * time.Millisecond):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("Some valid response content here over thirty characters long.")),
					Header:     make(http.Header),
				}, nil
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}),
	}

	fetcher := NewFetcher(config.JinaConfig{BaseURL: "https://r.jina.ai/"}).WithClient(inMemClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := fetcher.Fetch(ctx, "https://example.com/slow")
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	if !errors.Is(err, ErrFetchFailed) {
		t.Errorf("expected ErrFetchFailed on timeout, got %v", err)
	}
}
