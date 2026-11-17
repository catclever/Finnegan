package webcapture

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"Finnegan/core/chatlog"
)

var (
	reJinaTitle       = regexp.MustCompile(`(?i)^title:\s*(.+)$`)
	reJinaURLSource   = regexp.MustCompile(`(?i)^url source:\s*(.+)$`)
	reJinaFinalURL    = regexp.MustCompile(`(?i)^final url:\s*(.+)$`)
	reJinaPublished   = regexp.MustCompile(`(?i)^published time:\s*(.+)$`)
	reJinaContentMark = regexp.MustCompile(`(?i)^markdown content:\s*$`)

	reGeminiYouSaid     = regexp.MustCompile(`(?i)^#{1,6}\s+you said(?:\s+(.*))?$`)
	reGeminiYouSaidLine = regexp.MustCompile(`(?im)^#{1,6}\s+you said(?:\s+.*)?$`)
	reGeminiSaid        = regexp.MustCompile(`(?i)^(?:#{1,6}\s+)?gemini said:?\s*$`)
)

func isJinaHeaderLine(trimmed string) bool {
	return reJinaTitle.MatchString(trimmed) ||
		reJinaURLSource.MatchString(trimmed) ||
		reJinaFinalURL.MatchString(trimmed) ||
		reJinaPublished.MatchString(trimmed) ||
		reJinaContentMark.MatchString(trimmed)
}

func cleanInvisibleMarks(s string) string {
	s = strings.NewReplacer(
		"\u200e", "",
		"\u200f", "",
		"\ufeff", "",
		"\u200b", "",
	).Replace(s)
	return strings.TrimSpace(s)
}

func isGenericSiteTitle(title string) bool {
	cleaned := strings.ToLower(cleanInvisibleMarks(title))
	if cleaned == "" {
		return true
	}
	genericSubstrings := []string{
		"gemini - direct access to google ai",
		"google gemini",
		"just a moment",
	}
	for _, sub := range genericSubstrings {
		if strings.Contains(cleaned, sub) {
			return true
		}
	}
	switch cleaned {
	case "gemini", "chatgpt", "claude", "perplexity", "shared chat", "conversation":
		return true
	}
	return false
}

// CleanJinaHeaders strips Jina Reader metadata headers (Title:, URL Source:, Final URL:, Markdown Content:)
// from the beginning of text so they do not contaminate the captured note body.
func CleanJinaHeaders(text string) string {
	lines := strings.Split(text, "\n")
	startIdx := 0
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if isJinaHeaderLine(trimmed) {
			startIdx = i + 1
			continue
		}
		if trimmed == "" && startIdx == i {
			startIdx = i + 1
			continue
		}
		break
	}
	if startIdx > 0 && startIdx <= len(lines) {
		return strings.Join(lines[startIdx:], "\n")
	}
	return text
}

