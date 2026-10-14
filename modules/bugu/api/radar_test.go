package api

import (
	"Finnegan/modules/bugu/models"
	"testing"
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
