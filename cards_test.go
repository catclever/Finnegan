package main

import (
	"testing"
	"time"
)

func TestCardManager_Expiration(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	now := time.Now()
	freshCard := Card{
		ID:        "fresh-1",
		Type:      CardTypeBabel,
		Title:     "Fresh Card",
		CreatedAt: now,
	}
	expiredCard := Card{
		ID:        "expired-1",
		Type:      CardTypeBabel,
		Title:     "Expired Card",
		CreatedAt: now.Add(-25 * time.Hour),
	}

	cm.InjectBabelCards([]Card{freshCard, expiredCard})

	// GetTempCards should lazily filter and purge expired cards
	active := cm.GetTempCards()
	if len(active) != 1 {
		t.Fatalf("expected 1 active card, got %d", len(active))
	}
	if active[0].ID != "fresh-1" {
		t.Fatalf("expected fresh-1 to remain, got %s", active[0].ID)
	}

	// Verify the expired card was purged from internal storage
	cm.mutex.RLock()
	_, exists := cm.tempCards["expired-1"]
	cm.mutex.RUnlock()
	if exists {
		t.Fatal("expected expired-1 to be deleted from tempCards map")
	}
}

func TestCardManager_CleanExpiredCards(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	now := time.Now()
	cards := []Card{
		{ID: "c1", Type: CardTypeBabel, CreatedAt: now.Add(-26 * time.Hour)},
		{ID: "c2", Type: CardTypeBabel, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "c3", Type: CardTypeChatLog, CreatedAt: now.Add(-30 * time.Hour)},
	}

	cm.InjectBabelCards([]Card{cards[0], cards[1]})
	cm.InjectChatLogCard(cards[2])

	removed := cm.CleanExpiredCards()
	if removed != 2 {
		t.Fatalf("expected 2 removed cards, got %d", removed)
	}

	active := cm.GetTempCards()
	if len(active) != 1 || active[0].ID != "c2" {
		t.Fatalf("expected only c2 to remain, got %+v", active)
	}
}

func TestCardManager_EvictTempCard(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	cm.InjectBabelCards([]Card{{ID: "target-1", Type: CardTypeBabel, CreatedAt: time.Now()}})

	if !cm.EvictTempCard("target-1") {
		t.Fatal("expected EvictTempCard to return true for existing card")
	}
	if cm.EvictTempCard("target-1") {
		t.Fatal("expected EvictTempCard to return false for already deleted card")
	}
	if cm.EvictTempCard("non-existent") {
		t.Fatal("expected EvictTempCard to return false for non-existent card")
	}

	if len(cm.GetTempCards()) != 0 {
		t.Fatal("expected tempCards to be empty after eviction")
	}
}

func TestExtractHashtags(t *testing.T) {
	text := "今天测试了 #WebKitGTK 4.1 和 #跨平台UI, 以及 #v1.0.0。顺便体验了 #松弛感！"
	tags := ExtractHashtags(text)

	expected := []string{"WebKitGTK", "跨平台UI", "v1.0.0", "松弛感"}
	if len(tags) != len(expected) {
		t.Fatalf("expected %d tags, got %d: %v", len(expected), len(tags), tags)
	}
	for i, exp := range expected {
		if tags[i] != exp {
			t.Errorf("expected tag %d to be %q, got %q", i, exp, tags[i])
		}
	}
}
