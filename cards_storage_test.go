package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCardFileWithLocationAndWeather(t *testing.T) {
	content := `---
id: diary_123
type: diary
title: My Diary
location:
  place_id: chij_123
  name: Chijmes
  landmark: City Hall
  lat: 1.294
  lng: 103.851
weather:
  condition: Rain
  wmo_code: 61
  icon: rain.png
  temp_c: 25.5
  observed_hour: "2026-11-06 14"
  is_day: true
---

Hello world!`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "diary.md")
	os.WriteFile(filePath, []byte(content), 0644)

	card, err := ParseCardFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse card file: %v", err)
	}

	if card.Location == nil {
		t.Fatal("Expected location to be parsed, got nil")
	}
	if card.Location.Name != "Chijmes" {
		t.Errorf("Expected location name 'Chijmes', got '%s'", card.Location.Name)
	}
	if card.Location.Lat != 1.294 {
		t.Errorf("Expected lat 1.294, got %v", card.Location.Lat)
	}

	if card.Weather == nil {
		t.Fatal("Expected weather to be parsed, got nil")
	}
	if card.Weather.Condition != "Rain" {
		t.Errorf("Expected weather condition 'Rain', got '%s'", card.Weather.Condition)
	}
	if !card.Weather.IsDay {
		t.Errorf("Expected weather is_day to be true")
	}
}

func TestCardMeta_YAMLRoundTrip_And_StripCoreKeys(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	tmpHome := t.TempDir()
	now := time.Now()

	// 1. Temporary card with custom type + custom Meta + legacy Brand field
	tempCard := Card{
		ID:        "temp-plugin-001",
		Type:      CardType("taohuo"),
		Origin:    "taohuo",
		Title:     "Leica M6 TTL",
		Content:   "Classic rangefinder body #camera",
		Brand:     "Leica",
		Tags:      []string{"camera", "analog"},
		Images:    []string{"m6.png"},
		CreatedAt: now,
		Meta: map[string]interface{}{
			"rating": 4.9,
			"isbn":   "978-3-16-148410-0",
			"sku":    "SKU-M6-085",
			"brand":  "Leica", // collides with struct field Brand; should not produce duplicate YAML key
			"id":     "should-be-ignored",
			"nested": map[string]interface{}{
				"currency": "CNY",
				"verified": true,
			},
		},
	}

	cm.InjectBabelCards([]Card{tempCard})

	promoted, err := cm.PromoteToLocal("temp-plugin-001", tmpHome)
	if err != nil {
		t.Fatalf("PromoteToLocal failed: %v", err)
	}
	if promoted.Type != CardType("taohuo") {
		t.Fatalf("expected promoted type 'taohuo', got %q", promoted.Type)
	}

	loaded, err := ParseCardFile(promoted.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile failed: %v", err)
	}

	if loaded.ID != promoted.ID {
		t.Errorf("expected ID %q, got %q", promoted.ID, loaded.ID)
	}
	if loaded.Type != CardType("taohuo") {
		t.Errorf("expected Type 'taohuo', got %q", loaded.Type)
	}
	if loaded.Brand != "Leica" {
		t.Errorf("expected Brand 'Leica', got %q", loaded.Brand)
	}
	if loaded.Meta == nil {
		t.Fatal("expected non-nil card.Meta after ParseCardFile")
	}
	if loaded.Meta["rating"] != 4.9 {
		t.Errorf("expected Meta['rating'] == 4.9, got %v (%T)", loaded.Meta["rating"], loaded.Meta["rating"])
	}
	if loaded.Meta["isbn"] != "978-3-16-148410-0" {
		t.Errorf("expected Meta['isbn'], got %v", loaded.Meta["isbn"])
	}
	if loaded.Meta["sku"] != "SKU-M6-085" {
		t.Errorf("expected Meta['sku'], got %v", loaded.Meta["sku"])
	}
	if loaded.Meta["brand"] != "Leica" {
		t.Errorf("expected Meta['brand'] == 'Leica', got %v", loaded.Meta["brand"])
	}
	nested, ok := loaded.Meta["nested"].(map[string]interface{})
	if !ok || nested["currency"] != "CNY" || nested["verified"] != true {
		t.Errorf("expected Meta['nested'] preserved, got %#v", loaded.Meta["nested"])
	}

	for _, coreKey := range []string{"id", "type", "title", "source", "created", "updated", "publish", "images", "tags"} {
		if _, exists := loaded.Meta[coreKey]; exists {
			t.Errorf("core key %q should have been stripped from card.Meta", coreKey)
		}
	}
}
