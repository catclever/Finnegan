package keywords

import (
	"strings"
	"testing"
)

func TestMergeKeywordsToMarkdown_Lossless(t *testing.T) {
	origMD := `---
id: 01J987654321
type: thought
created: 2026-10-17T14:50:00+08:00
updated: 2026-10-17T14:50:00+08:00
source: local
tags:
  - tech
  - search
images:
  - img1.png
publish: false
---

# Title of Thought

This is the original body text that must be preserved completely.
Line 2 with special characters: *italic* **bold** [link](https://example.com).
`

	profile := KeywordsProfile{
		"entity":            {"SearxNG", "PageRank"},
		"judgment":          {"Local-First sync paradox"},
		"event":             {},
		"emotion":           {},
		"question":          {"How to balance freshness and depth?"},
		"metaphor":          {},
		"symptom":           {},
		"methodology":       {},
		"counter_intuitive": {},
		"person_work":       {},
	}

	mergedBytes, err := MergeKeywordsToMarkdown([]byte(origMD), profile)
	if err != nil {
		t.Fatalf("MergeKeywordsToMarkdown failed: %v", err)
	}

	mergedStr := string(mergedBytes)

	// Verify existing fields are preserved
	expectedFields := []string{
		"id: 01J987654321",
		"type: thought",
		"created: 2026-10-17T14:50:00+08:00",
		"updated: 2026-10-17T14:50:00+08:00",
		"source: local",
		"tags:",
		"- tech",
		"- search",
		"images:",
		"- img1.png",
		"publish: false",
	}

	for _, field := range expectedFields {
		if !strings.Contains(mergedStr, field) {
			t.Errorf("Expected field %q not found in merged output:\n%s", field, mergedStr)
		}
	}

	// Verify keywords are inserted properly
	if !strings.Contains(mergedStr, "keywords:") {
		t.Errorf("keywords block not found in merged output")
	}
	if !strings.Contains(mergedStr, "SearxNG") || !strings.Contains(mergedStr, "PageRank") {
		t.Errorf("entity keywords missing")
	}
	if !strings.Contains(mergedStr, "Local-First sync paradox") {
		t.Errorf("judgment keywords missing")
	}
	if !strings.Contains(mergedStr, "How to balance freshness and depth?") {
		t.Errorf("question keywords missing")
	}

	// Verify non-triggered dimensions (empty arrays) are omitted from frontmatter
	if strings.Contains(mergedStr, "emotion:") || strings.Contains(mergedStr, "metaphor:") {
		t.Errorf("empty gated dimensions should not appear in frontmatter:\n%s", mergedStr)
	}

	// Verify body is intact
	if !strings.Contains(mergedStr, "# Title of Thought") {
		t.Errorf("Body title missing")
	}
	if !strings.Contains(mergedStr, "Line 2 with special characters: *italic* **bold** [link](https://example.com).") {
		t.Errorf("Body content modified or lost")
	}

	// Verify roundtrip extraction
	extracted, err := ExtractKeywordsFromMarkdown(mergedBytes)
	if err != nil {
		t.Fatalf("ExtractKeywordsFromMarkdown failed: %v", err)
	}

	if len(extracted["entity"]) != 2 || extracted["entity"][0] != "SearxNG" {
		t.Errorf("Extracted entity mismatch: %v", extracted["entity"])
	}
	if len(extracted["judgment"]) != 1 || extracted["judgment"][0] != "Local-First sync paradox" {
		t.Errorf("Extracted judgment mismatch: %v", extracted["judgment"])
	}
	if len(extracted["question"]) != 1 || extracted["question"][0] != "How to balance freshness and depth?" {
		t.Errorf("Extracted question mismatch: %v", extracted["question"])
	}
	if len(extracted["emotion"]) != 0 {
		t.Errorf("Extracted emotion should be empty: %v", extracted["emotion"])
	}
}

func TestMergeKeywordsToMarkdown_UpdateExistingKeywords(t *testing.T) {
	origMD := `---
id: 01J111111111
keywords:
  entity:
    - OldEntity
---

Body text.
`
	newProfile := KeywordsProfile{
		"entity":   {"NewEntity"},
		"judgment": {"NewJudgment"},
	}

	merged, err := MergeKeywordsToMarkdown([]byte(origMD), newProfile)
	if err != nil {
		t.Fatalf("MergeKeywordsToMarkdown failed: %v", err)
	}

	mergedStr := string(merged)
	if strings.Contains(mergedStr, "OldEntity") {
		t.Errorf("OldEntity should be overwritten: %s", mergedStr)
	}
	if !strings.Contains(mergedStr, "NewEntity") || !strings.Contains(mergedStr, "NewJudgment") {
		t.Errorf("New keywords missing: %s", mergedStr)
	}

	// Verify only one "keywords:" key exists
	if strings.Count(mergedStr, "keywords:") != 1 {
		t.Errorf("Expected exactly one keywords block, found %d", strings.Count(mergedStr, "keywords:"))
	}
}

func TestMergeKeywordsToMarkdown_NoPriorFrontmatter(t *testing.T) {
	origMD := `# Just a Plain Note

No prior frontmatter here.
`
	profile := KeywordsProfile{
		"entity": {"Golang"},
	}

	merged, err := MergeKeywordsToMarkdown([]byte(origMD), profile)
	if err != nil {
		t.Fatalf("MergeKeywordsToMarkdown failed: %v", err)
	}

	mergedStr := string(merged)
	if !strings.HasPrefix(mergedStr, "---\n") {
		t.Errorf("Expected leading frontmatter delimiter: %s", mergedStr)
	}
	if !strings.Contains(mergedStr, "keywords:") || !strings.Contains(mergedStr, "Golang") {
		t.Errorf("Keywords missing from generated frontmatter: %s", mergedStr)
	}
	if !strings.Contains(mergedStr, "# Just a Plain Note\n\nNo prior frontmatter here.") {
		t.Errorf("Original body lost: %s", mergedStr)
	}
}

func TestMergeKeywordsToMarkdown_EmptyProfile(t *testing.T) {
	origMD := `---
id: 01J222222222
---

Hello world
`
	profile := KeywordsProfile{
		"entity":   {},
		"judgment": {},
	}

	merged, err := MergeKeywordsToMarkdown([]byte(origMD), profile)
	if err != nil {
		t.Fatalf("MergeKeywordsToMarkdown failed: %v", err)
	}

	mergedStr := string(merged)
	if !strings.Contains(mergedStr, "id: 01J222222222") {
		t.Errorf("id lost: %s", mergedStr)
	}
	if !strings.Contains(mergedStr, "Hello world") {
		t.Errorf("body lost: %s", mergedStr)
	}
}
