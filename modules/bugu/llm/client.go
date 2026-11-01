package llm

import (
	"context"

	corellm "Finnegan/core/llm"
	"Finnegan/core/module"
)

// CleanJSONText strips any markdown code blocks (```json ... ``` or ``` ... ```) from raw LLM output.
func CleanJSONText(raw string) string {
	return corellm.CleanJSONText(raw)
}

// GenerateCompletion calls either Google Gemini or OpenAI-compatible backend depending on llmCfg.
func GenerateCompletion(ctx context.Context, llmCfg module.LLMConfig, modelName string, prompt string) (string, error) {
	return corellm.GenerateCompletion(ctx, llmCfg, modelName, prompt)
}
