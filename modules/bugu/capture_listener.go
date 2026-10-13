package bugu

import (
	"log"
	"golang.org/x/sync/semaphore"
	"context"

	"Finnegan/modules/bugu/api"
	"Finnegan/core/bus"
)

type CaptureListener struct {
	module     *BuguModule
	injectFunc func(docID string, intent string, results []string)
	sem        *semaphore.Weighted
	ctx        context.Context
	executeFunc func(intent string) api.RadarResponse
}

func NewCaptureListener(m *BuguModule, concurrency int, injectFunc func(docID string, intent string, results []string)) *CaptureListener {
	if concurrency < 1 {
		concurrency = 1
	}
	l := &CaptureListener{
		module:      m,
		injectFunc:  injectFunc,
		sem:         semaphore.NewWeighted(int64(concurrency)),
		ctx:         context.Background(),
	}
	l.executeFunc = m.RunRadar
	return l
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

	if l.module == nil {
		log.Printf("Bugu CaptureListener module is nil, skipping radar for DocID %s", data.DocID)
		return
	}

	if err := l.sem.Acquire(l.ctx, 1); err != nil {
		log.Printf("Bugu CaptureListener semaphore error: %v", err)
		return
	}
	defer l.sem.Release(1)

	resp := l.executeFunc(data.Text)
	if resp.Status == "error" {
		log.Printf("Bugu radar failed for DocID %s: %s", data.DocID, resp.Error)
		return
	}

	if l.injectFunc != nil {
		l.injectFunc(data.DocID, data.Text, resp.Results)
	}
}
