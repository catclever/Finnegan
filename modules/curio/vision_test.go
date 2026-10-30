package curio

import (
	"context"
	"testing"
	"Finnegan/core/module"
)

func TestSanitizeSlug(t *testing.T) {
	cases := []struct{
		in string
		out string
	}{
		{"Hello World", "hello-world"},
		{"Apple/Banana", "apple-banana"},
		{"This is a very long title that should be truncated by the function", "this-is-a-very-long-title-that"},
	}

	for _, c := range cases {
		if res := sanitizeSlug(c.in); res != c.out {
			t.Errorf("sanitizeSlug(%q) = %q, expected %q", c.in, res, c.out)
		}
	}
}

func TestIngestScreenshotMissingAPIKey(t *testing.T) {
	ctx := context.Background()
	llmCfg := module.LLMConfig{} // Empty API Key

	err := IngestScreenshot(ctx, llmCfg, "dummy.jpg", "", nil)
	if err == nil {
		t.Errorf("Expected error for missing file or api key, got nil")
	}
}
