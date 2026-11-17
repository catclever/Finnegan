package webcapture

import (
	"net/url"
	"strings"

	"Finnegan/core/chatlog"
)

type CardType string

const (
	CardTypeChatLog CardType = "chat_log"
	CardTypeNote    CardType = "note"
)

// IsChatLogDomain tests whether a URL matches one of the known shared chat domains:
// - share.gemini.google/*
// - gemini.google.com/share/*
// - chatgpt.com/share/*
// - claude.ai/share/*
// - perplexity.ai/page/*
func IsChatLogDomain(rawURL string) bool {
	cleanURL := strings.TrimSpace(rawURL)
	if cleanURL == "" {
		return false
	}
	if !strings.Contains(cleanURL, "://") {
		cleanURL = "https://" + cleanURL
	}
	u, err := url.Parse(cleanURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())
	path := u.Path

	// 1. share.gemini.google/*
	if host == "share.gemini.google" {
		return true
	}
	// 2. gemini.google.com/share/*
	if (host == "gemini.google.com" || host == "www.gemini.google.com") &&
		(path == "/share" || strings.HasPrefix(path, "/share/")) {
		return true
	}
	// 3. chatgpt.com/share/*
	if (host == "chatgpt.com" || host == "www.chatgpt.com") &&
		(path == "/share" || strings.HasPrefix(path, "/share/")) {
		return true
	}
	// 4. claude.ai/share/*
	if (host == "claude.ai" || host == "www.claude.ai") &&
		(path == "/share" || strings.HasPrefix(path, "/share/")) {
		return true
	}
	// 5. perplexity.ai/page/*
	if (host == "perplexity.ai" || host == "www.perplexity.ai") &&
		(path == "/page" || strings.HasPrefix(path, "/page/")) {
		return true
	}

	return false
}

// IsGeminiShareDomain checks whether a URL is a Google Gemini shared conversation URL.
func IsGeminiShareDomain(rawURL string) bool {
	cleanURL := strings.TrimSpace(rawURL)
	if cleanURL == "" {
		return false
	}
	if !strings.Contains(cleanURL, "://") {
		cleanURL = "https://" + cleanURL
	}
	u, err := url.Parse(cleanURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	path := u.Path
	if host == "share.gemini.google" {
		return true
	}
	if (host == "gemini.google.com" || host == "www.gemini.google.com") &&
		(path == "/share" || strings.HasPrefix(path, "/share/")) {
		return true
	}
	return false
}

// IsGeminiShare returns true if the URL is a Gemini share domain or the markdown contains "##### You said".
func IsGeminiShare(rawURL string, markdown string) bool {
	if IsGeminiShareDomain(rawURL) {
		return true
	}
	return reGeminiYouSaidLine.MatchString(markdown)
}

// Route determines the appropriate card type for the given URL and markdown content.
// Rules:
// 1. Domain match -> CardTypeChatLog
// 2. Content heuristic (Gemini share or chatlog.IsChatLog) -> CardTypeChatLog
// 3. Fallback -> CardTypeNote
func Route(urlStr string, markdown string) CardType {
	if IsChatLogDomain(urlStr) || IsGeminiShare(urlStr, markdown) {
		return CardTypeChatLog
	}
	if chatlog.IsChatLog(markdown) {
		return CardTypeChatLog
	}
	return CardTypeNote
}

// DetermineCardType is an alias for Route.
func DetermineCardType(urlStr string, markdown string) CardType {
	return Route(urlStr, markdown)
}
