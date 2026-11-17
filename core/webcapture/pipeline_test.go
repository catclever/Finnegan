package webcapture

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"Finnegan/core/config"
)

func TestPipeline_CaptureNote(t *testing.T) {
	mockJinaResponse := `Title: Effective Go Architecture
URL Source: https://go.dev/doc/effective_go
Markdown Content:
# Effective Go Architecture

Go is a general-purpose language designed with systems programming in mind.
This note demonstrates clean code organization in Go backend projects.
`

	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockJinaResponse)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	pipeline := NewPipeline(config.JinaConfig{BaseURL: "https://r.jina.ai/"})
	pipeline.Fetcher.WithClient(inMemClient)

	card, err := pipeline.Capture(context.Background(), "https://go.dev/doc/effective_go")
	if err != nil {
		t.Fatalf("pipeline.Capture failed: %v", err)
	}

	if card.Type != CardTypeNote {
		t.Errorf("expected CardTypeNote, got %v", card.Type)
	}
	if card.Title != "Effective Go Architecture" {
		t.Errorf("expected Title 'Effective Go Architecture', got %q", card.Title)
	}
	if card.SourceURL != "https://go.dev/doc/effective_go" {
		t.Errorf("expected SourceURL 'https://go.dev/doc/effective_go', got %q", card.SourceURL)
	}

	// Verify Frontmatter
	if !strings.Contains(card.FullMarkdown, "type: note") {
		t.Errorf("FullMarkdown missing 'type: note'")
	}
	if !strings.Contains(card.FullMarkdown, "source: https://go.dev/doc/effective_go") {
		t.Errorf("FullMarkdown missing 'source: https://go.dev/doc/effective_go'")
	}
	if !strings.Contains(card.FullMarkdown, "title: Effective Go Architecture") {
		t.Errorf("FullMarkdown missing 'title: Effective Go Architecture'")
	}

	// Body should not contain Jina header artifacts
	if strings.Contains(card.Content, "URL Source:") {
		t.Errorf("Content should not contain Jina header 'URL Source:'")
	}
	if strings.Contains(card.Content, "Markdown Content:") {
		t.Errorf("Content should not contain Jina header 'Markdown Content:'")
	}
}

func TestPipeline_CaptureChatLog_Domain(t *testing.T) {
	mockSharedChat := `# LLM Prompt Engineering Session

User 2026-10-15 14:00
How do I structure a domain-driven pipeline in Go?

Assistant 2026-10-15 14:01
Separate your pure business logic into domain packages, keeping app wiring thin.
`

	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockSharedChat)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	pipeline := NewPipeline(config.JinaConfig{BaseURL: "https://r.jina.ai/"})
	pipeline.Fetcher.WithClient(inMemClient)

	card, err := pipeline.Capture(context.Background(), "https://chatgpt.com/share/672a1234")
	if err != nil {
		t.Fatalf("pipeline.Capture failed: %v", err)
	}

	if card.Type != CardTypeChatLog {
		t.Errorf("expected CardTypeChatLog, got %v", card.Type)
	}
	if card.Title != "LLM Prompt Engineering Session" {
		t.Errorf("expected Title 'LLM Prompt Engineering Session', got %q", card.Title)
	}
	if card.SourceURL != "https://chatgpt.com/share/672a1234" {
		t.Errorf("expected SourceURL 'https://chatgpt.com/share/672a1234', got %q", card.SourceURL)
	}

	// Frontmatter checks
	if !strings.Contains(card.FullMarkdown, "type: chat_log") {
		t.Errorf("FullMarkdown missing 'type: chat_log'")
	}
	if !strings.Contains(card.FullMarkdown, "source: https://chatgpt.com/share/672a1234") {
		t.Errorf("FullMarkdown missing source")
	}

	// Participants extracted
	if len(card.Participants) != 2 {
		t.Errorf("expected 2 participants, got %v", card.Participants)
	}
	if !strings.Contains(card.FullMarkdown, "participants: [User, Assistant]") {
		t.Errorf("FullMarkdown missing participants, got: %s", card.FullMarkdown)
	}
}

func TestPipeline_CaptureChatLog_FallbackBodyWhenParsingUnstructured(t *testing.T) {
	// A domain-matched shared chat URL where the markdown does not strictly have colon/timestamp headers
	mockText := `# Shared Claude Transcript

This is a transcript where messages are formatted as plain bullet points instead of colons:
- Speaker 1 said hello
- Speaker 2 replied with details
`

	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockText)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	pipeline := NewPipeline(config.JinaConfig{BaseURL: "https://r.jina.ai/"})
	pipeline.Fetcher.WithClient(inMemClient)

	card, err := pipeline.Capture(context.Background(), "https://claude.ai/share/uuid-unstructured")
	if err != nil {
		t.Fatalf("pipeline.Capture failed: %v", err)
	}

	if card.Type != CardTypeChatLog {
		t.Errorf("expected CardTypeChatLog due to domain rule, got %v", card.Type)
	}
	if card.Title != "Shared Claude Transcript" {
		t.Errorf("expected title 'Shared Claude Transcript', got %q", card.Title)
	}
	if !strings.Contains(card.Content, "Speaker 1 said hello") {
		t.Errorf("expected body to preserve raw content on fallback, got %q", card.Content)
	}
}

