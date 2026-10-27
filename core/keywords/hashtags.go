package keywords

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// hashtagRegex matches tokens like #tag, #WebKitGTK, #跨平台UI, #v1.0.0, #松弛感
	hashtagRegex = regexp.MustCompile(`#([\p{L}\p{N}_\-\.]+)`)
)

// StripAndExtractHashtags extracts clean hashtags from text, strips trailing/standalone
// hashtags completely from the card body, and strips '#' from inline hashtags to preserve
// grammatical fluency.
func StripAndExtractHashtags(text string) (string, []string) {
	if strings.TrimSpace(text) == "" {
		return "", nil
	}

	// 1. Extract all hashtags in order of appearance
	rawMatches := hashtagRegex.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var extractedTags []string
	for _, m := range rawMatches {
		if len(m) > 1 {
			tag := strings.TrimSpace(m[1])
			tag = strings.TrimRight(tag, ".,!?;:，。！？")
			if tag != "" && !seen[tag] {
				seen[tag] = true
				extractedTags = append(extractedTags, tag)
			}
		}
	}

	if len(extractedTags) == 0 {
		return strings.TrimSpace(text), nil
	}

	// 2. Identify standalone hashtag lines and trailing hashtags.
	// Split into lines.
	lines := strings.Split(text, "\n")
	var remainingLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			remainingLines = append(remainingLines, "")
			continue
		}

		// Check if line is purely standalone hashtags (e.g. "#tag1 #tag2", "#tag1, #tag2")
		if isStandaloneHashtagLine(trimmedLine) {
			// Strip this line completely!
			continue
		}

		// Check if the end of this line has trailing hashtags:
		// e.g. "Sentence text。 #tag1 #tag2" or "Sentence text #tag1 #tag2" (multiple trailing hashtags)
		cleanedLine := stripTrailingHashtagsFromLine(trimmedLine)
		remainingLines = append(remainingLines, cleanedLine)
	}

	// Remove trailing empty lines
	for len(remainingLines) > 0 && strings.TrimSpace(remainingLines[len(remainingLines)-1]) == "" {
		remainingLines = remainingLines[:len(remainingLines)-1]
	}

	intermediateText := strings.Join(remainingLines, "\n")

	// 3. For any remaining inline hashtags, strip '#' to preserve grammatical fluency.
	// E.g. "今天测试了 #WebKitGTK 4.1 和 #跨平台UI" -> "今天测试了 WebKitGTK 4.1 和 跨平台UI"
	cleanBody := hashtagRegex.ReplaceAllString(intermediateText, "$1")
	cleanBody = strings.TrimSpace(cleanBody)

	// If cleanBody became empty (e.g. the text only had hashtags), preserve the stripped text
	if cleanBody == "" && len(extractedTags) > 0 {
		cleanBody = strings.Join(extractedTags, " ")
	}

	return cleanBody, extractedTags
}

// isStandaloneHashtagLine returns true if line consists solely of hashtags and whitespace/separators.
func isStandaloneHashtagLine(line string) bool {
	matches := hashtagRegex.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return false
	}

	// Check if non-matched characters are only whitespace or punctuation delimiters (, ; ， ； - |)
	lastIdx := 0
	for _, m := range matches {
		between := line[lastIdx:m[0]]
		if !isSeparatorOnly(between) {
			return false
		}
		lastIdx = m[1]
	}
	remaining := line[lastIdx:]
	return isSeparatorOnly(remaining)
}

func isSeparatorOnly(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == ',' || r == ';' || r == '，' || r == '；' || r == '|' || r == '-' || r == '、' {
			continue
		}
		return false
	}
	return true
}

// stripTrailingHashtagsFromLine checks if the end of the line has trailing standalone hashtags
// (e.g. preceded by sentence-ending punctuation like 。/./!/? plus whitespace, or multiple consecutive hashtags at the end).
func stripTrailingHashtagsFromLine(line string) string {
	matches := hashtagRegex.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	// Check from the end: find consecutive hashtags at the line's tail
	tailStart := -1
	currentEnd := len(line)

	// Walk backwards through matches
	countAtTail := 0
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		after := line[m[1]:currentEnd]
		if !isSeparatorOnly(after) {
			break
		}
		tailStart = m[0]
		currentEnd = m[0]
		countAtTail++
	}

	if tailStart == -1 || countAtTail == 0 {
		return line
	}

	before := line[:tailStart]
	trimmedBefore := strings.TrimRight(before, " \t")

	// If preceded by sentence-ending punctuation (., 。, !, ?, ！, ？, ;, ；), or if there are >= 2 trailing hashtags:
	// Then it is clearly trailing metadata, strip it!
	isPrecededByPunct := false
	if len(trimmedBefore) > 0 {
		lastRune, _ := utf8.DecodeLastRuneInString(trimmedBefore)
		switch lastRune {
		case '.', '。', '!', '?', '！', '？', ';', '；', ':', '：':
			isPrecededByPunct = true
		}
	}

	if isPrecededByPunct || countAtTail >= 2 {
		return trimmedBefore
	}

	// Otherwise, leave it as inline to have its '#' stripped
	return line
}

