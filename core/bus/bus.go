package bus

import (
	"log"
	"sync"
	"time"
)

type EventType string

const (
	EventFileCreated        EventType = "FileCreated"
	EventFileModified       EventType = "FileModified"
	EventFileDeleted        EventType = "FileDeleted"
	EventThoughtCaptured    EventType = "ThoughtCaptured"
	EventBabelCardSpawned   EventType = "BabelCardSpawned"
	EventBuguRadarCompleted EventType = "BuguRadarCompleted"
)

type EventFileData struct {
	FilePath string
	DocID    string
	NewMeta  map[string]any
	OldMeta  map[string]any // nil on create
	Body     string
	Origin   string
}

type EventThoughtCapturedData struct {
	DocID        string     `json:"DocID"`
	FilePath     string     `json:"FilePath"`
	Text         string     `json:"Text"`
	CapturedAt   time.Time  `json:"CapturedAt"`
	Images       []string   `json:"Images,omitempty"`
	Title        string     `json:"Title,omitempty"`
	Type         string     `json:"Type,omitempty"`
	Participants []string   `json:"Participants,omitempty"`
	StartedAt    *time.Time `json:"StartedAt,omitempty"`
	EndedAt      *time.Time `json:"EndedAt,omitempty"`
}

type Event struct {
	Type EventType
	Data interface{}
}

type GlobalEventBus struct {
	subscribers map[EventType]map[string]chan Event
	mutex       sync.RWMutex
}

var instance *GlobalEventBus
var once sync.Once

func GetBus() *GlobalEventBus {
	once.Do(func() {
		instance = &GlobalEventBus{
			subscribers: make(map[EventType]map[string]chan Event),
		}
	})
	return instance
}

func (b *GlobalEventBus) Subscribe(eventType EventType, name string) <-chan Event {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	ch := make(chan Event, 100)
	if b.subscribers[eventType] == nil {
		b.subscribers[eventType] = make(map[string]chan Event)
	}
	b.subscribers[eventType][name] = ch
	return ch
}

func (b *GlobalEventBus) Unsubscribe(eventType EventType, name string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.subscribers[eventType] != nil {
		if ch, ok := b.subscribers[eventType][name]; ok {
			close(ch)
			delete(b.subscribers[eventType], name)
		}
	}
}

func (b *GlobalEventBus) Publish(event Event) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for name, ch := range b.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
			log.Printf("Warning: Subscriber %s channel is full, skipping event %v", name, event.Type)
		}
	}
}

func (b *GlobalEventBus) Close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, subs := range b.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	b.subscribers = make(map[EventType]map[string]chan Event)
}
