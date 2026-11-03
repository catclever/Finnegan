package keywords

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/module"
)

// mockLLMClient simulates Gemini response for testing gating and extraction.
type mockLLMClient struct {
	responseJSON   string
	lastPrompt     string
	lastModel      string
	lastImages     []ImageAttachment
	lastWebContext string
}

func (m *mockLLMClient) GenerateJSON(ctx context.Context, modelName string, prompt string, images []ImageAttachment, webContext string) (string, error) {
	m.lastModel = modelName
	m.lastPrompt = prompt
	m.lastImages = images
	m.lastWebContext = webContext
	return m.responseJSON, nil
}

type mockLinkFetcher struct {
	content string
	err     error
	urls    []string
}

func (m *mockLinkFetcher) Fetch(ctx context.Context, url string) (string, error) {
	m.urls = append(m.urls, url)
	return m.content, m.err
}

func (m *mockLinkFetcher) FetchAll(ctx context.Context, urls []string) string {
	m.urls = append(m.urls, urls...)
	return m.content
}

func TestExtractKeywords_4AnchorsAndSalienceGating(t *testing.T) {
	mockResp := `{
  "object": ["WebKitGTK 4.1", "Tauri v2"],
  "motif": ["排错排障", "编译失败"],
  "mood": [],
  "concept": [],
  "unrelated_dim": ["should be filtered out"]
}`

	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{
		DefaultModel: "gemini-3.8-flash",
		APIKey:       "test-key",
	}

	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	cardText := `Compiling WebKitGTK 4.1 with Tauri v2 fails with undefined symbol: g_once_init_enter_pointer`
	profile, err := engine.ExtractKeywords(context.Background(), cardText)
	if err != nil {
		t.Fatalf("ExtractKeywords failed: %v", err)
	}

	// 1. Check prompt includes active dimensions and anchor principles
	if !strings.Contains(mock.lastPrompt, DimensionObject) || !strings.Contains(mock.lastPrompt, DimensionMotif) {
		t.Errorf("Prompt should include active anchor dimensions: %s", mock.lastPrompt)
	}
	if !strings.Contains(mock.lastPrompt, "CONCRETE, COMPACT ANCHORS") || !strings.Contains(mock.lastPrompt, "实心锚点") {
		t.Errorf("Prompt should require concrete anchors: %s", mock.lastPrompt)
	}
	if !strings.Contains(mock.lastPrompt, "2 to 5 Chinese characters, or 1 to 3 English words") {
		t.Errorf("Prompt should enforce length limit: %s", mock.lastPrompt)
	}
	if !strings.Contains(mock.lastPrompt, "WebKitGTK 4.1") {
		t.Errorf("Prompt should contain card text: %s", mock.lastPrompt)
	}

	// 2. Check triggered dimensions
	if len(profile["object"]) != 2 || profile["object"][0] != "WebKitGTK 4.1" {
		t.Errorf("Expected object keywords, got %v", profile["object"])
	}
	if len(profile["motif"]) != 2 || profile["motif"][0] != "排错排障" {
		t.Errorf("Expected motif keywords, got %v", profile["motif"])
	}

	// 3. Check gated-out empty dimensions
	if len(profile["mood"]) != 0 {
		t.Errorf("Expected mood to be empty, got %v", profile["mood"])
	}
	if len(profile["concept"]) != 0 {
		t.Errorf("Expected concept to be empty, got %v", profile["concept"])
	}

	// 4. Check inactive/unrelated dimensions filtered
	if _, exists := profile["unrelated_dim"]; exists {
		t.Errorf("Unrelated dimension should have been filtered out")
	}
}

func TestExtractKeywords_LegacyDimensionMapping(t *testing.T) {
	mockResp := `{
  "keywords": {
    "entity": ["Claude", "Colab"],
    "event": ["体验新玩具"],
    "emotion": ["通透"],
    "judgment": ["伪提效"]
  }
}`

	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	profile, err := engine.ExtractKeywords(context.Background(), "今天体验了 Claude 和 Colab，感觉之前的所谓提效其实是伪提效，心里豁然通透。")
	if err != nil {
		t.Fatalf("ExtractKeywords failed: %v", err)
	}

	// Legacy entity should map to canonical object
	if len(profile["object"]) != 2 || profile["object"][0] != "Claude" {
		t.Errorf("Expected legacy entity mapped to object, got %v", profile["object"])
	}
	// Legacy event should map to canonical motif
	if len(profile["motif"]) != 1 || profile["motif"][0] != "体验新玩具" {
		t.Errorf("Expected legacy event mapped to motif, got %v", profile["motif"])
	}
	// Legacy emotion should map to canonical mood
	if len(profile["mood"]) != 1 || profile["mood"][0] != "通透" {
		t.Errorf("Expected legacy emotion mapped to mood, got %v", profile["mood"])
	}
	// Legacy judgment should map to canonical concept
	if len(profile["concept"]) != 1 || profile["concept"][0] != "伪提效" {
		t.Errorf("Expected legacy judgment mapped to concept, got %v", profile["concept"])
	}
}

