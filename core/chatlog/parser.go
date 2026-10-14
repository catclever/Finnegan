package chatlog

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Message represents a single parsed statement in a chat dialogue.
type Message struct {
	Speaker   string     `json:"speaker"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
	Content   string     `json:"content"`
}

// ParsedChatLog contains all structured dialogue metadata lifted from raw chat records.
type ParsedChatLog struct {
	Participants []string   `json:"participants"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	EndedAt      *time.Time `json:"endedAt,omitempty"`
	Messages     []Message  `json:"messages"`
	Coalesced    string     `json:"coalesced"`
	Images       []string   `json:"images,omitempty"`
}

var (
	// WeChat Chinese date format: 2026年10月14日  8:15 or 2026年10月14日 08:15:30
	reWeChatDate = regexp.MustCompile(`^(\d{4})年(\d{1,2})月(\d{1,2})日\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$`)
	// Standard ISO / numeric date format: 2026-10-14 08:15 or 2026/10/14 8:15:00
	reStandardDate = regexp.MustCompile(`^(\d{4})[-/](\d{1,2})[-/](\d{1,2})\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$`)
	// Month-day without year: 10月14日  8:15
	reMonthDayTime = regexp.MustCompile(`^(\d{1,2})月(\d{1,2})日\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$`)

	// Single line header regex: e.g. "巧先生 2026年10月14日  8:15" or "巧先生 2026-10-14 08:15"
	reSingleLineWeChat = regexp.MustCompile(`^([^\n\d]+?)\s+(\d{4}年\d{1,2}月\d{1,2}日\s+\d{1,2}:\d{2}(?::\d{2})?)$`)
	reSingleLineStandard = regexp.MustCompile(`^([^\n\d]+?)\s+(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)$`)
	// Bracketed timestamp header: "[2026-10-14 08:15:00] 巧先生:" or "巧先生 [2026-10-14 08:15:00]:"
	reBracketTimePrefix = regexp.MustCompile(`^\[(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)\]\s*(.+?):?$`)
	reBracketTimeSuffix = regexp.MustCompile(`^(.+?)\s*\[(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)\]:?$`)
	// Simple colon speaker header: "巧先生:"
	reColonSpeaker = regexp.MustCompile(`^([^\n:]{1,60}):\s*$`)
)

// ParseTimestamp attempts to parse various chat log timestamp strings into time.Time.
func ParseTimestamp(s string) (*time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}

	// 1. WeChat Chinese format (CST timezone: UTC+8)
	if m := reWeChatDate.FindStringSubmatch(s); len(m) > 0 {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		hour, _ := strconv.Atoi(m[4])
		min, _ := strconv.Atoi(m[5])
		sec := 0
		if m[6] != "" {
			sec, _ = strconv.Atoi(m[6])
		}
		loc := time.FixedZone("CST", 8*3600)
		t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
		return &t, true
	}

	// 2. Standard numeric format
	if m := reStandardDate.FindStringSubmatch(s); len(m) > 0 {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		hour, _ := strconv.Atoi(m[4])
		min, _ := strconv.Atoi(m[5])
		sec := 0
		if m[6] != "" {
			sec, _ = strconv.Atoi(m[6])
		}
		loc := time.FixedZone("CST", 8*3600)
		t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
		return &t, true
	}

	// 3. Month/Day format (defaults to current year)
	if m := reMonthDayTime.FindStringSubmatch(s); len(m) > 0 {
		year := time.Now().Year()
		month, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		hour, _ := strconv.Atoi(m[3])
		min, _ := strconv.Atoi(m[4])
		sec := 0
		if m[5] != "" {
			sec, _ = strconv.Atoi(m[5])
		}
		loc := time.FixedZone("CST", 8*3600)
		t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
		return &t, true
	}

	// 4. RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t, true
	}

	return nil, false
}

