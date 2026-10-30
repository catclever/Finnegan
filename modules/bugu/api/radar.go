package api

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"Finnegan/core/bus"
	coreconfig "Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/watcher"
	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/fetcher"
	"Finnegan/modules/bugu/llm"
	"Finnegan/modules/bugu/models"
)

type RadarResponse struct {
	Status  string   `json:"status"`
	Results []string `json:"results"`
	Error   string   `json:"error,omitempty"`
}

// isBlockedOrLowQualityMarkdown detects blocked/anti-crawler/empty pages (e.g. Reddit 403, Zhihu login walls)
func isBlockedOrLowQualityMarkdown(md string) bool {
	trimmed := strings.TrimSpace(md)
	if len(trimmed) < 80 {
		return true
	}
	lower := strings.ToLower(trimmed)
	blockPhrases := []string{
		"403 forbidden",
		"access denied",
		"just a moment...",
		"verify you are human",
		"blocked by network security",
		"sorry, you have been blocked",
		"please turn javascript on",
		"enable cookies",
		"登录知乎",
		"知乎 - 有问题，就会有答案",
		"安全验证",
		"sec.zhihu.com",
		"reddit.com/r/popular",
	}
	for _, p := range blockPhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// resolveTargetQueries safely resolves search queries for a target.
// It prioritizes exact language match, base language match, and universal English fallback.
// It STRICTLY PREVENTS sending Chinese queries to foreign/minor language websites.
func resolveTargetQueries(target models.Target, searchQueries map[string][]string) []string {
	if len(searchQueries) == 0 {
		return nil
	}

	// 1. Try exact target languages in order (e.g. "zh-CN", "ru", "fr", "de")
	for _, l := range target.Languages {
		l = strings.TrimSpace(l)
		if qs, ok := searchQueries[l]; ok && len(qs) > 0 {
			return qs
		}
		// 2. Try base language prefix (e.g. "zh" for "zh-CN", "en" for "en-US")
		base := strings.ToLower(strings.Split(l, "-")[0])
		if qs, ok := searchQueries[base]; ok && len(qs) > 0 {
			return qs
		}
	}

	// Determine if target is Chinese
	isChineseTarget := false
	for _, l := range target.Languages {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "zh") {
			isChineseTarget = true
			break
		}
	}

	// 3. Fallback for non-Chinese foreign targets: English ("en") is universal!
	// NEVER leak Chinese query into Russian, French, German, or Japanese searches!
	if !isChineseTarget {
		if qs, ok := searchQueries["en"]; ok && len(qs) > 0 {
			return qs
		}
		// If no English query, check other non-Chinese queries
		for k, qs := range searchQueries {
			if !strings.HasPrefix(strings.ToLower(k), "zh") && len(qs) > 0 {
				return qs
			}
		}
		// Strictly avoid returning Chinese queries for foreign targets
		return nil
	} else {
		// Chinese target: try "zh", then "zh-CN", then "en"
		if qs, ok := searchQueries["zh"]; ok && len(qs) > 0 {
			return qs
		}
		if qs, ok := searchQueries["zh-CN"]; ok && len(qs) > 0 {
			return qs
		}
		if qs, ok := searchQueries["en"]; ok && len(qs) > 0 {
			return qs
		}
		for _, qs := range searchQueries {
			if len(qs) > 0 {
				return qs
			}
		}
	}

	return nil
}

// FindCardFilePath locates the local Markdown file on disk corresponding to a docID.
func FindCardFilePath(docID string) string {
	if docID == "" {
		return ""
	}

	// 1. Direct file path check
	if fi, err := os.Stat(docID); err == nil && !fi.IsDir() {
		return docID
	}

	// 2. GlobalWatcher registry lookup
	if w := watcher.GetWatcher(); w != nil {
		if path := w.GetPathForID(docID); path != "" {
			if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
				return path
			}
		}
	}

	// 3. Search directory locations (configured dataDir, standard Scattering/Weaving paths)
	var searchDirs []string
	if _, dataDir, err := coreconfig.LoadConfig(); err == nil && dataDir != "" {
		searchDirs = append(searchDirs,
			filepath.Join(dataDir, "Scattering"),
			filepath.Join(dataDir, "Weaving"),
		)
	}
	home := os.Getenv("HOME")
	if home != "" {
		searchDirs = append(searchDirs,
			filepath.Join(home, "Documents", "Finnegan", "Scattering"),
			filepath.Join(home, "Documents", "Finnegan", "Weaving"),
		)
	}
	searchDirs = append(searchDirs, "Scattering", "Weaving")

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if strings.Contains(e.Name(), docID) {
				return path
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if strings.Contains(string(data), docID) {
				return path
			}
		}
	}
	return ""
}

