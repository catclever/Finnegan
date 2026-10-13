package bugu

import (
	"Finnegan/modules/bugu/api"
	"Finnegan/core/bus"
	"sync"
	"testing"
	"time"
)

func TestCaptureListener(t *testing.T) {
	_ = bus.GetBus()

	var mu sync.Mutex
	var injectedDocID, injectedIntent string
	var injectedResults []string
	
	injectFunc := func(docID string, intent string, results []string) {
		mu.Lock()
		defer mu.Unlock()
		injectedDocID = docID
		injectedIntent = intent
		injectedResults = results
	}
	
	listener := NewCaptureListener(&BuguModule{}, 1, injectFunc)
	
	var executedIntent string
	listener.executeFunc = func(docID string, intent string) api.RadarResponse {
		mu.Lock()
		executedIntent = intent
		mu.Unlock()
		return api.RadarResponse{
			Status: "success",
			Results: []string{"mock result 1"},
		}
	}
	
	listener.Start()
	
	bus.GetBus().Publish(bus.Event{
		Type: bus.EventFileCreated,
		Data: bus.EventFileData{
			DocID: "file-doc",
			Body: "file-body",
		},
	})
	
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if executedIntent != "" {
		t.Errorf("File event should not trigger ExecuteRadar")
	}
	mu.Unlock()
	
	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID: "thought-doc",
			Text: "my thought",
			CapturedAt: time.Now(),
		},
	})
	
	time.Sleep(100 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	if executedIntent != "my thought" {
		t.Errorf("ExecuteRadar was not called with expected intent. Got: %s", executedIntent)
	}
	
	if injectedDocID != "thought-doc" || injectedIntent != "my thought" || len(injectedResults) != 1 {
		t.Errorf("injectFunc was not called with expected args")
	}
}
