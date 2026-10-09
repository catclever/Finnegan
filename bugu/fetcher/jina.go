package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type JinaClient struct {
	BaseURL string
	Client  *http.Client
}

func NewJinaClient() *JinaClient {
	return &JinaClient{
		BaseURL: "https://r.jina.ai/",
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
	// Adding specific headers if needed by Jina
	req.Header.Set("Accept", "text/event-stream")

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
