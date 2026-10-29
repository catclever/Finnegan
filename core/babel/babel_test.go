package babel

import (
	"context"
	"strings"
	"testing"

	"Finnegan/core/module"
)

func TestUniversalSpamFilter(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		shouldSpam bool
	}{
		{"empty", "", true},
		{"whitespace", "   \n\t  ", true},
		{"too short", "hi", true},
		{"pure url", "https://example.com/article/12345", true},
		{"airdrop spam", "Claim your free airdrop now by dropping your wallet!", true},
		{"broken utf8", string([]byte{0xff, 0xfe, 0xfd}), true},
		{"mojibake replacement", "Hello \uFFFD world \uFFFD broken text", true},
		{"legitimate thought", "数字游牧的核心困境在于时区的割裂与真实物理锚点的丧失。", false},
		{"legitimate english", "Local-first software enables users to own their data across time and space.", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isSpam, reason := UniversalSpamFilter(tc.text)
			if isSpam != tc.shouldSpam {
				t.Errorf("expected isSpam=%v, got %v (reason: %s)", tc.shouldSpam, isSpam, reason)
			}
		})
	}
}

func TestUniversalTranslator_IsChinese(t *testing.T) {
	if !IsChinese("这是一段纯正的中文思考。") {
		t.Errorf("expected Chinese text to be detected as Chinese")
	}
	if IsChinese("This is purely English text with zero Han characters.") {
		t.Errorf("expected English text to not be detected as Chinese")
	}
	if !IsChinese("Finnegan 是一个基于 Wails 的本地优先意识流沙盒。") {
		t.Errorf("expected mixed text with sufficient Chinese to be recognized as Chinese")
	}
}

func TestUniversalTranslator_TranslateIfNeeded(t *testing.T) {
	mockCalled := false
	mockFn := func(ctx context.Context, llmCfg module.LLMConfig, modelName string, prompt string) (string, error) {
		mockCalled = true
		return "本地优先软件改变了数据所有权模式。", nil
	}

	tr := NewUniversalTranslator(module.LLMConfig{APIKey: "mock-key"}, "mock-model", mockFn)

	// Chinese text should NOT trigger translation
	resChinese, err := tr.TranslateIfNeeded(context.Background(), "中文内容不需要重复翻译")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resChinese != "" {
		t.Errorf("expected empty translation for Chinese text, got %q", resChinese)
	}
	if mockCalled {
		t.Errorf("expected mockFn NOT to be called for Chinese text")
	}

	// English text should trigger translation
	resEng, err := tr.TranslateIfNeeded(context.Background(), "Local-first software changes data ownership.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mockCalled {
		t.Errorf("expected mockFn to be called for English text")
	}
	if !strings.Contains(resEng, "本地优先") {
		t.Errorf("expected translated Chinese, got %q", resEng)
	}
}

func TestBuguGatekeeper(t *testing.T) {
	gk := NewBuguGatekeeper(nil)

	// Test 1: Discard card without matched anchor
	card1, err := gk.ValidateAndCurate(context.Background(), RawBabelInput{
		OriginalText:  "High quality paragraph without any anchor resonance.",
		MatchedAnchor: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card1 != nil {
		t.Errorf("expected BuguGatekeeper to discard unresonated card, got %+v", card1)
	}

	// Test 2: Accept card with valid anchor
	card2, err := gk.ValidateAndCurate(context.Background(), RawBabelInput{
		OriginalText:  "Deep discussion on asynchronous consciousness streams.",
		MatchedAnchor: "意识流",
		EchoReason:    "母题契合",
		URL:           "https://example.com/essay",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card2 == nil {
		t.Fatalf("expected card to pass BuguGatekeeper")
	}
	if card2.MatchedAnchor != "意识流" || card2.Origin != "bugu" {
		t.Errorf("unexpected card values: %+v", card2)
	}
}

func TestBlueskyGatekeeper(t *testing.T) {
	gk := NewBlueskyGatekeeper(nil)

	// Test 1: Reject spam bot account
	_, errSpam := gk.ValidateAndCurate(context.Background(), RawBabelInput{
		AuthorName:   "airdrop_promo_bot.bsky.social",
		OriginalText: "Check out this incredible post about software design!",
	})
	if errSpam == nil {
		t.Errorf("expected spam account to be rejected")
	}

	// Test 2: Accept valid post with hashtag extraction
	card, err := gk.ValidateAndCurate(context.Background(), RawBabelInput{
		AuthorName:     "alice.bsky.social",
		OriginalText:   "Loving the tactile feel of reactive e-ink notebooks. #indieweb",
		SearchKeywords: []string{"indieweb"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatalf("expected valid bluesky post to pass")
	}
	if card.MatchedAnchor != "#indieweb" && card.MatchedAnchor != "indieweb" {
		t.Errorf("expected anchor to be extracted, got %q", card.MatchedAnchor)
	}
	if card.Origin != "atproto" {
		t.Errorf("expected origin to be atproto, got %s", card.Origin)
	}
}