// ExtractKeywordsForDoc extracts stored keywords profile from the markdown card corresponding to docID.
func ExtractKeywordsForDoc(docID string) map[string][]string {
	if docID == "" {
		return nil
	}
	path := FindCardFilePath(docID)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	profile, err := keywords.ExtractKeywordsFromMarkdown(data)
	if err != nil || len(profile) == 0 {
		return nil
	}
	return map[string][]string(profile)
}

func ExecuteRadar(cfg *config.Config, router *llm.Router, searxURL, jinaKey string, maxConcurrency int, intent string, targetLanguage string, docID string, keywordsOpt ...map[string][]string) RadarResponse {
	var kw map[string][]string
	if len(keywordsOpt) > 0 && keywordsOpt[0] != nil {
		kw = keywordsOpt[0]
	}
	if len(kw) == 0 && docID != "" {
		path := FindCardFilePath(docID)
		if path != "" {
			if strings.HasSuffix(path, ".note.md") {
				return RadarResponse{
					Status: "error",
					Error:  "Bugu radar excludes note cards",
				}
			}
			if data, err := os.ReadFile(path); err == nil {
				contentStr := string(data)
				if strings.Contains(contentStr, "\ntype: note\n") || strings.Contains(contentStr, "\ntype: \"note\"\n") {
					return RadarResponse{
						Status: "error",
						Error:  "Bugu radar excludes note cards",
					}
				}
			}
		}
		kw = ExtractKeywordsForDoc(docID)
	}

	log.Printf("[Bugu Module] Processing intent: %s (docID: %s, keywords count: %d)\n", intent, docID, len(kw))

	eventBus := bus.GetBus()
	defer func() {
		eventBus.Publish(bus.Event{
			Type: bus.EventBabelSessionCompleted,
			Data: map[string]string{
				"docId": docID,
			},
		})
		eventBus.Publish(bus.Event{
			Type: bus.EventBuguRadarCompleted,
			Data: map[string]string{
				"docId": docID,
			},
		})
	}()

	searxClient := fetcher.NewSearxNGClient(searxURL)
	jinaClient := fetcher.NewJinaClient(jinaKey)
	redditClient := fetcher.NewRedditClient()

	// 1. Ask LLM for the plan using facet routing
	plan, err := router.GenerateFacetSearchPlan(intent, kw)
	if err != nil {
		log.Printf("[ERROR] LLM routing failed: %v", err)
		return RadarResponse{Status: "error", Error: err.Error()}
	}

	log.Printf("[LLM Plan] Selected Targets: %v (default_time_range: %s, target_time_ranges: %v)", plan.SelectedTargetIDs, plan.DefaultTimeRange, plan.TargetTimeRanges)

	// 2. Fetch results concurrently via SearxNG and stream unique items immediately to itemChan
	var wgSearx sync.WaitGroup
	itemChan := make(chan fetcher.SearchResult, 100)
	var seenMu sync.Mutex
	seenURLs := make(map[string]bool)

	// Smooth concurrency semaphore for SearxNG queries (max 3 concurrent) to prevent queue exhaustion
	searxSem := make(chan struct{}, 3)

	for _, targetID := range plan.SelectedTargetIDs {
		target, exists := cfg.TargetMap[targetID]
		if !exists {
			continue
		}

		queries := resolveTargetQueries(target, plan.SearchQueries)
		if len(queries) == 0 {
			log.Printf("[Bugu Module] Skipping target %s: no compatible queries for languages %v", target.ID, target.Languages)
			continue
		}

		timeRange := plan.DefaultTimeRange
		if plan.TargetTimeRanges != nil {
			if tr, ok := plan.TargetTimeRanges[target.ID]; ok && tr != "" {
				timeRange = tr
			}
		}

		for _, query := range queries {
			wgSearx.Add(1)
			go func(q string, t models.Target, tr string) {
				defer wgSearx.Done()
				searxSem <- struct{}{}
				defer func() { <-searxSem }()

				items, err := searxClient.Search(q, t, tr)
				if err != nil {
					log.Printf("[SearxNG Error] %s: %v", t.ID, err)
					return
				}
				for _, item := range items {
					if item.URL == "" {
						continue
					}
					seenMu.Lock()
					if seenURLs[item.URL] {
						seenMu.Unlock()
						continue
					}
					seenURLs[item.URL] = true
					seenMu.Unlock()
					itemChan <- item
				}
			}(query, target, timeRange)
		}
	}

	go func() {
		wgSearx.Wait()
		close(itemChan)
	}()

	// 3. Extract Markdown & Distill concurrently, with fallback to Search Snippet and discarding empty cards
	var wgJina sync.WaitGroup
	resultChan := make(chan string, 100)

	if maxConcurrency <= 0 {
		maxConcurrency = 4
	} else if maxConcurrency > 4 {
		maxConcurrency = 4
	}

	for i := 0; i < maxConcurrency; i++ {
		wgJina.Add(1)
		go func() {
			defer wgJina.Done()
			for item := range itemChan {
				var md string
				var jinaErr error
				usedSearchSnippet := false

				// Primary path: Extract full markdown via Jina Reader API
				log.Printf("[Jina Reader] Extracting -> %s", item.URL)
				md, jinaErr = jinaClient.Extract(item.URL)

				// Tier 1 Fallback: If Jina fails or is blocked on a Reddit URL, try direct Reddit .json
				if (jinaErr != nil || isBlockedOrLowQualityMarkdown(md)) && fetcher.IsRedditURL(item.URL) {
					log.Printf("[Reddit Direct Fallback] Jina failed (%v, len: %d), falling back to direct JSON -> %s", jinaErr, len(md), item.URL)
					if redditMD, rErr := redditClient.Extract(item.URL); rErr == nil && !isBlockedOrLowQualityMarkdown(redditMD) {
						md = redditMD
						jinaErr = nil
					} else {
						log.Printf("[Reddit Direct Fallback] Direct JSON also failed or blocked (%v, len: %d)", rErr, len(redditMD))
					}
				}

				// Tier 2 Fallback: If still failed or low quality, fall back to SearxNG search snippet
				if jinaErr != nil || isBlockedOrLowQualityMarkdown(md) {
					log.Printf("[Extraction Fallback] URL %s blocked or low quality (err: %v, len: %d). Falling back to search snippet.", item.URL, jinaErr, len(md))
					if strings.TrimSpace(item.Content) != "" || strings.TrimSpace(item.Title) != "" {
						md = fmt.Sprintf("# %s\n\n%s", item.Title, item.Content)
						usedSearchSnippet = true
					} else {
						// Both primary/secondary extraction and search snippet empty -> skip completely
						log.Printf("[Bugu Module] Skipping %s: extraction failed and empty search snippet", item.URL)
						continue
					}
				}

				// Distill markdown into a BabelCard
				log.Printf("[LLM Distiller] Processing -> %s", item.URL)
				card, err := router.DistillMarkdown(intent, md, item.URL, targetLanguage, kw)
				if err != nil {
					log.Printf("[Distiller Error] %s: %v", item.URL, err)
					continue
				}

				// If rejected by gating (nil returned) and we haven't used the search snippet yet, retry with snippet.
				if card == nil && !usedSearchSnippet {
					if strings.TrimSpace(item.Content) != "" {
						log.Printf("[Distiller Fallback] LLM rejected Jina markdown (no anchor hit or empty text). Retrying with search snippet for %s", item.URL)
						snippetMD := fmt.Sprintf("# %s\n\n%s", item.Title, item.Content)
						if retryCard, retryErr := router.DistillMarkdown(intent, snippetMD, item.URL, targetLanguage, kw); retryErr == nil {
							card = retryCard
						}
					}
				}

				// Final check: If still nil or empty, DO NOT show this card
				if card == nil || (strings.TrimSpace(card.OriginalText) == "" && strings.TrimSpace(card.TranslatedText) == "") {
					log.Printf("[Bugu Module] Discarding empty/rejected card for %s", item.URL)
					continue
				}

				card.EchoOf = docID

				// Broadcast the valid BabelCard immediately to the frontend
				eventBus.Publish(bus.Event{
					Type: bus.EventBabelCardSpawned,
					Data: *card,
				})

				resultChan <- fmt.Sprintf("Source: %s\n\n%s", item.URL, card.TranslatedText)
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
