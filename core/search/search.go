package search

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

// Card defines the searchable consciousness stream card model.
type Card struct {
	ID             string    `json:"id"`
	Type           string    `json:"type,omitempty"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Tags           []string  `json:"tags,omitempty"`
	SourceURL      string    `json:"sourceUrl,omitempty"`
	OriginalText   string    `json:"originalText,omitempty"`
	TranslatedText string    `json:"translatedText,omitempty"`
	EchoOf         string    `json:"echoOf,omitempty"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
}

// isWordBoundary checks if a rune is a whitespace or punctuation delimiter.
func isWordBoundary(r rune) bool {
	if unicode.IsSpace(r) || unicode.IsPunct(r) {
		return true
	}
	switch r {
	case '-', '_', '/', '\\', '.', ',', ':', ';', '(', ')', '[', ']', '{', '}', '"', '\'', '`', '#', '@', '|':
		return true
	default:
		return false
	}
}

// MatchScore calculates a fuzzy match score between target and query strings.
// It returns (score, true) if query matches target, or (0, false) if no match.
// Scoring rules:
// - Lowercase comparison for case-insensitivity.
// - Exact contiguous substring matches yield the highest scores (>= 100).
// - Subsequence matching: query runes must appear in order in target.
//   Includes contiguous run bonuses, word boundary bonuses, compactness bonuses,
//   and penalties for distance/span between first and last matched characters.
func MatchScore(target, query string) (score float64, matched bool) {
	targetLower := strings.ToLower(target)
	queryLower := strings.ToLower(query)

	tRunes := []rune(targetLower)
	qRunes := []rune(queryLower)

	if len(qRunes) == 0 {
		return 0, true
	}
	if len(tRunes) == 0 || len(qRunes) > len(tRunes) {
		return 0, false
	}

	// 1. Exact contiguous substring match check
	subIdx := strings.Index(targetLower, queryLower)
	if subIdx >= 0 {
		runeIdx := len([]rune(targetLower[:subIdx]))

		// Base score for contiguous substring match: guaranteed >= 100
		baseScore := 100.0 + float64(len(qRunes))*5.0

		// Bonus for full exact match
		if targetLower == queryLower {
			baseScore += 50.0
		} else if runeIdx == 0 {
			// Prefix match bonus
			baseScore += 25.0
		} else if runeIdx > 0 && isWordBoundary(tRunes[runeIdx-1]) {
			// Word boundary bonus
			baseScore += 15.0
		}

		// Compactness bonus: query length / target length
		compactness := float64(len(qRunes)) / float64(len(tRunes))
		baseScore += compactness * 20.0

		// Minor position penalty (matches closer to start are preferred)
		posPenalty := float64(runeIdx) * 0.05
		score = baseScore - posPenalty
		if score < 100.0 {
			score = 100.0
		}
		return score, true
	}

	// 2. Subsequence matching: query runes must appear in order in target
	qi := 0
	for _, r := range tRunes {
		if r == qRunes[qi] {
			qi++
			if qi == len(qRunes) {
				break
			}
		}
	}
	if qi < len(qRunes) {
		return 0, false
	}

	// Find the best scoring subsequence alignment
	bestScore := 0.0
	matchedStarts := 0

	for s := 0; s < len(tRunes); s++ {
		if tRunes[s] != qRunes[0] {
			continue
		}
		matchedStarts++
		if matchedStarts > 50 {
			break
		}

		// Greedily match remaining query runes starting from index s
		indices := make([]int, len(qRunes))
		indices[0] = s
		curQ := 1
		for curT := s + 1; curT < len(tRunes) && curQ < len(qRunes); curT++ {
			if tRunes[curT] == qRunes[curQ] {
				indices[curQ] = curT
				curQ++
			}
		}

		if curQ < len(qRunes) {
			continue
		}

		// Base score for subsequence match
		alignScore := 20.0 + float64(len(qRunes))*2.0

		// Bonus for contiguous consecutive matched characters
		streak := 0
		for k := 1; k < len(indices); k++ {
			if indices[k] == indices[k-1]+1 {
				streak++
				alignScore += float64(streak) * 6.0
			} else {
				streak = 0
			}
		}

		// Bonus for matching at word boundaries
		for _, pos := range indices {
			if pos == 0 || isWordBoundary(tRunes[pos-1]) {
				alignScore += 4.0
			}
		}

		// Compactness and distance/span calculations
		first := indices[0]
		last := indices[len(indices)-1]
		span := last - first + 1

		compactness := float64(len(qRunes)) / float64(span)
		alignScore += compactness * 15.0

		// Penalty for span/distance between first and last matched characters
		spanPenalty := float64(span-len(qRunes)) * 0.4
		alignScore -= spanPenalty

		// Minor start position penalty
		startPenalty := float64(first) * 0.05
		alignScore -= startPenalty

		if alignScore > bestScore {
			bestScore = alignScore
		}
	}

	if bestScore <= 0 {
		bestScore = 1.0
	}
	// Cap non-contiguous subsequence score below 95.0 so contiguous matches always rank higher
	if bestScore > 95.0 {
		bestScore = 95.0
	}

	return bestScore, true
}

// SearchCards searches cards using fuzzy subsequence matching with weighting on Title, Content, and Tags.
// If query is empty, it returns all cards sorted by CreatedAt descending.
// Matched cards are returned sorted by score descending.
func SearchCards(query string, cards []Card) []Card {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		res := make([]Card, len(cards))
		copy(res, cards)
		sort.SliceStable(res, func(i, j int) bool {
			if !res[i].CreatedAt.Equal(res[j].CreatedAt) {
				return res[i].CreatedAt.After(res[j].CreatedAt)
			}
			return res[i].ID < res[j].ID
		})
		return res
	}

	type scoredCard struct {
		card  Card
		score float64
	}

	scored := make([]scoredCard, 0, len(cards))

	for _, card := range cards {
		titleScore, titleMatched := MatchScore(card.Title, trimmedQuery)
		contentScore, contentMatched := MatchScore(card.Content, trimmedQuery)

		tagScore := 0.0
		tagMatched := false
		for _, tag := range card.Tags {
			s, m := MatchScore(tag, trimmedQuery)
			if m && s > tagScore {
				tagScore = s
				tagMatched = true
			}
		}

		if !titleMatched && !contentMatched && !tagMatched {
			continue
		}

		// Title has highest weight (3.0), Tags have medium weight (2.0), Content has base weight (1.0)
		totalScore := 0.0
		if titleMatched {
			totalScore += titleScore * 3.0
		}
		if tagMatched {
			totalScore += tagScore * 2.0
		}
		if contentMatched {
			totalScore += contentScore * 1.0
		}

		scored = append(scored, scoredCard{
			card:  card,
			score: totalScore,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].card.CreatedAt.Equal(scored[j].card.CreatedAt) {
			return scored[i].card.CreatedAt.After(scored[j].card.CreatedAt)
		}
		return scored[i].card.ID < scored[j].card.ID
	})

	results := make([]Card, len(scored))
	for i, sc := range scored {
		results[i] = sc.card
	}

	return results
}
