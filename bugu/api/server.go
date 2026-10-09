package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"Finnegan/bugu/config"
	"Finnegan/bugu/fetcher"
	"Finnegan/bugu/llm"
	"Finnegan/bugu/models"
)

type RadarRequest struct {
	Intent string `json:"intent"`
}

type RadarResponse struct {
	Status  string   `json:"status"`
	Results []string `json:"results"`
	Error   string   `json:"error,omitempty"`
}

// ExecuteRadar executes the full Bugu deep radar search workflow
func ExecuteRadar(cfg *config.Config, intent string) RadarResponse {
	router := llm.NewRouter(cfg)

	searxURL := os.Getenv("SEARXNG_URL")
	if searxURL == "" {
		searxURL = "http://localhost:8080"
	}
	searxClient := fetcher.NewSearxNGClient(searxURL)
	jinaClient := fetcher.NewJinaClient()

	log.Printf("[Bugu Radar] Processing intent: %s\n", intent)

	// 1. Ask LLM for the plan
	plan, err := router.GenerateSearchPlan(intent)
	if err != nil {
		log.Printf("[ERROR] LLM routing failed: %v", err)
		return RadarResponse{Status: "error", Error: err.Error()}
	}

	log.Printf("[LLM Plan] Selected Targets: %v", plan.SelectedTargetIDs)

	// 2. Fetch URLs concurrently via SearxNG
	var wgSearx sync.WaitGroup
	urlChan := make(chan string, 100)

	for _, targetID := range plan.SelectedTargetIDs {
		target, exists := cfg.TargetMap[targetID]
		if !exists {
			continue
		}

		// Decide language fallback
		lang := "en"
		if len(target.Languages) > 0 {
			lang = target.Languages[0]
		}
		queries, ok := plan.SearchQueries[lang]
		if !ok {
			// Fallback to "en" or first available queries
			for _, v := range plan.SearchQueries {
				queries = v
				break
			}
		}

		for _, query := range queries {
			wgSearx.Add(1)
			go func(q string, t models.Target) {
				defer wgSearx.Done()
				urls, err := searxClient.Search(q, t)
				if err != nil {
					log.Printf("[SearxNG Error] %s: %v", t.ID, err)
					return
				}
				for _, u := range urls {
					urlChan <- u
				}
			}(query, target)
		}
	}

	// Close urlChan when all Searx searches complete
	go func() {
		wgSearx.Wait()
		close(urlChan)
	}()

	// 3. Extract Markdown concurrently via Jina Reader
	var wgJina sync.WaitGroup
	resultChan := make(chan string, 100)

	// Create a worker pool of 10 workers for Jina to avoid rate limits
	for i := 0; i < 10; i++ {
		wgJina.Add(1)
		go func() {
			defer wgJina.Done()
			for u := range urlChan {
				log.Printf("[Jina Reader] Extracting -> %s", u)
				md, err := jinaClient.Extract(u)
				if err != nil {
					log.Printf("[Jina Error] %s: %v", u, err)
					continue
				}
				// Only keep substantial markdown (ignore empty/error pages)
				if len(md) > 100 {
					resultChan <- fmt.Sprintf("Source: %s\n\n%s", u, md)
				}
			}
		}()
	}

	// Close resultChan when all Jina extractions complete
	go func() {
		wgJina.Wait()
		close(resultChan)
	}()

	// 4. Collect results
	var finalResults []string
	for md := range resultChan {
		finalResults = append(finalResults, md)
	}

	log.Printf("[Bugu Radar] Completed! Scraped %d articles.\n", len(finalResults))

	return RadarResponse{
		Status:  "success",
		Results: finalResults,
	}
}

func StartServer(cfg *config.Config, port string) {
	http.HandleFunc("/api/radar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			http.Error(w, `{"status":"error", "error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req RadarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"status":"error", "error":"Bad Request"}`, http.StatusBadRequest)
			return
		}

		res := ExecuteRadar(cfg, req.Intent)
		json.NewEncoder(w).Encode(res)
	})

	log.Printf("🚀 Bugu Drone Daemon is listening on :%s ...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
