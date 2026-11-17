package webcapture

import (
	"testing"
)

func TestRouter_DomainRules(t *testing.T) {
	cases := []struct {
		url      string
		expected CardType
	}{
		// 1. share.gemini.google/*
		{"https://share.gemini.google/12345abc", CardTypeChatLog},
		{"http://share.gemini.google/xyz", CardTypeChatLog},
		{"share.gemini.google/test-convo", CardTypeChatLog},

		// 2. gemini.google.com/share/*
		{"https://gemini.google.com/share/987654", CardTypeChatLog},
		{"https://www.gemini.google.com/share/abc", CardTypeChatLog},
		{"gemini.google.com/share/def?hl=en", CardTypeChatLog},

		// 3. chatgpt.com/share/*
		{"https://chatgpt.com/share/67290123-abcd", CardTypeChatLog},
		{"https://www.chatgpt.com/share/xyz", CardTypeChatLog},
		{"chatgpt.com/share/some-id", CardTypeChatLog},

		// 4. claude.ai/share/*
		{"https://claude.ai/share/a1b2c3d4-e5f6", CardTypeChatLog},
		{"https://www.claude.ai/share/session-789", CardTypeChatLog},
		{"claude.ai/share/my-chat", CardTypeChatLog},

		// 5. perplexity.ai/page/*
		{"https://perplexity.ai/page/research-article-slug", CardTypeChatLog},
		{"https://www.perplexity.ai/page/query-summary-123", CardTypeChatLog},
		{"perplexity.ai/page/ai-agents", CardTypeChatLog},

		// Negative / regular domains
		{"https://gemini.google.com/app", CardTypeNote},
		{"https://chatgpt.com/g/g-some-custom-gpt", CardTypeNote},
		{"https://claude.ai/chat/123", CardTypeNote},
		{"https://perplexity.ai/search/some-query", CardTypeNote},
		{"https://github.com/dalembert/super-agent", CardTypeNote},
		{"https://news.ycombinator.com/item?id=123", CardTypeNote},
		{"https://blog.google/technology/ai", CardTypeNote},
	}

	nonChatMarkdown := "# Generic Article\n\nThis is a standard text article without dialogue headers."

	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			got := Route(tc.url, nonChatMarkdown)
			if got != tc.expected {
				t.Errorf("Route(%q) = %q, expected %q", tc.url, got, tc.expected)
			}
		})
	}
}

func TestRouter_ContentHeuristics(t *testing.T) {
	// Regular URL that would normally route to note, but content is a chat log dialogue
	regularURL := "https://example.com/transcript"

	dialogueMarkdown := `# Discussion on Architecture

Alice 2026-10-14 08:15
We should decouple the router from the transport layer.

Bob 2026-10-14 08:16
Agreed, that makes fast-fail testing much simpler.
`

	got := Route(regularURL, dialogueMarkdown)
	if got != CardTypeChatLog {
		t.Errorf("expected CardTypeChatLog via dialogue heuristic, got %q", got)
	}

	colonDialogue := `# Dialogue
User:
How does WebCapture handle rate limiting?

Assistant:
It triggers an immediate fast-fail returning ErrBlocked.
`
	gotColon := Route(regularURL, colonDialogue)
	if gotColon != CardTypeChatLog {
		t.Errorf("expected CardTypeChatLog via colon dialogue heuristic, got %q", gotColon)
	}

	// Regular article without dialogue
	regularMarkdown := `# Architecture Overview
In this document, we explore micro-kernel design in Go and Wails.
We show how event buses keep layers decoupled.
`
	gotNote := Route(regularURL, regularMarkdown)
	if gotNote != CardTypeNote {
		t.Errorf("expected CardTypeNote fallback, got %q", gotNote)
	}
}
