package babel

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"Finnegan/core/module"
)

// CompletionFunc is a function signature for invoking LLM text generation
type CompletionFunc func(ctx context.Context, llmCfg module.LLMConfig, modelName string, prompt string) (string, error)

// IsChinese detects whether the text has significant Chinese (Han) characters.
func IsChinese(text string) bool {
	hanCount := 0
	totalLetters := 0

	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			hanCount++
			totalLetters++
		} else if unicode.IsLetter(r) {
			totalLetters++
		}
	}

	if totalLetters == 0 {
		return false
	}

	// If more than 20% of letters are Han characters, or if there are at least 10 Han characters, treat as Chinese.
	return hanCount >= 10 || (float64(hanCount)/float64(totalLetters) >= 0.20)
}

// UniversalTranslator handles language detection and cross-language translation into Chinese.
type UniversalTranslator struct {
	LLMConfig      module.LLMConfig
	ModelName      string
	CompletionFunc CompletionFunc
}

// NewUniversalTranslator creates a new UniversalTranslator.
func NewUniversalTranslator(cfg module.LLMConfig, modelName string, fn CompletionFunc) *UniversalTranslator {
	if modelName == "" {
		modelName = "gemini-3.8-flash"
	}
	return &UniversalTranslator{
		LLMConfig:      cfg,
		ModelName:      modelName,
		CompletionFunc: fn,
	}
}

// TranslateIfNeeded checks the language of originalText.
// If it is already Chinese, it returns "" (no translation needed, avoiding bilingual duplication).
// If it is foreign, it calls the LLM to translate it to natural Simplified Chinese.
func (t *UniversalTranslator) TranslateIfNeeded(ctx context.Context, originalText string) (string, error) {
	trimmed := strings.TrimSpace(originalText)
	if trimmed == "" {
		return "", nil
	}

	// 1. Language check: If already Chinese, keep translated_text empty
	if IsChinese(trimmed) {
		return "", nil
	}

	// 2. If no completion func or no LLM configured, cannot translate
	if t.CompletionFunc == nil || (t.LLMConfig.APIKey == "" && t.LLMConfig.BaseURL == "") {
		return "", nil
	}

	prompt := fmt.Sprintf(`You are an elite multilingual translator.
Translate the following foreign text into fluent, natural, high-signal Simplified Chinese (zh-CN).
Preserve technical accuracy, nuances, and conversational tone.
Do NOT summarize, add preamble, or explain. Return ONLY the translated Chinese text.

TEXT TO TRANSLATE:
%s`, trimmed)

	translated, err := t.CompletionFunc(ctx, t.LLMConfig, t.ModelName, prompt)
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	cleaned := strings.TrimSpace(translated)
	// Strip markdown quotes if LLM wrapped in quotes
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	// If translation ended up identical to original, return empty
	if cleaned == trimmed {
		return "", nil
	}

	return cleaned, nil
}
