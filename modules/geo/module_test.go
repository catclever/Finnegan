package geo

import (
	"context"
	"testing"
	"time"
)

func TestIsModuleAllowed(t *testing.T) {
	m := &GeoModule{
		config: GeoConfig{
			Enabled:        true,
			AllowedModules: []string{"Diary"},
		},
	}

	if !m.IsModuleAllowed("diary") {
		t.Error("Expected 'diary' to be allowed (case-insensitive)")
	}

	if m.IsModuleAllowed("notes") {
		t.Error("Expected 'notes' to not be allowed")
	}

	m.config.Enabled = false
	if m.IsModuleAllowed("diary") {
		t.Error("Expected module to not be allowed when disabled")
	}
}

func TestPlaceCreationAndRecommendation(t *testing.T) {
	dataDir := t.TempDir()

	// Create place
	p1 := CreatePlace(dataDir, "阳台躺椅", "My Apartment", 30.0, 120.0)
	if p1.UseCount != 1 {
		t.Errorf("Expected use count 1, got %d", p1.UseCount)
	}

	// Create place at exactly the same coordinates with different name
	p2 := CreatePlace(dataDir, "三楼书房", "My Apartment", 30.0, 120.0)
	if p2.Name != "三楼书房" {
		t.Errorf("Expected place to be created with new name")
	}
	if p1.ID == p2.ID {
		t.Errorf("Expected different IDs")
	}

	// Create place at the same coordinates with same name
	p3 := CreatePlace(dataDir, "阳台躺椅", "My Apartment", 30.0, 120.00001) // within 15m
	if p3.ID != p1.ID {
		t.Errorf("Expected same place to be updated")
	}
	if p3.UseCount != 2 {
		t.Errorf("Expected use count 2, got %d", p3.UseCount)
	}

	// Create remote out-of-range place
	p4 := CreatePlace(dataDir, "Paris Eiffel", "Eiffel Tower", 48.8584, 2.2945)
	if p4.ID == "" {
		t.Errorf("Expected remote place to be created")
	}

	// Recommend places
	nearby := FindNearbyPlaces(dataDir, 30.0, 120.0, 100.0)
	if len(nearby) != 2 {
		t.Errorf("Expected 2 nearby places, got %d", len(nearby))
	}
	if nearby[0].ID != p1.ID {
		t.Errorf("Expected p1 to be first (highest use count)")
	}
}

func TestGetWeatherAtHour_FutureTime(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(2 * time.Hour)
	_, err := GetWeatherAtHour(ctx, 30.0, 120.0, future)
	if err != ErrFutureTimeNotAllowed {
		t.Errorf("Expected ErrFutureTimeNotAllowed, got %v", err)
	}
}
