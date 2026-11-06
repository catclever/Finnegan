package main

import (
	"os"
	"path/filepath"
	"testing"
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
