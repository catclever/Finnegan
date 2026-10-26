package llm

import (
	"strings"
	"testing"
)

func TestFormatKeywordsProfile(t *testing.T) {
	// Empty or nil
	if res := FormatKeywordsProfile(nil); res != "" {
		t.Errorf("expected empty string for nil keywords, got %q", res)
	}
	if res := FormatKeywordsProfile(map[string][]string{}); res != "" {
		t.Errorf("expected empty string for empty keywords, got %q", res)
	}
	if res := FormatKeywordsProfile(map[string][]string{"object": {}}); res != "" {
		t.Errorf("expected empty string for empty slice, got %q", res)
	}

	// Populated profile - verify alphabetical ordering of dimensions: concept < mood < motif < object
	kw := map[string][]string{
		"object":  {"WebKitGTK 4.1", "libsoup"},
		"motif":   {"排查内存泄漏"},
		"mood":    {"烦躁", "抽离感"},
		"concept": {"以Bug改Bug"},
	}

	formatted := FormatKeywordsProfile(kw)
	expectedLines := []string{
		"- concept: 以Bug改Bug",
		"- mood: 烦躁, 抽离感",
		"- motif: 排查内存泄漏",
		"- object: WebKitGTK 4.1, libsoup",
	}

	for _, line := range expectedLines {
		if !strings.Contains(formatted, line) {
			t.Errorf("expected formatted profile to contain %q, but got:\n%s", line, formatted)
		}
	}

	// Verify order: concept < mood < motif < object
	idxConcept := strings.Index(formatted, "- concept:")
	idxMood := strings.Index(formatted, "- mood:")
	idxMotif := strings.Index(formatted, "- motif:")
	idxObject := strings.Index(formatted, "- object:")
	if !(idxConcept < idxMood && idxMood < idxMotif && idxMotif < idxObject) {
		t.Errorf("expected alphabetical order for dimensions, got:\n%s", formatted)
	}
}

func TestBuildFacetSearchPlanPrompt_WithKeywords(t *testing.T) {
	intent := "Investigate WebKitGTK memory leaks in Wayland"
	catalog := "ID: hn | Type: community\nID: github | Type: code"
	kw := map[string][]string{
		"object":  {"WebKitGTK", "Wayland"},
		"motif":   {"排查内存泄漏", "GNOME 46 release"},
		"mood":    {"烦躁"},
		"concept": {"以Bug改Bug", "性能劣化"},
	}

	prompt := BuildFacetSearchPlanPrompt(intent, catalog, kw)

	// 1. Current real year 2026 injection
	if !strings.Contains(prompt, "2026") {
		t.Errorf("expected prompt to contain current year '2026', got:\n%s", prompt)
	}

	// 2. User intent & catalog
	if !strings.Contains(prompt, intent) {
		t.Errorf("expected prompt to contain user intent")
	}
	if !strings.Contains(prompt, catalog) {
		t.Errorf("expected prompt to contain target catalog")
	}

	// 3. Keywords profile section
	if !strings.Contains(prompt, "KEYWORDS PROFILE:") {
		t.Errorf("expected prompt to contain KEYWORDS PROFILE: section")
	}
	if !strings.Contains(prompt, "- object: WebKitGTK, Wayland") {
		t.Errorf("expected object keywords in prompt")
	}
	if !strings.Contains(prompt, "- motif: 排查内存泄漏, GNOME 46 release") {
		t.Errorf("expected motif keywords in prompt")
	}
	if !strings.Contains(prompt, "- mood: 烦躁") {
		t.Errorf("expected mood keywords in prompt")
	}
	if !strings.Contains(prompt, "- concept: 以Bug改Bug, 性能劣化") {
		t.Errorf("expected concept keywords in prompt")
	}

	// 4. Dimension-based heuristics (4 Concrete Anchors)
	if !strings.Contains(prompt, "object (名物与通用指涉): Match technical documentation, code repositories, vertical encyclopedias") {
		t.Errorf("expected object heuristic in prompt")
	}
	if !strings.Contains(prompt, "motif (处境与母题): Prioritize real-time discussion platforms and forums (HN, Reddit, V2EX") {
		t.Errorf("expected motif heuristic in prompt")
	}
	if !strings.Contains(prompt, "mood (心境与心绪): Match personal indie blogs, IndieWeb, literary essays") {
		t.Errorf("expected mood heuristic in prompt")
	}
	if !strings.Contains(prompt, "concept (观念与议题): Match long-form analytical blogs, Substack, LessWrong") {
		t.Errorf("expected concept heuristic in prompt")
	}

	// 5. Time range instructions
	if !strings.Contains(prompt, "default_time_range") || !strings.Contains(prompt, "target_time_ranges") {
		t.Errorf("expected time range fields in prompt schema")
	}
}

