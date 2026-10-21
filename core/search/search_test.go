package search

import (
	"testing"
	"time"
)

func TestMatchScore_Contiguous(t *testing.T) {
	// Full match
	scoreFull, matched := MatchScore("Finnegan", "finnegan")
	if !matched || scoreFull < 150 {
		t.Fatalf("expected full match with score >= 150, got matched=%v, score=%f", matched, scoreFull)
	}

	// Prefix contiguous match
	scorePrefix, matched := MatchScore("Finnegan Omnibar", "finnegan")
	if !matched || scorePrefix < 100 {
		t.Fatalf("expected prefix match with score >= 100, got matched=%v, score=%f", matched, scorePrefix)
	}

	// Word boundary contiguous match
	scoreWord, matched := MatchScore("Finnegan Omnibar", "omnibar")
	if !matched || scoreWord < 100 {
		t.Fatalf("expected word boundary match with score >= 100, got matched=%v, score=%f", matched, scoreWord)
	}

	// Mid-word contiguous match
	scoreMid, matched := MatchScore("Finnegan Omnibar", "nibar")
	if !matched || scoreMid < 100 {
		t.Fatalf("expected mid-word contiguous match with score >= 100, got matched=%v, score=%f", matched, scoreMid)
	}

	// Word boundary should beat mid-word
	if scoreWord <= scoreMid {
		t.Fatalf("expected word boundary match score (%f) to be higher than mid-word match score (%f)", scoreWord, scoreMid)
	}
}

func TestMatchScore_Subsequence(t *testing.T) {
	// Acronym / subsequence match
	scoreSub, matched := MatchScore("Philosophy of Mind", "pom")
	if !matched {
		t.Fatalf("expected subsequence match for pom in 'Philosophy of Mind'")
	}
	if scoreSub <= 0 || scoreSub >= 100 {
		t.Fatalf("expected subsequence score between 0 and 100, got %f", scoreSub)
	}

	// Contiguous run vs gapped run
	scoreCompact, matchedCompact := MatchScore("abcdef", "bcd")
	scoreGapped, matchedGapped := MatchScore("abcdef", "bdf")
	if !matchedCompact || !matchedGapped {
		t.Fatalf("expected both to match")
	}
	if scoreCompact <= scoreGapped {
		t.Fatalf("expected compact match score (%f) to be higher than gapped match score (%f)", scoreCompact, scoreGapped)
	}
}

func TestMatchScore_Multilingual(t *testing.T) {
	// Chinese contiguous
	scoreZhContig, matched := MatchScore("布谷深网雷达", "深网")
	if !matched || scoreZhContig < 100 {
		t.Fatalf("expected Chinese contiguous match with score >= 100, got %v, %f", matched, scoreZhContig)
	}

	// Chinese subsequence
	scoreZhSub, matched := MatchScore("布谷深网雷达", "布网")
	if !matched || scoreZhSub <= 0 || scoreZhSub >= 100 {
		t.Fatalf("expected Chinese subsequence match, got %v, %f", matched, scoreZhSub)
	}

	// Chinese non-match
	_, matched = MatchScore("布谷深网雷达", "猫咪")
	if matched {
		t.Fatalf("expected no match for unrelated Chinese characters")
	}
}

func TestMatchScore_EdgeCases(t *testing.T) {
	// Empty query matches with score 0
	score, matched := MatchScore("anything", "")
	if !matched || score != 0 {
		t.Fatalf("expected empty query to match with score 0, got %v, %f", matched, score)
	}

	// Empty target does not match non-empty query
	_, matched = MatchScore("", "query")
	if matched {
		t.Fatalf("expected empty target not to match")
	}

	// Query longer than target does not match
	_, matched = MatchScore("hi", "hello")
	if matched {
		t.Fatalf("expected longer query not to match shorter target")
	}

	// Case insensitivity
	s1, m1 := MatchScore("ReSeArCh", "research")
	s2, m2 := MatchScore("research", "RESEARCH")
	if !m1 || !m2 || s1 != s2 {
		t.Fatalf("expected case-insensitive equality, got m1=%v, m2=%v, s1=%f, s2=%f", m1, m2, s1, s2)
	}
}

func TestSearchCards(t *testing.T) {
	now := time.Now()
	cards := []Card{
		{
			ID:        "card-1",
			Title:     "Quantum Computing Fundamentals",
			Content:   "Introduction to qubits and superposition",
			Tags:      []string{"physics", "computing"},
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "card-2",
			Title:     "Daily Journal Note",
			Content:   "Today I read about quantum computing advances in cryptography",
			Tags:      []string{"journal"},
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "card-3",
			Title:     "Artificial Intelligence Overview",
			Content:   "Deep neural networks and LLMs",
			Tags:      []string{"ai", "machine-learning"},
			CreatedAt: now,
		},
	}

	// 1. Empty query returns all cards sorted by CreatedAt descending
	allResults := SearchCards("", cards)
	if len(allResults) != len(cards) {
		t.Fatalf("expected all %d cards for empty query, got %d", len(cards), len(allResults))
	}
	if allResults[0].ID != "card-3" || allResults[1].ID != "card-2" || allResults[2].ID != "card-1" {
		t.Fatalf("expected empty query results to be sorted by CreatedAt descending, got order [%s, %s, %s]",
			allResults[0].ID, allResults[1].ID, allResults[2].ID)
	}

	// 2. Title match weighting > Content match weighting
	// "Quantum" appears in Card 1 title, and Card 2 content only.
	// Card 1 must be ranked first!
	qResults := SearchCards("Quantum", cards)
	if len(qResults) < 2 {
		t.Fatalf("expected at least 2 matches for 'Quantum', got %d", len(qResults))
	}
	if qResults[0].ID != "card-1" {
		t.Fatalf("expected title-match card-1 to be ranked #1, but got %s", qResults[0].ID)
	}

	// 3. Tag match
	tagResults := SearchCards("physics", cards)
	if len(tagResults) == 0 {
		t.Fatalf("expected match for tag 'physics'")
	}
	if tagResults[0].ID != "card-1" {
		t.Fatalf("expected card-1 for tag 'physics', got %s", tagResults[0].ID)
	}

	// 4. Subsequence search
	subResults := SearchCards("qc", cards) // Quantum Computing
	if len(subResults) == 0 {
		t.Fatalf("expected match for subsequence 'qc'")
	}
	if subResults[0].ID != "card-1" {
		t.Fatalf("expected card-1 to match 'qc', got %s", subResults[0].ID)
	}

	// 5. No match returns empty slice
	noResults := SearchCards("xylophone", cards)
	if len(noResults) != 0 {
		t.Fatalf("expected 0 matches for 'xylophone', got %d", len(noResults))
	}
}