// Known dimension lexicons
var (
	knownMoods = map[string]bool{
		"松弛感": true, "抽离感": true, "吐槽欲": true, "通透": true, "戏谑": true,
		"会心一笑": true, "烦躁": true, "踏实": true, "荒谬感": true, "物哀": true,
		"安宁": true, "焦虑": true, "欣喜": true, "疲惫": true, "无力感": true,
		"落寞": true, "释然": true, "治愈": true, "emo": true, "消沉": true,
		"亢奋": true, "孤独感": true, "虚无感": true, "归属感": true, "钝感": true,
		"挫败感": true, "秩序感": true, "分寸感": true, "边界感": true, "安全感": true,
		"幸福感": true, "失落感": true, "紧迫感": true, "表达欲": true, "分享欲": true,
		"掌控欲": true, "胜负欲": true, "求知欲": true, "探索欲": true,
	}

	knownMotifs = map[string]bool{
		"逃席": true, "吐槽模型": true, "创造需求": true, "薅羊毛": true, "体验新玩具": true,
		"额度焦虑": true, "熟人借钱": true, "调侃调休": true, "深夜排错": true, "摸鱼": true,
		"带薪拉屎": true, "甩锅": true, "画大饼": true, "内卷": true, "躺平": true,
		"打卡": true, "复盘": true, "对齐": true, "排错": true, "抓包": true,
		"踩坑": true, "填坑": true, "交接": true, "碎碎念": true, "立flag": true,
	}

	knownConcepts = map[string]bool{
		"伪提效": true, "存量陷阱": true, "机械迎合": true, "以Bug改Bug": true, "前端即逻辑": true,
		"平台套路": true, "人情味": true, "朋友圈人设": true, "信息茧房": true, "消费主义": true,
		"技术负债": true, "极简主义": true, "长期主义": true, "认知差": true, "信息差": true,
		"幸存者偏差": true, "价值认同": true, "路径依赖": true, "黑盒假象": true, "数字游民": true,
	}
)

// RouteTagToDimension routes a tag to its known cognitive dimension (object/motif/mood/concept)
// or falls back to object.
func RouteTagToDimension(tag string) string {
	clean := strings.TrimSpace(tag)
	clean = strings.TrimPrefix(clean, "#")
	clean = strings.TrimSuffix(clean, "*")
	clean = strings.TrimSpace(clean)

	if clean == "" {
		return DimensionObject
	}

	// Exact dictionary lookups
	lower := strings.ToLower(clean)
	if knownMoods[clean] || knownMoods[lower] {
		return DimensionMood
	}
	if knownMotifs[clean] || knownMotifs[lower] {
		return DimensionMotif
	}
	if knownConcepts[clean] || knownConcepts[lower] {
		return DimensionConcept
	}

	// Morphological / heuristic rules
	runes := []rune(clean)
	runeLen := len(runes)

	// Mood heuristics: 2~5 chars ending with "感" or 2~4 chars ending with "欲"
	if runeLen >= 2 && runeLen <= 5 && runes[runeLen-1] == '感' {
		return DimensionMood
	}
	if runeLen >= 2 && runeLen <= 4 && runes[runeLen-1] == '欲' {
		return DimensionMood
	}

	// Concept heuristics: words ending with 主义, 陷阱, 偏差, 悖论, 套路
	if strings.HasSuffix(clean, "主义") || strings.HasSuffix(clean, "陷阱") ||
		strings.HasSuffix(clean, "偏差") || strings.HasSuffix(clean, "悖论") ||
		strings.HasSuffix(clean, "套路") || strings.HasSuffix(clean, "思维") {
		return DimensionConcept
	}

	// Fallback to object
	return DimensionObject
}

// RouteTagsToKeywords routes each tag to its cognitive dimension, appending '*' suffix
// (e.g. 'WebKitGTK 4.1*', '松弛感*') in frontmatter keywords.
func RouteTagsToKeywords(tags []string) KeywordsProfile {
	profile := make(KeywordsProfile)
	seen := make(map[string]map[string]bool)

	for _, tag := range tags {
		clean := strings.TrimSpace(tag)
		clean = strings.TrimPrefix(clean, "#")
		clean = strings.TrimSuffix(clean, "*")
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}

		dim := RouteTagToDimension(clean)
		tagWithAsterisk := clean + "*"

		if seen[dim] == nil {
			seen[dim] = make(map[string]bool)
		}
		lower := strings.ToLower(clean)
		if seen[dim][lower] {
			continue
		}
		seen[dim][lower] = true
		profile[dim] = append(profile[dim], tagWithAsterisk)
	}

	return profile
}