// IsChatLog returns true if the input text exhibits distinct chat dialogue patterns.
func IsChatLog(text string) bool {
	clean := strings.ReplaceAll(text, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")
	lines := strings.Split(clean, "\n")
	if len(lines) < 2 {
		return false
	}

	matchCount := 0
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Check WeChat 2-line header pattern
		if _, ok := ParseTimestamp(line); ok && i > 0 {
			prev := strings.TrimSpace(lines[i-1])
			if prev != "" && len(prev) <= 60 && !strings.HasPrefix(prev, "#") && !strings.HasPrefix(prev, "---") {
				matchCount++
			}
		}

		// Check single-line timestamp header
		if reSingleLineWeChat.MatchString(line) || reSingleLineStandard.MatchString(line) ||
			reBracketTimePrefix.MatchString(line) || reBracketTimeSuffix.MatchString(line) {
			matchCount++
		}

		// Check colon speaker pattern
		if reColonSpeaker.MatchString(line) {
			matchCount++
		}
	}

	// A valid chat log should have at least 1 two-line header or at least 2 colon/single-line headers
	return matchCount >= 1
}

type rawHeader struct {
	speakerLineIdx int
	speaker        string
	timestamp      *time.Time
	contentStart   int
}

// Parse parses raw dialogue text into structured messages, lifted timestamps, and coalesced text.
func Parse(text string) (*ParsedChatLog, error) {
	clean := strings.ReplaceAll(text, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")
	lines := strings.Split(clean, "\n")

	if len(lines) == 0 {
		return nil, fmt.Errorf("empty chat log")
	}

	var headers []rawHeader

	// Pass 1: Try finding two-line headers (WeChat style: Speaker \n Timestamp)
	for i := 0; i < len(lines); i++ {
		ts, ok := ParseTimestamp(lines[i])
		if ok && i > 0 {
			prev := strings.TrimSpace(lines[i-1])
			if prev != "" && len(prev) <= 60 && !strings.HasPrefix(prev, "#") && !strings.HasPrefix(prev, "---") {
				headers = append(headers, rawHeader{
					speakerLineIdx: i - 1,
					speaker:        prev,
					timestamp:      ts,
					contentStart:   i + 1,
				})
			}
		}
	}

	// Pass 2: If no two-line headers found, check single-line headers
	if len(headers) == 0 {
		for i := 0; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}

			if m := reSingleLineWeChat.FindStringSubmatch(line); len(m) > 0 {
				ts, _ := ParseTimestamp(m[2])
				headers = append(headers, rawHeader{
					speakerLineIdx: i,
					speaker:        strings.TrimSpace(m[1]),
					timestamp:      ts,
					contentStart:   i + 1,
				})
				continue
			}

			if m := reSingleLineStandard.FindStringSubmatch(line); len(m) > 0 {
				ts, _ := ParseTimestamp(m[2])
				headers = append(headers, rawHeader{
					speakerLineIdx: i,
					speaker:        strings.TrimSpace(m[1]),
					timestamp:      ts,
					contentStart:   i + 1,
				})
				continue
			}

			if m := reBracketTimePrefix.FindStringSubmatch(line); len(m) > 0 {
				ts, _ := ParseTimestamp(m[1])
				headers = append(headers, rawHeader{
					speakerLineIdx: i,
					speaker:        strings.TrimSpace(m[2]),
					timestamp:      ts,
					contentStart:   i + 1,
				})
				continue
			}

			if m := reBracketTimeSuffix.FindStringSubmatch(line); len(m) > 0 {
				ts, _ := ParseTimestamp(m[2])
				headers = append(headers, rawHeader{
					speakerLineIdx: i,
					speaker:        strings.TrimSpace(m[1]),
					timestamp:      ts,
					contentStart:   i + 1,
				})
				continue
			}

			if m := reColonSpeaker.FindStringSubmatch(line); len(m) > 0 {
				headers = append(headers, rawHeader{
					speakerLineIdx: i,
					speaker:        strings.TrimSpace(m[1]),
					timestamp:      nil,
					contentStart:   i + 1,
				})
				continue
			}
		}
	}

	if len(headers) == 0 {
		return nil, fmt.Errorf("no dialogue headers found in text")
	}

	// Extract messages between headers
	var messages []Message
	for idx, h := range headers {
		endLine := len(lines)
		if idx+1 < len(headers) {
			endLine = headers[idx+1].speakerLineIdx
		}

		if h.contentStart > endLine {
			messages = append(messages, Message{
				Speaker:   h.speaker,
				Timestamp: h.timestamp,
				Content:   "",
			})
			continue
		}

		rawContentLines := lines[h.contentStart:endLine]

		// Trim leading empty lines
		start := 0
		for start < len(rawContentLines) && strings.TrimSpace(rawContentLines[start]) == "" {
			start++
		}

		// Trim trailing empty lines
		end := len(rawContentLines)
		for end > start && strings.TrimSpace(rawContentLines[end-1]) == "" {
			end--
		}

		content := ""
		if start < end {
			content = strings.Join(rawContentLines[start:end], "\n")
		}

		messages = append(messages, Message{
			Speaker:   h.speaker,
			Timestamp: h.timestamp,
			Content:   content,
		})
	}

	// Lift started_at & ended_at timestamps
	var startedAt *time.Time
	var endedAt *time.Time
	for _, m := range messages {
		if m.Timestamp != nil {
			if startedAt == nil {
				t := *m.Timestamp
				startedAt = &t
			}
			t := *m.Timestamp
			endedAt = &t
		}
	}

	// Extract distinct participants preserving order of appearance
	var participants []string
	seenParticipants := make(map[string]bool)
	for _, m := range messages {
		if m.Speaker != "" && !seenParticipants[m.Speaker] {
			seenParticipants[m.Speaker] = true
			participants = append(participants, m.Speaker)
		}
	}

	// Message coalescing
	coalesced := CoalesceMessages(messages)

	return &ParsedChatLog{
		Participants: participants,
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Messages:     messages,
		Coalesced:    coalesced,
	}, nil
}

