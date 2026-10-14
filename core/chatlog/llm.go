package chatlog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"Finnegan/core/module"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// TitleResult holds the title and participants derived from a dialogue.
type TitleResult struct {
	Title        string   `json:"title"`
	Participants []string `json:"participants"`
}

// DeriveFallbackTitle generates a deterministic, meaningful title when LLM is unavailable.
func DeriveFallbackTitle(dialogue string, participants []string) string {
	var firstMsg string
	lines := strings.Split(dialogue, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasSuffix(trimmed, ":") {
			continue
		}
		firstMsg = trimmed
		break
	}

	prefix := ""
	if len(participants) == 1 {
		prefix = participants[0] + ": "
	} else if len(participants) >= 2 {
		prefix = participants[0] + "与" + participants[1] + ": "
	}

	if firstMsg == "" {
		if prefix != "" {
			return strings.TrimSuffix(prefix, ": ") + "的对话"
		}
		return "对话记录"
	}

	candidate := prefix + firstMsg
	runes := []rune(candidate)
	if len(runes) > 24 {
		return string(runes[:24]) + "..."
	}
	return candidate
}

// DeriveTitle derives a natural title and participant list using LLM (Gemini 3.8 Flash).
// If LLM config is missing or the API call fails, it gracefully falls back to deterministic derivation.
func DeriveTitle(ctx context.Context, llmCfg module.LLMConfig, dialogue string, defaultParticipants []string) (*TitleResult, error) {
	if llmCfg.APIKey == "" {
		return &TitleResult{
			Title:        DeriveFallbackTitle(dialogue, defaultParticipants),
			Participants: defaultParticipants,
		}, nil
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(llmCfg.APIKey))
	if err != nil {
		return &TitleResult{
			Title:        DeriveFallbackTitle(dialogue, defaultParticipants),
			Participants: defaultParticipants,
		}, fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	modelName := "gemini-3.8-flash"
	if llmCfg.DefaultModel != "" {
		modelName = llmCfg.DefaultModel
	}
	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	prompt := fmt.Sprintf(`You are an elite dialogue editor and archivist.
Your goal is to extract a concise, vivid, natural title and list of participants from the provided chat log.

CHAT LOG:
%s

INSTRUCTIONS:
1. "title": A succinct, engaging, natural title in Chinese/English capturing the core topic or spirit of what was discussed (typically 8-25 characters; avoid robotic, passive, or generic titles like "聊天记录" or "关于...的讨论").
2. "participants": An array of participant names speaking in the chat log.

Output strict JSON matching this exact structure:
{
  "title": "...",
  "participants": ["..."]
}`, dialogue)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return &TitleResult{
			Title:        DeriveFallbackTitle(dialogue, defaultParticipants),
			Participants: defaultParticipants,
		}, fmt.Errorf("gemini title generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return &TitleResult{
			Title:        DeriveFallbackTitle(dialogue, defaultParticipants),
			Participants: defaultParticipants,
		}, nil
	}

	var rawJSON string
	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		rawJSON = string(textPart)
	} else {
		return &TitleResult{
			Title:        DeriveFallbackTitle(dialogue, defaultParticipants),
			Participants: defaultParticipants,
		}, nil
	}

	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")
	rawJSON = strings.TrimSpace(rawJSON)

	var res TitleResult
	if err := json.Unmarshal([]byte(rawJSON), &res); err != nil {
		return &TitleResult{
			Title:        DeriveFallbackTitle(dialogue, defaultParticipants),
			Participants: defaultParticipants,
		}, nil
	}

	res.Title = strings.TrimSpace(res.Title)
	if res.Title == "" {
		res.Title = DeriveFallbackTitle(dialogue, defaultParticipants)
	}
	if len(res.Participants) == 0 {
		res.Participants = defaultParticipants
	}

	// Safety check on title length
	if utf8.RuneCountInString(res.Title) > 40 {
		runes := []rune(res.Title)
		res.Title = string(runes[:40]) + "..."
	}

	return &res, nil
}
