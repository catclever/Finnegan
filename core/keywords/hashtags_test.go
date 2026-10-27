package keywords

import (
	"reflect"
	"testing"
)

func TestStripAndExtractHashtags(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedBody  string
		expectedTags  []string
	}{
		{
			name:         "Inline hashtags preserving grammatical fluency",
			input:        "今天测试了 #WebKitGTK 4.1 和 #跨平台UI, 以及 #v1.0.0。顺便体验了 #松弛感！",
			expectedBody: "今天测试了 WebKitGTK 4.1 和 跨平台UI, 以及 v1.0.0。顺便体验了 松弛感！",
			expectedTags: []string{"WebKitGTK", "跨平台UI", "v1.0.0", "松弛感"},
		},
		{
			name:         "Standalone lines of hashtags stripped completely",
			input:        "第一段想法记录。\n\n#WebKitGTK #跨平台UI #松弛感",
			expectedBody: "第一段想法记录。",
			expectedTags: []string{"WebKitGTK", "跨平台UI", "松弛感"},
		},
		{
			name:         "Trailing hashtags after sentence-ending punctuation stripped completely",
			input:        "今天完成了一个大特性。 #release #v2.0",
			expectedBody: "今天完成了一个大特性。",
			expectedTags: []string{"release", "v2.0"},
		},
		{
			name:         "Mixed inline and standalone hashtag lines",
			input:        "#Finnegan\n今天体验了 #松弛感 的设计。\n#生活 #感悟",
			expectedBody: "今天体验了 松弛感 的设计。",
			expectedTags: []string{"Finnegan", "松弛感", "生活", "感悟"},
		},
		{
			name:         "Single inline hashtag at end without punctuation",
			input:        "今天体验了 #松弛感",
			expectedBody: "今天体验了 松弛感",
			expectedTags: []string{"松弛感"},
		},
		{
			name:         "Multiple trailing hashtags without punctuation",
			input:        "今天体验了微风拂面 #松弛感 #治愈",
			expectedBody: "今天体验了微风拂面",
			expectedTags: []string{"松弛感", "治愈"},
		},
		{
			name:         "Pure hashtags input fallback",
			input:        "#松弛感 #治愈",
			expectedBody: "松弛感 治愈",
			expectedTags: []string{"松弛感", "治愈"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, tags := StripAndExtractHashtags(tc.input)
			if body != tc.expectedBody {
				t.Errorf("expected body:\n%q\ngot:\n%q", tc.expectedBody, body)
			}
			if !reflect.DeepEqual(tags, tc.expectedTags) {
				t.Errorf("expected tags %v, got %v", tc.expectedTags, tags)
			}
		})
	}
}

func TestRouteTagToDimension(t *testing.T) {
	tests := []struct {
		tag      string
		expected string
	}{
		{"松弛感", DimensionMood},
		{"抽离感", DimensionMood},
		{"吐槽欲", DimensionMood},
		{"安全感", DimensionMood}, // heuristic ends with 感
		{"探索欲", DimensionMood}, // heuristic ends with 欲
		{"逃席", DimensionMotif},
		{"体验新玩具", DimensionMotif},
		{"吐槽模型", DimensionMotif},
		{"伪提效", DimensionConcept},
		{"存量陷阱", DimensionConcept},
		{"消费主义", DimensionConcept}, // heuristic ends with 主义
		{"WebKitGTK 4.1", DimensionObject},
		{"WebKitGTK", DimensionObject},
		{"跨平台UI", DimensionObject},
		{"v1.0.0", DimensionObject},
		{"Golang", DimensionObject},
	}

	for _, tc := range tests {
		got := RouteTagToDimension(tc.tag)
		if got != tc.expected {
			t.Errorf("RouteTagToDimension(%q) = %q, expected %q", tc.tag, got, tc.expected)
		}
	}
}

func TestRouteTagsToKeywords(t *testing.T) {
	tags := []string{"WebKitGTK 4.1", "松弛感", "逃席", "伪提效", "跨平台UI"}
	kw := RouteTagsToKeywords(tags)

	if len(kw[DimensionObject]) != 2 || kw[DimensionObject][0] != "WebKitGTK 4.1*" || kw[DimensionObject][1] != "跨平台UI*" {
		t.Errorf("unexpected object keywords: %v", kw[DimensionObject])
	}
	if len(kw[DimensionMood]) != 1 || kw[DimensionMood][0] != "松弛感*" {
		t.Errorf("unexpected mood keywords: %v", kw[DimensionMood])
	}
	if len(kw[DimensionMotif]) != 1 || kw[DimensionMotif][0] != "逃席*" {
		t.Errorf("unexpected motif keywords: %v", kw[DimensionMotif])
	}
	if len(kw[DimensionConcept]) != 1 || kw[DimensionConcept][0] != "伪提效*" {
		t.Errorf("unexpected concept keywords: %v", kw[DimensionConcept])
	}
}
