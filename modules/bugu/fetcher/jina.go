package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type JinaClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func NewJinaClient(apiKey string) *JinaClient {
	return &JinaClient{
		BaseURL: "https://r.jina.ai/",
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
	
	// Adding specific headers for Jina API
	req.Header.Set("Accept", "text/event-stream")
	
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}
