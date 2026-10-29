package atproto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/module"
	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/fetcher"
	"Finnegan/modules/atproto/models"
	"Finnegan/modules/atproto/publisher"
)

func TestAtProtoModule_InterfaceAndCommands(t *testing.T) {
	m := &AtProtoModule{
		cfg: &config.Config{
			PDSUrl: "https://bsky.social",
		},
	}

	var _ module.Module = m
	var _ module.CommandProvider = m

	if m.Name() != "atproto" {
		t.Fatalf("expected module name 'atproto', got '%s'", m.Name())
	}

	cmds := m.GetCommands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}

	cmdMap := make(map[string]module.CommandDef)
	for _, c := range cmds {
		cmdMap[c.ID] = c
	}

	echoCmd, ok := cmdMap["atproto:echo"]
	if !ok {
		t.Fatalf("missing atproto:echo command")
	}
	if echoCmd.Trigger != "bsky" {
		t.Errorf("expected trigger 'bsky', got '%s'", echoCmd.Trigger)
	}
	if echoCmd.Namespace != "bsky" || !echoCmd.IsPrimary {
		t.Errorf("expected namespace 'bsky' and isPrimary true for atproto:echo, got %s, %v", echoCmd.Namespace, echoCmd.IsPrimary)
	}
	if len(echoCmd.Params) != 1 || echoCmd.Params[0].Type != module.ParamTypeCard || !echoCmd.Params[0].RequireKeywords {
		t.Errorf("unexpected params for atproto:echo: %+v", echoCmd.Params)
	}

	pubCmd, ok := cmdMap["atproto:publish"]
	if !ok {
		t.Fatalf("missing atproto:publish command")
	}
	if pubCmd.Trigger != "bsky-push" {
		t.Errorf("expected trigger 'bsky-push', got '%s'", pubCmd.Trigger)
	}
	if pubCmd.Namespace != "bsky" || pubCmd.IsPrimary || pubCmd.SubCommand != "push" {
		t.Errorf("expected namespace 'bsky', isPrimary false, subCommand 'push' for atproto:publish, got %s, %v, %s", pubCmd.Namespace, pubCmd.IsPrimary, pubCmd.SubCommand)
	}
	if len(pubCmd.Params) != 1 || pubCmd.Params[0].Type != module.ParamTypeCard || pubCmd.Params[0].RequireKeywords {
		t.Errorf("unexpected params for atproto:publish: %+v", pubCmd.Params)
	}
}

