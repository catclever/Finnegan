package bugu

import (
	"log"
	"golang.org/x/sync/semaphore"
	"context"

	"Finnegan/bugu/api"
	"Finnegan/bugu/config"
	"Finnegan/core/bus"
)

type CaptureListener struct {
	cfg        *config.Config
	injectFunc func(docID string, intent string, results []string)
	sem        *semaphore.Weighted
	ctx        context.Context
	executeFunc func(cfg *config.Config, intent string) api.RadarResponse
}

func NewCaptureListener(cfg *config.Config, concurrency int, injectFunc func(docID string, intent string, results []string)) *CaptureListener {
	if concurrency < 1 {
		concurrency = 1
	}
	return &CaptureListener{
		cfg:         cfg,
		injectFunc:  injectFunc,
		sem:         semaphore.NewWeighted(int64(concurrency)),
		ctx:         context.Background(),
		executeFunc: api.ExecuteRadar,
	}
}

func (l *CaptureListener) Start() {
	eventBus := bus.GetBus()
	ch := eventBus.Subscribe(bus.EventThoughtCaptured, "bugu")

	go func() {
		for event := range ch {
			data, ok := event.Data.(bus.EventThoughtCapturedData)
			if !ok {
				continue
			}

			go l.processEvent(data)
		}
	}()
}

func (l *CaptureListener) processEvent(data bus.EventThoughtCapturedData) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Bugu CaptureListener panic for DocID %s: %v", data.DocID, r)
		}
	}()

	if l.cfg == nil {
		log.Printf("Bugu CaptureListener config is nil, skipping radar for DocID %s", data.DocID)
		return
	}

	if err := l.sem.Acquire(l.ctx, 1); err != nil {
		log.Printf("Bugu CaptureListener semaphore error: %v", err)
		return
	}
	defer l.sem.Release(1)

	resp := l.executeFunc(l.cfg, data.Text)
	if resp.Status == "error" {
		log.Printf("Bugu radar failed for DocID %s: %s", data.DocID, resp.Error)
		return
	}

	if l.injectFunc != nil {
		l.injectFunc(data.DocID, data.Text, resp.Results)
	}
}
