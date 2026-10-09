package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"Finnegan/bugu/models"
)

type SearxNGClient struct {
	BaseURL string
	Client  *http.Client
}

func NewSearxNGClient(baseURL string) *SearxNGClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &SearxNGClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SearxNGClient) Search(query string, target models.Target) ([]string, error) {
	// e.g. "LLM vulnerabilities site:linuxfr.org"
	finalQuery := fmt.Sprintf("%s %s", query, target.Value)

	reqURL := fmt.Sprintf("%s/search?q=%s&format=json", s.BaseURL, url.QueryEscape(finalQuery))

	resp, err := s.Client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("searxng returned status %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var urls []string
	// Limit to top 3 results per target-query combo to prevent flooding
	for i, r := range result.Results {
		if i >= 3 {
			break
		}
		urls = append(urls, r.URL)
	}

	return urls, nil
}