// PreprocessGeminiShare cleans Google Gemini share boilerplate (header, footer, and duplicated prompt lines)
// and transforms "##### You said <prompt>" sections into clean "User:\n<prompt>\n\nGemini:\n<response>" blocks.
func PreprocessGeminiShare(text string) string {
	cleaned := CleanJinaHeaders(text)
	lines := strings.Split(cleaned, "\n")

	// 1. Strip Google footer ("Gemini may display inaccurate info..." and everything after it)
	footerIdx := -1
	for i, l := range lines {
		lower := strings.ToLower(strings.TrimSpace(l))
		if strings.Contains(lower, "gemini may display inaccurate info") ||
			strings.Contains(lower, "your privacy & gemini apps") {
			footerIdx = i
			break
		}
	}
	if footerIdx >= 0 {
		lines = lines[:footerIdx]
	}
	for len(lines) > 0 {
		lastLower := strings.ToLower(strings.TrimSpace(lines[len(lines)-1]))
		if lastLower == "" || lastLower == "opens in a new window" {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}

	// 2. Locate first H1 title and first "##### You said" turn
	firstYouSaidIdx := -1
	for i, l := range lines {
		if reGeminiYouSaid.MatchString(strings.TrimSpace(l)) {
			firstYouSaidIdx = i
			break
		}
	}

	if firstYouSaidIdx == -1 {
		return strings.Join(lines, "\n")
	}

	var h1TitleLine string
	for i := 0; i < firstYouSaidIdx; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), "#") {
			h1TitleLine = trimmed
			break
		}
	}

	// 3. Collect all "##### You said" turn indices
	var turnIndices []int
	for i := firstYouSaidIdx; i < len(lines); i++ {
		if reGeminiYouSaid.MatchString(strings.TrimSpace(lines[i])) {
			turnIndices = append(turnIndices, i)
		}
	}

	var blocks []string
	for tIdx, startLine := range turnIndices {
		endLine := len(lines)
		if tIdx+1 < len(turnIndices) {
			endLine = turnIndices[tIdx+1]
		}

		m := reGeminiYouSaid.FindStringSubmatch(strings.TrimSpace(lines[startLine]))
		firstPromptLine := ""
		if len(m) > 1 {
			firstPromptLine = strings.TrimSpace(m[1])
		}

		cur := startLine + 1
		for cur < endLine && strings.TrimSpace(lines[cur]) == "" {
			cur++
		}

		var prompt string
		if firstPromptLine != "" {
			// Check for the duplicated prompt line(s) rendered by Gemini DOM
			dupIdx := -1
			searchLimit := cur + 20
			if searchLimit > endLine {
				searchLimit = endLine
			}
			for k := cur; k < searchLimit; k++ {
				if strings.TrimSpace(lines[k]) == firstPromptLine {
					dupIdx = k
					break
				}
			}

			if dupIdx >= cur {
				extraRaw := lines[cur:dupIdx]
				extraPromptLines := trimBlankLines(extraRaw)
				promptParts := append([]string{firstPromptLine}, extraPromptLines...)
				prompt = strings.Join(promptParts, "\n")
				cur = dupIdx + 1
				for _, el := range extraPromptLines {
					elTrim := strings.TrimSpace(el)
					if elTrim == "" {
						continue
					}
					for cur < endLine && strings.TrimSpace(lines[cur]) == "" {
						cur++
					}
					if cur < endLine && strings.TrimSpace(lines[cur]) == elTrim {
						cur++
					}
				}
			} else {
				// Check if firstPromptLine is a space-flattened version of multi-line duplicate block
				acc := ""
				matchedEnd := -1
				for k := cur; k < searchLimit; k++ {
					lt := strings.TrimSpace(lines[k])
					if lt == "" {
						continue
					}
					if acc == "" {
						acc = lt
					} else {
						acc += " " + lt
					}
					if acc == firstPromptLine {
						matchedEnd = k
						break
					}
					if !strings.HasPrefix(firstPromptLine, acc) {
						break
					}
				}
				if matchedEnd != -1 {
					prompt = strings.Join(trimBlankLines(lines[cur:matchedEnd+1]), "\n")
					cur = matchedEnd + 1
				} else {
					prompt = firstPromptLine
				}
			}
		} else if cur < endLine {
			firstPromptLine = strings.TrimSpace(lines[cur])
			cur++
			for cur < endLine && strings.TrimSpace(lines[cur]) == "" {
				cur++
			}
			if cur < endLine && strings.TrimSpace(lines[cur]) == firstPromptLine {
				cur++
			}
			prompt = firstPromptLine
		}

		for cur < endLine && strings.TrimSpace(lines[cur]) == "" {
			cur++
		}
		for cur < endLine {
			lt := strings.TrimSpace(lines[cur])
			if reGeminiSaid.MatchString(lt) || strings.EqualFold(lt, "show thinking") || strings.EqualFold(lt, "hide thinking") {
				cur++
				for cur < endLine && strings.TrimSpace(lines[cur]) == "" {
					cur++
				}
				continue
			}
			break
		}

		respLines := trimBlankLines(lines[cur:endLine])
		turnBlock := fmt.Sprintf("User:\n%s", prompt)
		if len(respLines) > 0 {
			turnBlock += fmt.Sprintf("\n\nGemini:\n%s", strings.Join(respLines, "\n"))
		}
		blocks = append(blocks, turnBlock)
	}

	result := strings.Join(blocks, "\n\n")
	if h1TitleLine != "" {
		return h1TitleLine + "\n\n" + result
	}
	return result
}

func trimBlankLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	if start >= end {
		return nil
	}
	return lines[start:end]
}

