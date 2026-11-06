package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Place struct {
	ID        string    `yaml:"id" json:"id"`
	Name      string    `yaml:"name" json:"name"`
	Landmark  string    `yaml:"landmark" json:"landmark"`
	Lat       float64   `yaml:"lat" json:"lat"`
	Lng       float64   `yaml:"lng" json:"lng"`
	CreatedAt time.Time `yaml:"created_at" json:"createdAt"`
	UseCount  int       `yaml:"use_count" json:"useCount"`
}

type PlaceWithDistance struct {
	Place
	DistanceM float64 `json:"distanceM"`
}

type LandmarkCandidate struct {
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Distance float64 `json:"distance"`
}

var (
	placesMu sync.RWMutex
	places   map[string]*Place
)

func getPlacesFilePath(dataDir string) string {
	if dataDir != "" {
		return filepath.Join(dataDir, "places.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".finnegan", "places.yml")
}

func loadPlaces(dataDir string) {
	placesMu.Lock()
	defer placesMu.Unlock()

	if places != nil {
		return // already loaded
	}

	places = make(map[string]*Place)

	path := getPlacesFilePath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var loadedPlaces []Place
	if err := yaml.Unmarshal(data, &loadedPlaces); err == nil {
		for i, p := range loadedPlaces {
			places[p.ID] = &loadedPlaces[i]
		}
	}
}

func savePlaces(dataDir string) {
	placesMu.RLock()
	var list []Place
	for _, p := range places {
		list = append(list, *p)
	}
	placesMu.RUnlock()

	path := getPlacesFilePath(dataDir)
	os.MkdirAll(filepath.Dir(path), 0755)

	data, err := yaml.Marshal(list)
	if err == nil {
		os.WriteFile(path, data, 0644)
	}
}

func FindNearbyPlaces(dataDir string, lat, lng, radiusM float64) []PlaceWithDistance {
	loadPlaces(dataDir)
	placesMu.RLock()
	defer placesMu.RUnlock()

	var result []PlaceWithDistance
	for _, p := range places {
		dist := HaversineDistanceMeters(lat, lng, p.Lat, p.Lng)
		if dist <= radiusM {
			result = append(result, PlaceWithDistance{Place: *p, DistanceM: dist})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].DistanceM != result[j].DistanceM {
			return result[i].DistanceM < result[j].DistanceM
		}
		return result[i].UseCount > result[j].UseCount
	})

	return result
}

func CreatePlace(dataDir string, name, landmark string, lat, lng float64) *Place {
	loadPlaces(dataDir)

	placesMu.Lock()
	// Check if a place with the same name exists within 15m
	for _, p := range places {
		if p.Name == name {
			dist := HaversineDistanceMeters(lat, lng, p.Lat, p.Lng)
			if dist <= 15.0 {
				p.UseCount++
				placesMu.Unlock()
				savePlaces(dataDir)
				return p
			}
		}
	}

	id := fmt.Sprintf("place_%d", time.Now().UnixNano())
	newPlace := &Place{
		ID:        id,
		Name:      name,
		Landmark:  landmark,
		Lat:       lat,
		Lng:       lng,
		CreatedAt: time.Now(),
		UseCount:  1,
	}
	places[id] = newPlace
	placesMu.Unlock()

	savePlaces(dataDir)
	return newPlace
}

func FetchNearbyLandmarks(ctx context.Context, lat, lng float64, provider, googleAPIKey, amapAPIKey string) ([]LandmarkCandidate, error) {
	if provider == "google" && googleAPIKey != "" {
		return fetchGoogleLandmarks(ctx, lat, lng, googleAPIKey)
	} else if provider == "amap" && amapAPIKey != "" {
		return fetchAMapLandmarks(ctx, lat, lng, amapAPIKey)
	}
	return fetchOSMLandmarks(ctx, lat, lng)
}

func fetchGoogleLandmarks(ctx context.Context, lat, lng float64, apiKey string) ([]LandmarkCandidate, error) {
	// Simple stub for Google Places API
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f,%f&radius=100&key=%s", lat, lng, apiKey)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			Name string `json:"name"`
			Vicinity string `json:"vicinity"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var candidates []LandmarkCandidate
	for _, r := range data.Results {
		candidates = append(candidates, LandmarkCandidate{
			Name: r.Name,
			Address: r.Vicinity,
		})
	}
	return candidates, nil
}

func fetchAMapLandmarks(ctx context.Context, lat, lng float64, apiKey string) ([]LandmarkCandidate, error) {
	// Convert to GCJ02 first
	gcjLat, gcjLng := WGS84ToGCJ02(lat, lng)
	url := fmt.Sprintf("https://restapi.amap.com/v3/geocode/regeo?location=%f,%f&key=%s&radius=100&extensions=all", gcjLng, gcjLat, apiKey)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Regeocode struct {
			Pois []struct {
				Name string `json:"name"`
				Address string `json:"address"`
				Distance string `json:"distance"`
			} `json:"pois"`
		} `json:"regeocode"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var candidates []LandmarkCandidate
	for _, p := range data.Regeocode.Pois {
		var dist float64
		fmt.Sscanf(p.Distance, "%f", &dist)
		candidates = append(candidates, LandmarkCandidate{
			Name: p.Name,
			Address: p.Address,
			Distance: dist,
		})
	}
	return candidates, nil
}

func fetchOSMLandmarks(ctx context.Context, lat, lng float64) ([]LandmarkCandidate, error) {
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f&zoom=18&addressdetails=1", lat, lng)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Finnegan-App/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data struct {
		DisplayName string `json:"display_name"`
		Address map[string]string `json:"address"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	name := data.Address["amenity"]
	if name == "" {
		name = data.Address["building"]
	}
	if name == "" {
		name = data.Address["road"]
	}
	if name == "" {
		name = "Unknown Landmark"
	}

	return []LandmarkCandidate{{
		Name: name,
		Address: data.DisplayName,
	}}, nil
}
