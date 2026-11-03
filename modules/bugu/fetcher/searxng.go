package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Finnegan/modules/bugu/models"
)

type SearxNGClient struct {
	BaseURL string
	Client  *http.Client
}

func NewSearxNGClient(baseURL string) *SearxNGClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &SearxNGClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

type SearchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (s *SearxNGClient) Search(query string, target models.Target, timeRange string) ([]SearchResult, error) {
	var finalQuery string
	var reqURL string

	if strings.EqualFold(strings.TrimSpace(target.Type), "engine") {
		finalQuery = strings.TrimSpace(query)
		reqURL = fmt.Sprintf("%s/search?q=%s&format=json", s.BaseURL, url.QueryEscape(finalQuery))
		reqURL = fmt.Sprintf("%s&engines=%s", reqURL, url.QueryEscape(strings.TrimSpace(target.Value)))
	} else {
		if target.Value != "" {
			finalQuery = fmt.Sprintf("%s %s", query, target.Value)
		} else {
			finalQuery = query
		}
		reqURL = fmt.Sprintf("%s/search?q=%s&format=json", s.BaseURL, url.QueryEscape(finalQuery))
		if len(target.PreferredEngines) > 0 {
			reqURL = fmt.Sprintf("%s&engines=%s", reqURL, url.QueryEscape(strings.Join(target.PreferredEngines, ",")))
		}
	}

	if tr := strings.TrimSpace(timeRange); tr != "" {
		reqURL = fmt.Sprintf("%s&time_range=%s", reqURL, url.QueryEscape(tr))
	}

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
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []SearchResult
	// Limit to top 3 results per target-query combo to prevent flooding
	for i, r := range result.Results {
		if i >= 3 {
			break
		}
		if r.URL == "" {
			continue
		}
		items = append(items, SearchResult{
			URL:     r.URL,
			Title:   r.Title,
			Content: r.Content,
		})
	}

	return items, nil
}
