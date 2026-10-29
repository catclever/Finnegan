package babel

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	urlRegex      = regexp.MustCompile(`https?://[^\s]+`)
	linkSpamWords = []string{
		"airdrop", "giveaway", "free mint", "claim free", "presale",
		"whitelist", "dm for promo", "100x gem", "casino", "poker",
		"slot machine", "telegram:", "t.me/", "whatsapp:", "join my channel",
		"follow & rt", "retweet & like", "drop your address", "buy now",
		"limited discount", "click here to read",
	}
)

// UniversalSpamFilter provides foundational hygiene checks for any incoming text.
// Returns (isSpam bool, reason string).
func UniversalSpamFilter(text string) (bool, string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true, "empty_content"
	}

	// Substantive character check (ignore whitespaces, markdown formatting characters)
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '#' || r == '*' || r == '`' || r == '>' || r == '-' || r == '_' || r == '[' || r == ']' || r == '(' || r == ')' {
			return -1
		}
		return r
	}, trimmed)

	if len([]rune(clean)) < 4 {
		return true, "too_short"
	}

	// UTF-8 Validity & Mojibake Check
	if !utf8.ValidString(trimmed) {
		return true, "invalid_utf8"
	}

	// Replacement character check (broken encoding)
	if strings.Count(trimmed, "\uFFFD") >= 2 {
		return true, "mojibake_replacement_chars"
	}

	// Excessive control characters check
	controlCount := 0
	totalRunes := 0
	for _, r := range trimmed {
		totalRunes++
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
	}
	if totalRunes > 0 && float64(controlCount)/float64(totalRunes) > 0.05 {
		return true, "excessive_control_chars"
	}

	// Pure Link Drainage check: If text without URLs is almost empty
	textWithoutURLs := strings.TrimSpace(urlRegex.ReplaceAllString(trimmed, ""))
	if len([]rune(textWithoutURLs)) < 6 && urlRegex.MatchString(trimmed) {
		return true, "pure_link_drainage"
	}

	// SEO/Scam link spam check
	lower := strings.ToLower(trimmed)
	for _, badWord := range linkSpamWords {
		if strings.Contains(lower, badWord) {
			return true, "spam_promotional_keyword:" + badWord
		}
	}

	return false, ""
}
