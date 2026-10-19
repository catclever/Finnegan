package fetcher

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"Finnegan/modules/bugu/models"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSearxNGClient_Search_TimeRange(t *testing.T) {
	tests := []struct {
		name              string
		timeRange         string
		expectedParam     string
		shouldContainTime bool
	}{
		{
			name:              "With month time_range",
			timeRange:         "month",
			expectedParam:     "time_range=month",
			shouldContainTime: true,
		},
		{
			name:              "With year time_range",
			timeRange:         "year",
			expectedParam:     "time_range=year",
			shouldContainTime: true,
		},
		{
			name:              "With week time_range",
			timeRange:         "week",
			expectedParam:     "time_range=week",
			shouldContainTime: true,
		},
		{
			name:              "With empty time_range (no time restriction)",
			timeRange:         "",
			expectedParam:     "time_range=",
			shouldContainTime: false,
		},
		{
			name:              "With whitespace time_range",
			timeRange:         "   ",
			expectedParam:     "time_range=",
			shouldContainTime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recordedURL string
			client := NewSearxNGClient("http://mock-searxng.local")
			client.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				recordedURL = req.URL.String()
				body := `{"results":[{"url":"https://example.com/article1","title":"Article 1","content":"Snippet 1"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
				}, nil
			})

			target := models.Target{
				ID:    "hn",
				Type:  "community",
				Value: "site:news.ycombinator.com",
			}

			results, err := client.Search("golang concurrency", target, tt.timeRange)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Title != "Article 1" {
				t.Errorf("expected Title 'Article 1', got '%s'", results[0].Title)
			}

			if tt.shouldContainTime {
				if !strings.Contains(recordedURL, tt.expectedParam) {
					t.Errorf("expected URL to contain %q, but URL was: %s", tt.expectedParam, recordedURL)
				}
			} else {
				if strings.Contains(recordedURL, "time_range") {
					t.Errorf("expected URL to NOT contain time_range, but URL was: %s", recordedURL)
				}
			}

			// Verify query and format=json are present
			if !strings.Contains(recordedURL, "format=json") {
				t.Errorf("expected format=json in URL: %s", recordedURL)
			}
			if !strings.Contains(recordedURL, "site%3Anews.ycombinator.com") && !strings.Contains(recordedURL, "site:news.ycombinator.com") {
				t.Errorf("expected target value in URL: %s", recordedURL)
			}
		})
	}
}

func TestSearxNGClient_Search_ErrorStatus(t *testing.T) {
	client := NewSearxNGClient("http://mock-searxng.local")
	client.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
			Header:     make(http.Header),
		}, nil
	})

	target := models.Target{ID: "test", Value: "site:test.com"}

	_, err := client.Search("test query", target, "month")
	if err == nil {
		t.Fatal("expected error on 500 status code, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error message to mention 500, got: %v", err)
	}
}
