package api

import (
	"fmt"
	"log"
	"sync"

	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/fetcher"
	"Finnegan/modules/bugu/llm"
	"Finnegan/modules/bugu/models"
	"Finnegan/core/bus"
)

type RadarResponse struct {
	Status  string   `json:"status"`
	Results []string `json:"results"`
	Error   string   `json:"error,omitempty"`
}

func ExecuteRadar(cfg *config.Config, router *llm.Router, searxURL, jinaKey string, maxConcurrency int, intent string, targetLanguage string, docID string) RadarResponse {
	log.Printf("[Bugu Module] Processing intent: %s (docID: %s)\n", intent, docID)

	searxClient := fetcher.NewSearxNGClient(searxURL)
	jinaClient := fetcher.NewJinaClient(jinaKey)

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

		lang := "en"
		if len(target.Languages) > 0 {
			lang = target.Languages[0]
		}
		queries, ok := plan.SearchQueries[lang]
		if !ok {
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

	go func() {
		wgSearx.Wait()
		close(urlChan)
	}()

	// 3. Extract Markdown & Distill concurrently
	var wgJina sync.WaitGroup
	resultChan := make(chan string, 100)

	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}

	eventBus := bus.GetBus()

	for i := 0; i < maxConcurrency; i++ {
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
				if len(md) > 100 {
					// Distill the markdown into a BabelCard
					log.Printf("[LLM Distiller] Processing -> %s", u)
					card, err := router.DistillMarkdown(intent, md, u, targetLanguage)
					if err != nil {
						log.Printf("[Distiller Error] %s: %v", u, err)
						continue
					}
					card.EchoOf = docID
					
					// Broadcast the BabelCard to the frontend via Event Bus
					eventBus.Publish(bus.Event{
						Type: bus.EventBabelCardSpawned,
						Data: *card,
					})
					
					// Also append to standard results for legacy/sync support
					resultChan <- fmt.Sprintf("Source: %s\n\n%s", u, card.TranslatedText)
				}
			}
		}()
	}

	go func() {
		wgJina.Wait()
		close(resultChan)
	}()

	// 4. Collect results (Sync return)
	var finalResults []string
	for md := range resultChan {
		finalResults = append(finalResults, md)
	}

	log.Printf("[Bugu Module] Completed! Scraped and distilled %d articles.\n", len(finalResults))

	return RadarResponse{
		Status:  "success",
		Results: finalResults,
	}
}