func TestNormalizeDimension_AllLegacyAliases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"entity", DimensionObject},
		{"person_work", DimensionObject},
		{"object", DimensionObject},
		{"event", DimensionMotif},
		{"symptom", DimensionMotif},
		{"motif", DimensionMotif},
		{"emotion", DimensionMood},
		{"metaphor", DimensionMood},
		{"mood", DimensionMood},
		{"judgment", DimensionConcept},
		{"question", DimensionConcept},
		{"methodology", DimensionConcept},
		{"counter_intuitive", DimensionConcept},
		{"concept", DimensionConcept},
	}

	for _, tt := range tests {
		got := NormalizeDimension(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeDimension(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestBuildPrompt_ConcreteAnchorsRules(t *testing.T) {
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg)

	prompt := engine.BuildPrompt("菜市场里的叔叔和外卖员，充满了生活气息。")

	// Must enforce concrete anchors
	if !strings.Contains(prompt, "CONCRETE, COMPACT ANCHORS") || !strings.Contains(prompt, "实心锚点") {
		t.Errorf("Expected prompt to contain concrete anchors header")
	}
	if !strings.Contains(prompt, "2 to 5 Chinese characters, or 1 to 3 English words") {
		t.Errorf("Expected prompt to contain character count bounds")
	}
	if !strings.Contains(prompt, "No term may exceed 8 characters") {
		t.Errorf("Expected prompt to forbid long sentences > 8 chars")
	}
	if !strings.Contains(prompt, "SALIENCE GATING (两步合一门控)") {
		t.Errorf("Expected prompt to include salience gating instructions")
	}

	// Must include 4 anchor dimensions
	if !strings.Contains(prompt, "[object] (名物与通用指涉)") {
		t.Errorf("Expected prompt to include object dimension description")
	}
	if !strings.Contains(prompt, "[motif] (处境与母题)") {
		t.Errorf("Expected prompt to include motif dimension description")
	}
	if !strings.Contains(prompt, "[mood] (心境与心绪)") {
		t.Errorf("Expected prompt to include mood dimension description")
	}
	if !strings.Contains(prompt, "[concept] (观念与议题)") {
		t.Errorf("Expected prompt to include concept dimension description")
	}
}

func TestProcessCard_SuccessAndEventPublishing(t *testing.T) {
	tempDir := t.TempDir()
	cardFile := filepath.Join(tempDir, "20261020-1500_test.md")
	docID := "01J9TESTDOCID12345"

	initialContent := `---
id: 01J9TESTDOCID12345
type: thought
created: 2026-10-20T15:00:00+08:00
updated: 2026-10-20T15:00:00+08:00
source: local
tags: []
publish: false
---

# Local Sync Architecture

Exploring CRDT and local-first architecture trade-offs.
`
	if err := os.WriteFile(cardFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial card: %v", err)
	}

	mockResp := `{
  "object": ["CRDT", "Local-First"],
  "motif": ["架构选型"],
  "mood": ["踏实"],
  "concept": ["本地优先"]
}`
	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	// Subscribe to EventKeywordsExtracted on EventBus
	extractedCh := bus.GetBus().Subscribe(bus.EventKeywordsExtracted, "test-subscriber")
	defer bus.GetBus().Unsubscribe(bus.EventKeywordsExtracted, "test-subscriber")

	// Run ProcessCardSync
	err := engine.ProcessCardSync(context.Background(), cardFile, docID, "Exploring CRDT and local-first architecture trade-offs.")
	if err != nil {
		t.Fatalf("ProcessCardSync failed: %v", err)
	}

	// 1. Verify EventKeywordsExtracted event received
	select {
	case event := <-extractedCh:
		data, ok := event.Data.(bus.EventKeywordsExtractedData)
		if !ok {
			t.Fatalf("Event data is not EventKeywordsExtractedData: %T", event.Data)
		}
		if data.DocID != docID {
			t.Errorf("Expected DocID %s, got %s", docID, data.DocID)
		}
		if data.FilePath != cardFile {
			t.Errorf("Expected FilePath %s, got %s", cardFile, data.FilePath)
		}
		if len(data.Keywords["object"]) != 2 {
			t.Errorf("Expected 2 object keywords in event, got %v", data.Keywords["object"])
		}
		if len(data.Keywords["concept"]) != 1 {
			t.Errorf("Expected 1 concept keyword in event, got %v", data.Keywords["concept"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for EventKeywordsExtracted event")
	}

	// 2. Verify file content on disk has updated frontmatter
	updatedBytes, err := os.ReadFile(cardFile)
	if err != nil {
		t.Fatalf("Failed to read updated card file: %v", err)
	}
	updatedStr := string(updatedBytes)

	if !strings.Contains(updatedStr, "keywords:") {
		t.Errorf("Updated file missing keywords: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "CRDT") || !strings.Contains(updatedStr, "Local-First") {
		t.Errorf("Updated file missing object keywords: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "本地优先") {
		t.Errorf("Updated file missing concept keyword: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "Exploring CRDT and local-first architecture trade-offs.") {
		t.Errorf("Original body lost in updated file: %s", updatedStr)
	}
}

func TestExtractKeywordsWithContext_MultimodalAndLinkSummaries(t *testing.T) {
	mockResp := `{
  "keywords": {
    "object": ["Gemini", "存算分离"],
    "motif": ["吐槽模型"],
    "mood": ["戏谑"],
    "concept": ["智能代理"]
  },
  "visual_summary": "画面中展示了两只可爱的猫猫表情包，配文戏谑幽默。",
  "link_summary": "Dalembert GitHub 仓库展示了 super-agent 的存算分离实现方案。"
}`

	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	images := []ImageAttachment{
		{MIME: "image/png", Data: []byte("fake-image-bytes-1")},
		{MIME: "image/jpeg", Data: []byte("fake-image-bytes-2")},
	}
	webContext := "[URL: https://github.com/dalembert/super-agent]\nSuper agent compute-storage decoupling repo."

	res, err := engine.ExtractKeywordsWithContext(context.Background(), "gemini是懂梗的~", images, webContext)
	if err != nil {
		t.Fatalf("ExtractKeywordsWithContext failed: %v", err)
	}

	// 1. Verify LLM received images and webContext
	if len(mock.lastImages) != 2 {
		t.Errorf("Expected 2 images passed to LLMClient, got %d", len(mock.lastImages))
	}
	if mock.lastWebContext != webContext {
		t.Errorf("Expected webContext passed to LLMClient, got %q", mock.lastWebContext)
	}

	// 2. Verify prompt contains multimodal and web content instructions
	if !strings.Contains(mock.lastPrompt, "MULTIMODAL IMAGE ANALYSIS") {
		t.Errorf("Prompt missing multimodal instructions: %s", mock.lastPrompt)
	}
	if !strings.Contains(mock.lastPrompt, "REFERENCED WEB CONTENT ANALYSIS") {
		t.Errorf("Prompt missing web content instructions: %s", mock.lastPrompt)
	}
	if !strings.Contains(mock.lastPrompt, "REFERENCED WEB CONTENT:") {
		t.Errorf("Prompt missing referenced web content section: %s", mock.lastPrompt)
	}

	// 3. Verify extracted profile and summaries
	if len(res.Keywords["object"]) != 2 || res.Keywords["object"][0] != "Gemini" {
		t.Errorf("Expected object keywords, got %v", res.Keywords["object"])
	}
	if res.VisualSummary != "画面中展示了两只可爱的猫猫表情包，配文戏谑幽默。" {
		t.Errorf("Unexpected visual summary: %s", res.VisualSummary)
	}
	if res.LinkSummary != "Dalembert GitHub 仓库展示了 super-agent 的存算分离实现方案。" {
		t.Errorf("Unexpected link summary: %s", res.LinkSummary)
	}
}

func TestExtractKeywordsWithContext_FallbackFlatFormat(t *testing.T) {
	// Simulate legacy model response with flat top-level keys
	mockResp := `{
  "object": ["Claude", "Colab"],
  "mood": ["通透"],
  "visual_summary": "Legacy flat visual summary",
  "link_summary": "Legacy flat link summary"
}`

	mock := &mockLLMClient{responseJSON: mockResp}
	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg).WithClient(mock)

	res, err := engine.ExtractKeywordsWithContext(context.Background(), "今天体验了新工具", nil, "")
	if err != nil {
		t.Fatalf("ExtractKeywordsWithContext failed on flat format: %v", err)
	}

	if len(res.Keywords["object"]) != 2 {
		t.Errorf("Expected 2 object keywords from flat format, got %v", res.Keywords["object"])
	}
	if res.VisualSummary != "Legacy flat visual summary" {
		t.Errorf("Expected visual summary from flat format, got %s", res.VisualSummary)
	}
	if res.LinkSummary != "Legacy flat link summary" {
		t.Errorf("Expected link summary from flat format, got %s", res.LinkSummary)
	}
}

func TestProcessCardSync_MultimodalAndLinkContextIntegration(t *testing.T) {
	tempHome := t.TempDir()
	assetsDir := filepath.Join(tempHome, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("Failed to create temp assets dir: %v", err)
	}

	// Create test image in assets
	imgName := "test-image.png"
	imgBytes := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00}
	if err := os.WriteFile(filepath.Join(assetsDir, imgName), imgBytes, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Create card with image and URL
	cardFile := filepath.Join(tempHome, "20261102-test-multimodal.epiphany.md")
	docID := "01M6TESTCARD0000000001"
	cardContent := `---
id: 01M6TESTCARD0000000001
type: thought
created: 2026-11-02T14:33:00+08:00
updated: 2026-11-02T14:33:00+08:00
source: local
images:
  - test-image.png
---

来看看 Dalembert 仓库 https://github.com/dalembert/super-agent，很有启发！
`
	if err := os.WriteFile(cardFile, []byte(cardContent), 0644); err != nil {
		t.Fatalf("Failed to write test card: %v", err)
	}

	mockResp := `{
  "keywords": {
    "object": ["Dalembert", "super-agent"],
    "motif": ["架构探索"],
    "mood": ["启发感"],
    "concept": ["存算分离"]
  },
  "visual_summary": "测试生成的视觉总结：图片展示了系统架构图与测试截图。",
  "link_summary": "测试生成的链接总结：Dalembert 探索了去中心化存算分离架构。"
}`
	mockLLM := &mockLLMClient{responseJSON: mockResp}
	mockFetcher := &mockLinkFetcher{
		content: "# Dalembert Super Agent\nDecentralized compute and storage architecture.",
	}

	cfg := DefaultConfig()
	llmCfg := module.LLMConfig{DefaultModel: "gemini-3.8-flash", APIKey: "test-key"}
	engine := NewKeywordEngine(llmCfg, cfg).
		WithClient(mockLLM).
		WithLinkFetcher(mockFetcher).
		WithHomeDir(tempHome)

	// Run ProcessCardSync
	err := engine.ProcessCardSync(context.Background(), cardFile, docID, "")
	if err != nil {
		t.Fatalf("ProcessCardSync failed: %v", err)
	}

	// 1. Verify image attachment was loaded and passed to LLM
	if len(mockLLM.lastImages) != 1 {
		t.Fatalf("Expected 1 image passed to LLM, got %d", len(mockLLM.lastImages))
	}
	if mockLLM.lastImages[0].MIME != "image/png" {
		t.Errorf("Expected image/png MIME, got %s", mockLLM.lastImages[0].MIME)
	}

	// 2. Verify link fetcher sniffed the URL
	if len(mockFetcher.urls) == 0 || mockFetcher.urls[0] != "https://github.com/dalembert/super-agent" {
		t.Errorf("Expected sniffed URL https://github.com/dalembert/super-agent, got %v", mockFetcher.urls)
	}

	// 3. Verify updated file frontmatter contains keywords, visual_summary, and link_summary
	updatedData, err := os.ReadFile(cardFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}
	updatedStr := string(updatedData)

	if !strings.Contains(updatedStr, "keywords:") || !strings.Contains(updatedStr, "Dalembert") {
		t.Errorf("Updated file missing keywords: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "visual_summary:") || !strings.Contains(updatedStr, "测试生成的视觉总结") {
		t.Errorf("Updated file missing visual_summary: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "link_summary:") || !strings.Contains(updatedStr, "测试生成的链接总结") {
		t.Errorf("Updated file missing link_summary: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "来看看 Dalembert 仓库") {
		t.Errorf("Card body lost in updated file: %s", updatedStr)
	}
}
