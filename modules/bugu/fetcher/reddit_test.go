package fetcher

import (
	"strings"
	"testing"
)

func TestIsRedditURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.reddit.com/r/golang/comments/17xyz/title/", true},
		{"https://reddit.com/r/technology/", true},
		{"https://old.reddit.com/r/LocalLLaMA/comments/123/test", true},
		{"https://redd.it/18abc", true},
		{"https://github.com/reddit/reddit", false},
		{"https://medium.com/article", false},
	}

	for _, tc := range tests {
		actual := IsRedditURL(tc.url)
		if actual != tc.expected {
			t.Errorf("IsRedditURL(%s) = %v, expected %v", tc.url, actual, tc.expected)
		}
	}
}

func TestFormatRedditJSONURL(t *testing.T) {
	raw := "https://www.reddit.com/r/golang/comments/18xyz/sample_post/?utm_source=share"
	expectedPath := "https://www.reddit.com/r/golang/comments/18xyz/sample_post.json?limit=8"

	formatted, err := FormatRedditJSONURL(raw)
	if err != nil {
		t.Fatalf("FormatRedditJSONURL failed: %v", err)
	}

	if formatted != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, formatted)
	}
}

func TestParseRedditJSONToMarkdown(t *testing.T) {
	sampleJSON := `[
		{
			"kind": "Listing",
			"data": {
				"children": [
					{
						"kind": "t3",
						"data": {
							"title": "Why Claude 3.5 Sonnet is still superior",
							"selftext": "After extensive benchmarks, sonnet outperforms opus 5.5 in coding accuracy.",
							"author": "prompt_engineer",
							"score": 450,
							"subreddit": "LocalLLaMA",
							"num_comments": 88,
							"created_utc": 1715000000,
							"url": "https://www.reddit.com/r/LocalLLaMA/comments/123/sonnet/"
						}
					}
				]
			}
		},
		{
			"kind": "Listing",
			"data": {
				"children": [
					{
						"kind": "t1",
						"data": {
							"author": "gopher_master",
							"body": "Completely agree, tool use is much better.",
							"score": 120,
							"created_utc": 1715001000
						}
					},
					{
						"kind": "t1",
						"data": {
							"author": "[deleted]",
							"body": "[deleted]",
							"score": 0
						}
					}
				]
			}
		}
	]`

	md, err := ParseRedditJSONToMarkdown([]byte(sampleJSON), "https://www.reddit.com/r/LocalLLaMA/comments/123/sonnet/")
	if err != nil {
		t.Fatalf("ParseRedditJSONToMarkdown returned error: %v", err)
	}

	if !strings.Contains(md, "# Why Claude 3.5 Sonnet is still superior") {
		t.Errorf("expected markdown to contain post title, got: %s", md)
	}
	if !strings.Contains(md, "u/prompt_engineer") {
		t.Errorf("expected markdown to contain author, got: %s", md)
	}
	if !strings.Contains(md, "u/gopher_master") {
		t.Errorf("expected markdown to contain top comment author, got: %s", md)
	}
	if strings.Contains(md, "[deleted]") {
		t.Errorf("expected deleted comments to be filtered, got: %s", md)
	}
}
