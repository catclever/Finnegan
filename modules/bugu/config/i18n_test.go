package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranslateTag(t *testing.T) {
	tests := []struct {
		tag      string
		lang     string
		expected string
	}{
		{"技术", "en", "technology"},
		{"技术", "ru", "технологии"},
		{"技术", "de", "Technologie"},
		{"技术", "fr", "technologie"},
		{"技术", "ja", "技術"},
		{"技术", "zh-CN", "技术"},
		{"黑客", "ru", "хакеры"},
		{"黑客", "en", "hacking"},
		{"新闻", "de", "Nachrichten"},
		{"编程", "fr", "programmation"},
		{"网络安全", "ru", "кибербезопасность"},
	}

	for _, tc := range tests {
		actual := TranslateTag(tc.tag, tc.lang)
		if actual != tc.expected {
			t.Errorf("TranslateTag(%s, %s) = %s, expected %s", tc.tag, tc.lang, actual, tc.expected)
		}
	}
}

func TestTranslateTags(t *testing.T) {
	tags := []string{"技术", "黑客", "开源"}
	ruTags := TranslateTags(tags, "ru")
	if len(ruTags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(ruTags))
	}
	if ruTags[0] != "технологии" || ruTags[1] != "хакеры" || ruTags[2] != "открытый исходный код" {
		t.Errorf("unexpected translated tags: %v", ruTags)
	}

	zhTags := TranslateTags(tags, "zh-CN")
	if zhTags[0] != "技术" || zhTags[1] != "黑客" {
		t.Errorf("unexpected chinese tags: %v", zhTags)
	}
}

func TestCustomTranslationsAndPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "tags_i18n.json")

	// 1. Initially "赛博朋克" has no translation
	if HasTagTranslation("赛博朋克") {
		t.Errorf("expected 赛博朋克 not to be translated initially")
	}

	// 2. Merge new translation
	newTranslations := map[string]map[string]string{
		"赛博朋克": {
			"en": "cyberpunk",
			"ru": "киберпанк",
			"ja": "サイバーパンク",
		},
	}
	if err := MergeCustomTranslations(newTranslations, jsonPath); err != nil {
		t.Fatalf("MergeCustomTranslations failed: %v", err)
	}

	// 3. Now HasTagTranslation should be true
	if !HasTagTranslation("赛博朋克") {
		t.Errorf("expected 赛博朋克 to have translation after merge")
	}

	// 4. Verify translation lookup
	if tr := TranslateTag("赛博朋克", "ru"); tr != "киберпанк" {
		t.Errorf("expected киберпанк, got %s", tr)
	}
	if tr := TranslateTag("赛博朋克", "ja"); tr != "サイバーパンク" {
		t.Errorf("expected サイバーパンク, got %s", tr)
	}
	if tr := TranslateTag("赛博朋克", "de"); tr != "cyberpunk" { // fallback to en
		t.Errorf("expected fallback to cyberpunk, got %s", tr)
	}

	// 5. Verify JSON file was written
	if _, err := os.Stat(jsonPath); err != nil {
		t.Errorf("expected %s to exist, err: %v", jsonPath, err)
	}
}