// ExtractTitle finds the best title from the content and returns (title, remainingContent).
// Priority order:
// 1. chatlog.ExtractHashTitle (first or last # Title line)
// 2. Explicit # H1 heading in content (preferred over generic Jina Title: metadata)
// 3. Non-generic Jina "Title: <name>" header line
// 4. Fallback string
// 5. First non-empty line (capped to 50 runes)
func ExtractTitle(text string, fallback string) (string, string) {
	// 1. Try chatlog.ExtractHashTitle
	if title, clean := chatlog.ExtractHashTitle(text); title != "" {
		cleanedTitle := cleanInvisibleMarks(title)
		if cleanedTitle != "" {
			return cleanedTitle, clean
		}
	}

	lines := strings.Split(text, "\n")

	// Scan for Jina Title: header and first # H1 heading in content
	jinaTitleIdx := -1
	jinaTitle := ""
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if m := reJinaTitle.FindStringSubmatch(trimmed); len(m) > 1 {
			candidate := cleanInvisibleMarks(m[1])
			if candidate != "" {
				jinaTitleIdx = i
				jinaTitle = candidate
				break
			}
		}
	}

	h1Idx := -1
	h1Title := ""
	nonEmptyCount := 0
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || isJinaHeaderLine(trimmed) {
			continue
		}
		nonEmptyCount++
		if strings.HasPrefix(trimmed, "# ") {
			candidate := cleanInvisibleMarks(strings.TrimPrefix(trimmed, "# "))
			if candidate != "" && !strings.HasPrefix(candidate, "#") {
				h1Idx = i
				h1Title = candidate
				break
			}
		}
		if nonEmptyCount > 30 {
			break
		}
	}

	// 2. Prefer explicit # H1 heading if present
	if h1Title != "" {
		newLines := make([]string, 0, len(lines)-1)
		for i, l := range lines {
			if i == h1Idx || i == jinaTitleIdx {
				continue
			}
			newLines = append(newLines, l)
		}
		return h1Title, strings.Join(newLines, "\n")
	}

	// 3. Use Jina Title: header if non-empty and not a generic site title
	if jinaTitle != "" && !isGenericSiteTitle(jinaTitle) {
		newLines := make([]string, 0, len(lines)-1)
		newLines = append(newLines, lines[:jinaTitleIdx]...)
		newLines = append(newLines, lines[jinaTitleIdx+1:]...)
		return jinaTitle, strings.Join(newLines, "\n")
	}

	// Strip generic Jina Title: line if present so it isn't picked up as first line
	if jinaTitleIdx >= 0 {
		newLines := make([]string, 0, len(lines)-1)
		newLines = append(newLines, lines[:jinaTitleIdx]...)
		newLines = append(newLines, lines[jinaTitleIdx+1:]...)
		lines = newLines
		text = strings.Join(lines, "\n")
	}

	// 4. Fallback if specified
	if fallback != "" {
		return fallback, text
	}

	// 5. First non-empty non-header line
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !isJinaHeaderLine(trimmed) {
			runes := []rune(trimmed)
			if len(runes) > 50 {
				return string(runes[:50]), text
			}
			return trimmed, text
		}
	}

	return "Web Capture", text
}

// FormatChatLogCard generates markdown card with YAML frontmatter for chat_log cards.
func FormatChatLogCard(id string, title string, sourceURL string, body string, parsed *chatlog.ParsedChatLog, createdAt time.Time) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", id))
	sb.WriteString("type: chat_log\n")
	if title != "" {
		if strings.ContainsAny(title, ":#\"'\n\r\t[]{}") {
			sb.WriteString(fmt.Sprintf("title: %q\n", title))
		} else {
			sb.WriteString(fmt.Sprintf("title: %s\n", title))
		}
	}
	sb.WriteString(fmt.Sprintf("source: %s\n", sourceURL))
	sb.WriteString(fmt.Sprintf("source_url: %s\n", sourceURL))

	if parsed != nil && len(parsed.Participants) > 0 {
		var quoted []string
		for _, p := range parsed.Participants {
			if strings.ContainsAny(p, ",:#\"'[]{}") {
				quoted = append(quoted, fmt.Sprintf("%q", p))
			} else {
				quoted = append(quoted, p)
			}
		}
		sb.WriteString(fmt.Sprintf("participants: [%s]\n", strings.Join(quoted, ", ")))
	}
	if parsed != nil && parsed.StartedAt != nil {
		sb.WriteString(fmt.Sprintf("started_at: %s\n", parsed.StartedAt.Format(time.RFC3339)))
	}
	if parsed != nil && parsed.EndedAt != nil {
		sb.WriteString(fmt.Sprintf("ended_at: %s\n", parsed.EndedAt.Format(time.RFC3339)))
	}
	sb.WriteString(fmt.Sprintf("created_at: %s\n", createdAt.Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(body))
	sb.WriteString("\n")
	return sb.String()
}

// FormatNoteCard generates markdown card with YAML frontmatter for note cards.
func FormatNoteCard(id string, title string, sourceURL string, body string, createdAt time.Time) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", id))
	sb.WriteString("type: note\n")
	if title != "" {
		if strings.ContainsAny(title, ":#\"'\n\r\t[]{}") {
			sb.WriteString(fmt.Sprintf("title: %q\n", title))
		} else {
			sb.WriteString(fmt.Sprintf("title: %s\n", title))
		}
	}
	sb.WriteString(fmt.Sprintf("source: %s\n", sourceURL))
	sb.WriteString(fmt.Sprintf("source_url: %s\n", sourceURL))
	sb.WriteString(fmt.Sprintf("created: %s\n", createdAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("created_at: %s\n", createdAt.Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(body))
	sb.WriteString("\n")
	return sb.String()
}