func TestAtProtoModule_ExecuteCommand_Echo(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		expectedQuery := "astronomy | wonder | deep space | telescope | exploration | cosmic"
		if query != expectedQuery {
			t.Errorf("expected query '%s', got '%s'", expectedQuery, query)
		}
		if !strings.Contains(r.URL.RawQuery, "%7C") {
			t.Errorf("expected RawQuery to contain escaped '|', got '%s'", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"posts": [
				{
					"uri": "at://did:plc:test1/app.bsky.feed.post/post123",
					"cid": "cid123",
					"author": {"handle": "skywalker.bsky.social", "avatar": "https://avatar.png"},
					"record": {"text": "A galaxy far away", "createdAt": "2026-10-28T00:00:00Z"},
					"replyCount": 5,
					"repostCount": 2,
					"likeCount": 10
				}
			]
		}`))
	}))
	defer mockServer.Close()

	originalEndpoint := fetcher.PublicAPIEndpoint
	originalFallback := fetcher.FallbackAPIEndpoint
	fetcher.PublicAPIEndpoint = mockServer.URL
	fetcher.FallbackAPIEndpoint = ""
	defer func() {
		fetcher.PublicAPIEndpoint = originalEndpoint
		fetcher.FallbackAPIEndpoint = originalFallback
	}()

	f := fetcher.NewFetcher()
	f.HTTPClient = mockServer.Client()

	m := &AtProtoModule{
		cfg: &config.Config{
			PDSUrl: "https://bsky.social",
		},
		fetcher: f,
	}

	b := bus.GetBus()
	spawnCh := b.Subscribe(bus.EventBabelCardSpawned, "test-subscriber")
	defer b.Unsubscribe(bus.EventBabelCardSpawned, "test-subscriber")

	doneCh := b.Subscribe(bus.EventBuguRadarCompleted, "test-subscriber-done")
	defer b.Unsubscribe(bus.EventBuguRadarCompleted, "test-subscriber-done")

	params := map[string]interface{}{
		"cardId": "card-001",
		"card": map[string]interface{}{
			"id":      "card-001",
			"content": "Deep space observation #space #astronomy",
			"keywords": map[string]interface{}{
				"object":  []interface{}{"telescope*"},
				"motif":   []interface{}{"deep space"},
				"mood":    []interface{}{"wonder*"},
				"concept": []interface{}{"astronomy"},
			},
			"tags": []interface{}{"#exploration", "cosmic*"},
		},
	}

	res, err := m.ExecuteCommand(context.Background(), "atproto:echo", params)
	if err != nil {
		t.Fatalf("ExecuteCommand atproto:echo failed: %v", err)
	}

	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "triggered" {
		t.Fatalf("expected triggered status, got: %+v", res)
	}

	// Verify keywords and tags extracted and cleaned
	kws, ok := resMap["keywords"].([]string)
	if !ok {
		t.Fatalf("expected []string keywords in result, got: %v", resMap["keywords"])
	}
	expectedKeywords := []string{"astronomy", "wonder", "deep space", "telescope", "exploration", "cosmic"}
	if !reflect.DeepEqual(kws, expectedKeywords) {
		t.Errorf("expected keywords %v, got %v", expectedKeywords, kws)
	}

	select {
	case event := <-spawnCh:
		post, ok := event.Data.(models.Post)
		if !ok {
			t.Fatalf("expected models.Post event data, got: %T", event.Data)
		}
		if post.Origin != "atproto" {
			t.Errorf("expected origin 'atproto', got '%s'", post.Origin)
		}
		if post.EchoOf != "card-001" {
			t.Errorf("expected echoOf 'card-001', got '%s'", post.EchoOf)
		}
		if post.ReplyCount != 5 || post.RepostCount != 2 || post.LikeCount != 10 {
			t.Errorf("unexpected counts: reply=%d repost=%d like=%d", post.ReplyCount, post.RepostCount, post.LikeCount)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for BabelCardSpawned event")
	}

	select {
	case event := <-doneCh:
		doneMap, ok := event.Data.(map[string]string)
		if !ok || doneMap["cardId"] != "card-001" {
			t.Errorf("unexpected EventBuguRadarCompleted data: %+v", event.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for BuguRadarCompleted event")
	}
}

func TestAtProtoModule_ExecuteCommand_Echo_FreeText(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query != "quantum computing" {
			t.Errorf("expected query 'quantum computing', got '%s'", query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"posts": [
				{
					"uri": "at://did:plc:test2/app.bsky.feed.post/post789",
					"cid": "cid789",
					"author": {"handle": "quantum.bsky.social", "avatar": "https://avatar2.png"},
					"record": {"text": "Qubits are entangled", "createdAt": "2026-10-28T00:00:00Z"},
					"replyCount": 1,
					"repostCount": 0,
					"likeCount": 4
				}
			]
		}`))
	}))
	defer mockServer.Close()

	originalEndpoint := fetcher.PublicAPIEndpoint
	originalFallback := fetcher.FallbackAPIEndpoint
	fetcher.PublicAPIEndpoint = mockServer.URL
	fetcher.FallbackAPIEndpoint = ""
	defer func() {
		fetcher.PublicAPIEndpoint = originalEndpoint
		fetcher.FallbackAPIEndpoint = originalFallback
	}()

	f := fetcher.NewFetcher()
	f.HTTPClient = mockServer.Client()

	m := &AtProtoModule{
		cfg: &config.Config{
			PDSUrl: "https://bsky.social",
		},
		fetcher: f,
	}

	b := bus.GetBus()
	spawnCh := b.Subscribe(bus.EventBabelCardSpawned, "test-subscriber-freetext")
	defer b.Unsubscribe(bus.EventBabelCardSpawned, "test-subscriber-freetext")

	doneCh := b.Subscribe(bus.EventBuguRadarCompleted, "test-subscriber-freetext-done")
	defer b.Unsubscribe(bus.EventBuguRadarCompleted, "test-subscriber-freetext-done")

	params := map[string]interface{}{
		"text": "quantum computing",
	}

	res, err := m.ExecuteCommand(context.Background(), "atproto:echo", params)
	if err != nil {
		t.Fatalf("ExecuteCommand atproto:echo failed: %v", err)
	}

	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "triggered" {
		t.Fatalf("expected triggered status, got: %+v", res)
	}
	if resMap["card_id"] != "" {
		t.Errorf("expected empty card_id for free text search, got '%v'", resMap["card_id"])
	}

	kws, ok := resMap["keywords"].([]string)
	if !ok || !reflect.DeepEqual(kws, []string{"quantum computing"}) {
		t.Errorf("expected keywords ['quantum computing'], got %v", resMap["keywords"])
	}

	select {
	case event := <-spawnCh:
		post, ok := event.Data.(models.Post)
		if !ok {
			t.Fatalf("expected models.Post event data, got: %T", event.Data)
		}
		if post.Origin != "atproto" {
			t.Errorf("expected origin 'atproto', got '%s'", post.Origin)
		}
		if post.EchoOf != "" {
			t.Errorf("expected empty EchoOf, got '%s'", post.EchoOf)
		}
		if post.AuthorName != "quantum.bsky.social" {
			t.Errorf("expected author 'quantum.bsky.social', got '%s'", post.AuthorName)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for BabelCardSpawned event")
	}

	select {
	case event := <-doneCh:
		doneMap, ok := event.Data.(map[string]string)
		if !ok || doneMap["cardId"] != "" {
			t.Errorf("expected empty cardId in EventBuguRadarCompleted, got: %+v", event.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for BuguRadarCompleted event")
	}
}

func TestAtProtoModule_ExecuteCommand_Echo_Empty(t *testing.T) {
	m := &AtProtoModule{
		cfg: &config.Config{
			PDSUrl: "https://bsky.social",
		},
	}
	_, err := m.ExecuteCommand(context.Background(), "atproto:echo", map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected error for empty search, got nil")
	}
	if !strings.Contains(err.Error(), "no content or card provided for atproto:echo") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAtProtoModule_ExecuteCommand_Publish(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"accessJwt": "mock-access", "refreshJwt": "mock-refresh", "did": "did:plc:test"}`))
	})
	mux.HandleFunc("/xrpc/com.atproto.repo.createRecord", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"uri": "at://did:plc:test/app.bsky.feed.post/post456", "cid": "cid456"}`))
	})
	mockServer := httptest.NewServer(mux)
	defer mockServer.Close()

	cfg := &config.Config{
		Handle:   "test.bsky.social",
		Password: "password",
		PDSUrl:   mockServer.URL,
	}

	pub := publisher.NewPublisher(cfg)
	m := &AtProtoModule{
		cfg:       cfg,
		publisher: pub,
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "card.md")
	initialContent := "---\ntitle: Publish Test\n---\nHello Bluesky World!"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	params := map[string]interface{}{
		"filePath": filePath,
		"content":  "Hello Bluesky World!",
	}

	res, err := m.ExecuteCommand(context.Background(), "atproto:publish", params)
	if err != nil {
		t.Fatalf("ExecuteCommand atproto:publish failed: %v", err)
	}

	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "success" {
		t.Fatalf("expected success status, got: %+v", res)
	}
	if resMap["uri"] != "at://did:plc:test/app.bsky.feed.post/post456" {
		t.Errorf("unexpected uri: %v", resMap["uri"])
	}

	// Verify frontmatter was updated in file
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(contentBytes)
	if !strings.Contains(contentStr, "uri: at://did:plc:test/app.bsky.feed.post/post456") {
		t.Errorf("expected frontmatter to contain updated uri, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "published: true") {
		t.Errorf("expected frontmatter to contain published: true, got: %s", contentStr)
	}
}
