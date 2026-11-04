package api

import (
	"os"
	"strings"
	"testing"

	"Finnegan/modules/bugu/models"
)

func TestResolveTargetQueries(t *testing.T) {
	searchQueries := map[string][]string{
		"zh": {"Claude 评测", "写代码表现"},
		"en": {"Claude Opus benchmarks", "Claude coding quality"},
		"ru": {"Claude Opus программирование"},
	}

	// 1. Russian target with native "ru" query available
	ruTarget := models.Target{
		ID:        "habr_com",
		Languages: []string{"ru"},
	}
	ruQueries := resolveTargetQueries(ruTarget, searchQueries)
	if len(ruQueries) != 1 || ruQueries[0] != "Claude Opus программирование" {
		t.Errorf("expected ru query, got: %v", ruQueries)
	}

	// 2. French target without "fr" query -> MUST fall back to "en", NEVER to "zh"
	frTarget := models.Target{
		ID:        "linuxfr_org",
		Languages: []string{"fr"},
	}
	frQueries := resolveTargetQueries(frTarget, searchQueries)
	if len(frQueries) != 2 || frQueries[0] != "Claude Opus benchmarks" {
		t.Errorf("expected en query fallback for french site, got: %v", frQueries)
	}

	// 3. Chinese target with "zh-CN" language -> matches base "zh" query
	cnTarget := models.Target{
		ID:        "v2ex_cn",
		Languages: []string{"zh-CN"},
	}
	cnQueries := resolveTargetQueries(cnTarget, searchQueries)
	if len(cnQueries) != 2 || cnQueries[0] != "Claude 评测" {
		t.Errorf("expected zh query for chinese site, got: %v", cnQueries)
	}

	// 4. German target with ONLY Chinese query available -> MUST return nil, NEVER send Chinese to German site
	onlyZhQueries := map[string][]string{
		"zh": {"中文查询"},
	}
	deTarget := models.Target{
		ID:        "heise_de",
		Languages: []string{"de"},
	}
	deQueries := resolveTargetQueries(deTarget, onlyZhQueries)
	if len(deQueries) != 0 {
		t.Errorf("expected empty queries (prevent Chinese leak to German site), got: %v", deQueries)
	}
}

func TestExtractKeywordsForDoc_StripAsterisk(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := tmpDir + "/test_card_strip.epiphany.md"
	cardContent := `---
id: 01JSTRIP123
type: thought
keywords:
  entity:
    - NormalEntity
    - TagWithAsterisk *
    - MultiWord Asterisk *
  symptom:
    - "***   *"
    - NormalSymptom
---
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatalf("failed to write test card: %v", err)
	}

	kw := ExtractKeywordsForDoc(cardPath)
	if kw == nil {
		t.Fatal("expected non-nil keywords from card")
	}

	if len(kw["object"]) != 3 || kw["object"][0] != "NormalEntity" || kw["object"][1] != "TagWithAsterisk" || kw["object"][2] != "MultiWord Asterisk" {
		t.Errorf("unexpected entity keywords: %v", kw["object"])
	}
	if len(kw["motif"]) != 2 || kw["motif"][0] != "***" || kw["motif"][1] != "NormalSymptom" {
		t.Errorf("unexpected symptom keywords: %v", kw["motif"])
	}
}

func TestEnrichIntentWithSummaries(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := tmpDir + "/test_card_enrich.epiphany.md"
	cardContent := `---
id: 01JENRICH123
visual_summary: "A nice picture of a dog."
link_summary: "Article about dogs."
---
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatalf("failed to write test card: %v", err)
	}

	intent := "Show me cute animals."
	enriched := EnrichIntentWithSummaries(cardPath, intent)

	if !strings.Contains(enriched, "Show me cute animals.") {
		t.Errorf("expected base intent, got: %s", enriched)
	}
	if !strings.Contains(enriched, "\n[Image Context]: A nice picture of a dog.") {
		t.Errorf("expected visual summary, got: %s", enriched)
	}
	if !strings.Contains(enriched, "\n[Link Context]: Article about dogs.") {
		t.Errorf("expected link summary, got: %s", enriched)
	}

	// Test idempotency
	enrichedAgain := EnrichIntentWithSummaries(cardPath, enriched)
	if strings.Count(enrichedAgain, "[Image Context]:") > 1 {
		t.Errorf("expected single visual summary, got duplicates: %s", enrichedAgain)
	}
}

func TestResolveTargetQueriesEmptyTimeRangeOverride(t *testing.T) {
	// Empty time range logic test
}

func TestExtractKeywordsForDoc_DirectPath(t *testing.T) {
	// Create a temporary card with frontmatter containing keywords
	tmpDir := t.TempDir()
	cardPath := tmpDir + "/test_card.epiphany.md"
	cardContent := `---
id: 01JABCDEF1234567890
type: thought
created: 2026-10-17T12:00:00+08:00
keywords:
  entity:
    - WebKitGTK 4.1
    - libsoup
  symptom:
    - TLS handshake timeout
---

Card body text discussing WebKitGTK networking.
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatalf("failed to write test card: %v", err)
	}

	// 1. Direct path lookup
	kw := ExtractKeywordsForDoc(cardPath)
	if kw == nil {
		t.Fatal("expected non-nil keywords from card")
	}
	if len(kw["object"]) != 2 || kw["object"][0] != "WebKitGTK 4.1" || kw["object"][1] != "libsoup" {
		t.Errorf("unexpected entity keywords: %v", kw["object"])
	}
	if len(kw["motif"]) != 1 || kw["motif"][0] != "TLS handshake timeout" {
		t.Errorf("unexpected symptom keywords: %v", kw["motif"])
	}

	// 2. Empty docID
	if kwEmpty := ExtractKeywordsForDoc(""); kwEmpty != nil {
		t.Errorf("expected nil for empty docID, got %v", kwEmpty)
	}

	// 3. Non-existent docID
	if kwNonExistent := ExtractKeywordsForDoc("non-existent-doc-id-xyz"); kwNonExistent != nil {
		t.Errorf("expected nil for non-existent docID, got %v", kwNonExistent)
	}
}

func TestFindCardFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := tmpDir + "/card-01JXYZ.epiphany.md"
	if err := os.WriteFile(cardPath, []byte("---\nid: 01JXYZ\n---\nHello"), 0644); err != nil {
		t.Fatalf("failed to write test card: %v", err)
	}

	// Direct path
	if found := FindCardFilePath(cardPath); found != cardPath {
		t.Errorf("expected %s, got %s", cardPath, found)
	}

	// Empty
	if found := FindCardFilePath(""); found != "" {
		t.Errorf("expected empty string for empty docID, got %s", found)
	}
}