func TestBuildFacetSearchPlanPrompt_WithoutKeywords(t *testing.T) {
	intent := "Explore indie web aesthetics"
	catalog := "ID: indie_web | Type: blog"

	prompt := BuildFacetSearchPlanPrompt(intent, catalog, nil)

	// Year 2026 must still be present
	if !strings.Contains(prompt, "2026") {
		t.Errorf("expected prompt to contain '2026'")
	}
	if !strings.Contains(prompt, intent) {
		t.Errorf("expected prompt to contain user intent")
	}
	// Keywords profile section should NOT be present when nil/empty
	if strings.Contains(prompt, "KEYWORDS PROFILE:") {
		t.Errorf("expected prompt to omit KEYWORDS PROFILE when empty")
	}
	// Time range instructions should still be present
	if !strings.Contains(prompt, "default_time_range") || !strings.Contains(prompt, "target_time_ranges") {
		t.Errorf("expected time range fields in prompt schema")
	}
}

func TestParseSearchPlanJSON_ValidWithTimeRanges(t *testing.T) {
	raw := "```json\n" + `{
  "selected_target_ids": ["hn", "github_issues", "indie_blog"],
  "search_queries": {
    "en": ["WebKitGTK Wayland buffer leak"],
    "zh": ["WebKitGTK 内存泄露"]
  },
  "default_time_range": "year",
  "target_time_ranges": {
    "hn": "month",
    "github_issues": "week",
    "indie_blog": ""
  }
}` + "\n```"

	plan, err := ParseSearchPlanJSON(raw)
	if err != nil {
		t.Fatalf("failed to parse search plan JSON: %v", err)
	}

	if len(plan.SelectedTargetIDs) != 3 {
		t.Errorf("expected 3 targets, got %d", len(plan.SelectedTargetIDs))
	}
	if plan.DefaultTimeRange != "year" {
		t.Errorf("expected default_time_range 'year', got %q", plan.DefaultTimeRange)
	}
	if len(plan.TargetTimeRanges) != 3 {
		t.Fatalf("expected 3 target_time_ranges, got %d", len(plan.TargetTimeRanges))
	}
	if plan.TargetTimeRanges["hn"] != "month" {
		t.Errorf("expected target_time_ranges['hn'] = 'month', got %q", plan.TargetTimeRanges["hn"])
	}
	if plan.TargetTimeRanges["github_issues"] != "week" {
		t.Errorf("expected target_time_ranges['github_issues'] = 'week', got %q", plan.TargetTimeRanges["github_issues"])
	}
	if plan.TargetTimeRanges["indie_blog"] != "" {
		t.Errorf("expected target_time_ranges['indie_blog'] = '', got %q", plan.TargetTimeRanges["indie_blog"])
	}

	// Verify queries
	if len(plan.SearchQueries["en"]) != 1 || plan.SearchQueries["en"][0] != "WebKitGTK Wayland buffer leak" {
		t.Errorf("unexpected en query: %v", plan.SearchQueries["en"])
	}
	if len(plan.SearchQueries["zh"]) != 1 || plan.SearchQueries["zh"][0] != "WebKitGTK 内存泄露" {
		t.Errorf("unexpected zh query: %v", plan.SearchQueries["zh"])
	}
}

func TestParseSearchPlanJSON_BackwardCompatibility(t *testing.T) {
	// Old format without time_range fields
	raw := `{
  "selected_target_ids": ["reddit"],
  "search_queries": {
    "en": ["indie web design"]
  }
}`

	plan, err := ParseSearchPlanJSON(raw)
	if err != nil {
		t.Fatalf("failed to parse backward compatible JSON: %v", err)
	}

	if len(plan.SelectedTargetIDs) != 1 || plan.SelectedTargetIDs[0] != "reddit" {
		t.Errorf("unexpected selected_target_ids: %v", plan.SelectedTargetIDs)
	}
	if plan.DefaultTimeRange != "" {
		t.Errorf("expected empty default_time_range, got %q", plan.DefaultTimeRange)
	}
	if plan.TargetTimeRanges != nil {
		t.Errorf("expected nil target_time_ranges, got %v", plan.TargetTimeRanges)
	}
}

func TestParseSearchPlanJSON_InvalidJSON(t *testing.T) {
	_, err := ParseSearchPlanJSON("invalid json string {[[")
	if err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
}