// CoalesceMessages coalesces consecutive messages from the same speaker and removes inline timestamp clutter.
func CoalesceMessages(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}

	var blocks []string
	var currentSpeaker string
	var currentContents []string

	for _, m := range messages {
		if m.Speaker == currentSpeaker {
			if m.Content != "" {
				currentContents = append(currentContents, m.Content)
			}
		} else {
			if currentSpeaker != "" {
				body := strings.Join(currentContents, "\n")
				blocks = append(blocks, fmt.Sprintf("%s:\n%s", currentSpeaker, body))
			}
			currentSpeaker = m.Speaker
			if m.Content != "" {
				currentContents = []string{m.Content}
			} else {
				currentContents = []string{}
			}
		}
	}

	if currentSpeaker != "" {
		body := strings.Join(currentContents, "\n")
		blocks = append(blocks, fmt.Sprintf("%s:\n%s", currentSpeaker, body))
	}

	return strings.Join(blocks, "\n\n")
}

// FormatCardMarkdown generates the standardized Markdown card with YAML Frontmatter.
func FormatCardMarkdown(id string, title string, parsed *ParsedChatLog, images ...[]string) string {
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
	if len(parsed.Participants) > 0 {
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
	if parsed.StartedAt != nil {
		sb.WriteString(fmt.Sprintf("started_at: %s\n", parsed.StartedAt.Format(time.RFC3339)))
	}
	if parsed.EndedAt != nil {
		sb.WriteString(fmt.Sprintf("ended_at: %s\n", parsed.EndedAt.Format(time.RFC3339)))
	}
	if parsed.StartedAt != nil {
		sb.WriteString(fmt.Sprintf("created_at: %s\n", parsed.StartedAt.Format(time.RFC3339)))
	} else {
		sb.WriteString(fmt.Sprintf("created_at: %s\n", time.Now().Format(time.RFC3339)))
	}

	var cardImages []string
	if len(images) > 0 {
		cardImages = images[0]
	} else if parsed != nil && len(parsed.Images) > 0 {
		cardImages = parsed.Images
	}

	if len(cardImages) > 0 {
		sb.WriteString("images:\n")
		for _, img := range cardImages {
			sb.WriteString(fmt.Sprintf("  - %s\n", img))
		}
	}

	sb.WriteString("---\n\n")
	if parsed != nil {
		sb.WriteString(parsed.Coalesced)
	}
	sb.WriteString("\n")

	return sb.String()
}