func TestProcessContent_TimestampsAndFormatting(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 30, 0, 0, time.UTC)
	docID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"

	t.Run("Note formatting", func(t *testing.T) {
		res, err := ProcessContent("https://example.com/note", "# My Title\n\nContent here with plenty of detail.", docID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Type != CardTypeNote {
			t.Errorf("expected CardTypeNote, got %v", res.Type)
		}
		if !strings.Contains(res.FullMarkdown, "id: 01ARZ3NDEKTSV4RRFFQ69G5FAV") {
			t.Errorf("missing docID in frontmatter")
		}
		if !strings.Contains(res.FullMarkdown, "created_at: 2026-11-06T10:30:00Z") {
			t.Errorf("missing created_at in frontmatter")
		}
	})
}

func TestPipeline_CaptureGeminiShare(t *testing.T) {
	mockGeminiShareResponse := "Title: \u200eGemini - direct access to Google AI\n\n" +
		"URL Source: https://share.gemini.google/pJUOrIJC5jS6\n\n" +
		"Final URL: https://gemini.google.com/share/66a78b9c1234\n\n" +
		"Markdown Content:\n" +
		"[Sign in](https://accounts.google.com/ServiceLogin?hl=en&passive=true&continue=https://gemini.google.com/share/66a78b9c1234)\n\n" +
		"![Image 1](https://www.gstatic.com/lamda/images/gemini_sparkle_v002_d4735304ff6292a690345.svg)\n\n" +
		"Created with **2.5 Pro**\n\n" +
		"May 16, 2025 at 11:01 PM\n\n" +
		"Published May 16, 2025 at 11:03 PM\n\n" +
		"# 软件工程Agent架构解析\n\n" +
		"##### You said 请帮我深入解析现代软件工程Agent的核心架构与模块划分\n\n" +
		"请帮我深入解析现代软件工程Agent的核心架构与模块划分\n\n" +
		"现代软件工程 Agent 的核心架构通常由感知层、规划与推理层、工具执行层以及持久记忆层组成。\n\n" +
		"### 1. 规划与推理层\n" +
		"- **任务分解**: 将复杂需求拆解为可验证的子目标。\n" +
		"- **反思机制**: 根据沙箱测试反馈动态修正执行路径。\n\n" +
		"##### You said 这种Agent架构如何保障代码修改的安全性与幂等性？\n\n" +
		"这种Agent架构如何保障代码修改的安全性与幂等性？\n\n" +
		"主要通过受限沙箱环境（Sandboxing）、最小权限控制以及基于 Git 快照的事务性回滚机制来确保每次操作安全可控。\n\n" +
		"Gemini may display inaccurate info, including about people, so double-check its responses. [Your privacy & Gemini Apps](https://support.google.com/gemini?p=privacy_notice)\n\n" +
		"Opens in a new window\n"

	inMemClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockGeminiShareResponse)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	pipeline := NewPipeline(config.JinaConfig{BaseURL: "https://r.jina.ai/"})
	pipeline.Fetcher.WithClient(inMemClient)

	targetURL := "https://share.gemini.google/pJUOrIJC5jS6"
	card, err := pipeline.Capture(context.Background(), targetURL)
	if err != nil {
		t.Fatalf("pipeline.Capture failed: %v", err)
	}

	if card.Type != CardTypeChatLog {
		t.Errorf("expected CardTypeChatLog, got %v", card.Type)
	}
	if card.Title != "软件工程Agent架构解析" {
		t.Errorf("expected Title '软件工程Agent架构解析', got %q", card.Title)
	}
	if len(card.Participants) != 2 || card.Participants[0] != "User" || card.Participants[1] != "Gemini" {
		t.Errorf("expected Participants [User, Gemini], got %v", card.Participants)
	}

	// Verify Jina headers and Google header/footer boilerplate are stripped
	forbiddenSubstrings := []string{
		"Final URL:",
		"URL Source:",
		"Markdown Content:",
		"Gemini - direct access to Google AI",
		"Sign in",
		"Created with",
		"Gemini may display inaccurate info",
		"Your privacy & Gemini Apps",
		"Opens in a new window",
		"##### You said",
	}
	for _, sub := range forbiddenSubstrings {
		if strings.Contains(card.Content, sub) {
			t.Errorf("expected card.Content not to contain boilerplate %q, got:\n%s", sub, card.Content)
		}
	}

	// Verify formatted turns and deduplicated prompt
	if count := strings.Count(card.Content, "请帮我深入解析现代软件工程Agent的核心架构与模块划分"); count != 1 {
		t.Errorf("expected first prompt to appear once after deduplication, appeared %d times in:\n%s", count, card.Content)
	}
	if !strings.Contains(card.Content, "User:\n请帮我深入解析现代软件工程Agent的核心架构与模块划分") {
		t.Errorf("expected formatted User turn in card.Content, got:\n%s", card.Content)
	}
	if !strings.Contains(card.Content, "Gemini:\n现代软件工程 Agent 的核心架构通常由感知层") {
		t.Errorf("expected formatted Gemini turn in card.Content, got:\n%s", card.Content)
	}
	if !strings.Contains(card.FullMarkdown, "participants: [User, Gemini]") {
		t.Errorf("expected frontmatter to contain 'participants: [User, Gemini]', got:\n%s", card.FullMarkdown)
	}
}

