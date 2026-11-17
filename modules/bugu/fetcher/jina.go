package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	coreConfig "Finnegan/core/config"
)

type JinaClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func NewJinaClient(apiKey string, baseURL ...string) *JinaClient {
	bURL := ""
	if len(baseURL) > 0 {
		bURL = strings.TrimSpace(baseURL[0])
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" || bURL == "" {
		cfg, _, err := coreConfig.LoadConfig()
		if err == nil && cfg != nil {
			if apiKey == "" && cfg.Jina.APIKey != "" {
				apiKey = cfg.Jina.APIKey
			}
			if bURL == "" && cfg.Jina.BaseURL != "" {
				bURL = cfg.Jina.BaseURL
			}
		}
	}

	if apiKey == "" {
		apiKey = os.Getenv("JINA_API_KEY")
	}
	if bURL == "" {
		bURL = "https://r.jina.ai/"
	}

	return &JinaClient{
		BaseURL: bURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second, // Jina can be slow to render JS
		},
	}
}

func (j *JinaClient) Extract(targetURL string) (string, error) {
	reqURL := j.BaseURL + targetURL

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	
	// Request plain markdown directly rather than SSE stream so each article returns immediately
	req.Header.Set("Accept", "text/plain")
	
	if j.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+j.APIKey)
	}

	resp, err := j.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("jina returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", err
	}

	result := string(bodyBytes)
	runes := []rune(result)
	if len(runes) > 15000 {
		return string(runes[:15000]), nil
	}
	return result, nil
}
