package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"Finnegan/modules/bugu/models"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// DistillMarkdown takes raw markdown from Jina and the user's intent, and distills it into a BabelCard.
func (r *Router) DistillMarkdown(intent string, markdown string, url string, targetLanguage string) (*models.BabelCard, error) {
	if r.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM API Key is not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(r.LLM.APIKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	modelName := "gemini-3.8-flash"
	if r.LLM.DefaultModel != "" {
		modelName = r.LLM.DefaultModel
	}
	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	prompt := fmt.Sprintf(`You are an elite intelligence analyst. 
Your goal is to extract the most critical, high-signal information from the provided webpage markdown, strictly based on the user's intent.

USER INTENT: %s

SOURCE URL: %s

WEBPAGE MARKDOWN:
%s

INSTRUCTIONS:
1. Locate the 1 to 2 most relevant paragraphs in the markdown that directly address the user's intent. Do not summarize the whole page.
2. Extract the exact original text.
3. Provide a highly accurate, natural %s translation of the extracted text.
4. Output strict JSON matching this exact structure:
{
  "source_url": "the original url",
  "original_text": "the exact original text extracted",
  "translated_text": "the translation in %s"
}`, intent, url, markdown, targetLanguage, targetLanguage)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM distiller")
	}

	var rawJSON string
	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		rawJSON = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected LLM response format")
	}

	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	var card models.BabelCard
	if err := json.Unmarshal([]byte(rawJSON), &card); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nRaw string: %s", err, rawJSON)
	}

	// Safety fallback for URL
	if card.SourceURL == "" {
		card.SourceURL = url
	}

	return &card, nil
}
