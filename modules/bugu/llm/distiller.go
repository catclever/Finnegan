package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"Finnegan/modules/bugu/models"
)

// DistillMarkdown takes raw markdown from Jina and the user's intent, and distills it into a BabelCard.
func (r *Router) DistillMarkdown(intent string, markdown string, url string, targetLanguage string, keywords map[string][]string) (*models.BabelCard, error) {
	dLLM := r.GetDistillerLLM()
	if dLLM.APIKey == "" && dLLM.BaseURL == "" {
		return nil, fmt.Errorf("Distiller LLM is not configured (missing api_key or base_url)")
	}

	modelName := "gemini-3.8-flash"
	if r.Config != nil && r.Config.GetDistillerModel() != "" {
		modelName = r.Config.GetDistillerModel()
	} else if dLLM.DefaultModel != "" {
		modelName = dLLM.DefaultModel
	}

	kwJSON, _ := json.MarshalIndent(keywords, "", "  ")

	prompt := fmt.Sprintf(`You are an elite intelligence analyst. 
Your goal is to extract the most critical, high-signal information from the provided webpage markdown, strictly based on the user's intent.

USER INTENT: %s

SOURCE URL: %s

CARD ANCHORS (Keywords):
%s

WEBPAGE MARKDOWN:
%s

INSTRUCTIONS:
1. Locate the 1 to 2 most relevant paragraphs in the markdown that directly address the user's intent. Do not summarize the whole page.
2. Evaluate if the extracted text genuinely responds to or resonates with at least one solid anchor (object, motif, mood, concept) from the CARD ANCHORS provided.
   - If it is SEO spam, a 403 error page, or fails to hit any anchor meaningfully, set "matched_anchor" to null or an empty string.
   - Otherwise, set "matched_anchor" to the specific anchor name it hits (e.g. "伪提效", "额度焦虑"), and provide a brief one-sentence explanation in "echo_reason".
3. Extract the exact original text into "original_text".
4. Check the language of the extracted original text:
   - If it is in a foreign language (different from %s), provide a highly accurate, natural %s translation in "translated_text".
   - If it is ALREADY in %s, leave "translated_text" as an empty string (""). Absolutely do NOT translate, rephrase, or duplicate text that is already in %s.
5. Extract publication and author metadata from the webpage markdown:
   - "published_at": The original publication date or release time of the article/post (e.g. "2024-05-12", "March 15, 2024"). If not explicitly mentioned in the markdown, leave as an empty string ("").
   - "author_name": The author, creator, or publisher name (e.g. "John Doe", "@username"). If not found, leave as an empty string ("").
   - "author_avatar": The URL of the author's avatar or profile image if present in markdown image tags or metadata. If not found, leave as an empty string ("").
6. Output strict JSON matching this exact structure:
{
  "source_url": "%s",
  "original_text": "the exact original text extracted",
  "translated_text": "translation or empty string",
  "published_at": "publication date or empty string",
  "author_name": "author name or empty string",
  "author_avatar": "author avatar URL or empty string",
  "matched_anchor": "hit anchor name or empty string",
  "echo_reason": "one sentence explanation or empty string"
}`, intent, url, string(kwJSON), markdown, targetLanguage, targetLanguage, targetLanguage, targetLanguage, url)

	ctx := context.Background()
	rawJSON, err := GenerateCompletion(ctx, dLLM, modelName, prompt)
	if err != nil {
		return nil, fmt.Errorf("distiller generation failed: %w", err)
	}

	var card models.BabelCard
	if err := json.Unmarshal([]byte(rawJSON), &card); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nRaw string: %s", err, rawJSON)
	}

	// Safety fallback for URL
	if card.SourceURL == "" {
		card.SourceURL = url
	}

	// Gating: If matched_anchor is empty or original_text is empty, return nil, nil
	if strings.TrimSpace(card.MatchedAnchor) == "" || strings.TrimSpace(card.OriginalText) == "" {
		return nil, nil // Indicate discarded
	}

	// If LLM echoed identical text or whitespace, clear translated_text
	if strings.TrimSpace(card.TranslatedText) == strings.TrimSpace(card.OriginalText) {
		card.TranslatedText = ""
	}

	card.CreatedAt = time.Now()

	return &card, nil
}
