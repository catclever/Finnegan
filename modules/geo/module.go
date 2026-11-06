package geo

import (
	"context"
	"strings"
	"time"

	"Finnegan/core/module"
)

// GeoConfig holds configuration for the geo module
type GeoConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	Provider         string   `yaml:"provider" json:"provider"`
	GoogleAPIKey     string   `yaml:"google_api_key" json:"googleApiKey"`
	AMapAPIKey       string   `yaml:"amap_api_key" json:"amapApiKey"`
	RecommendRadiusM float64  `yaml:"recommend_radius_m" json:"recommendRadiusM"`
	AllowedModules   []string `yaml:"allowed_modules" json:"allowedModules"`
	DataDir          string   `yaml:"data_dir" json:"-"`
}

func (c *GeoConfig) setDefaults() {
	c.Enabled = true
	c.Provider = "auto"
	c.RecommendRadiusM = 100.0
	c.AllowedModules = []string{"diary"}
}

// GeoModule provides geospatial services, nearby places, and hourly weather.
type GeoModule struct {
	config GeoConfig
	ctx    *module.Context
}

func init() {
	module.Register(&GeoModule{
		config: GeoConfig{
			Enabled:          true,
			Provider:         "auto",
			RecommendRadiusM: 100.0,
			AllowedModules:   []string{"diary"},
		},
	})
}

func (m *GeoModule) Name() string {
	return "geo"
}

func (m *GeoModule) ConfigTarget() interface{} {
	return &m.config
}

func (m *GeoModule) Init(ctx *module.Context) error {
	m.ctx = ctx
	// We init defaults here in case the unmarshal left them as zero values
	if m.config.Provider == "" {
		m.config.Provider = "auto"
	}
	if m.config.RecommendRadiusM == 0 {
		m.config.RecommendRadiusM = 100.0
	}
	if m.config.AllowedModules == nil {
		m.config.AllowedModules = []string{"diary"}
	}
	return nil
}

func (m *GeoModule) Bindings() []interface{} {
	return nil
}

func (m *GeoModule) GetManifest() module.ModuleManifest {
	return module.ModuleManifest{
		Name:        "geo",
		Description: "Geolocation, Places and Hourly Weather Module",
	}
}

func (m *GeoModule) GetCommands() []module.CommandDef {
	return nil
}

func (m *GeoModule) IsModuleAllowed(moduleName string) bool {
	if !m.config.Enabled {
		return false
	}
	target := strings.ToLower(moduleName)
	for _, a := range m.config.AllowedModules {
		if strings.ToLower(a) == target {
			return true
		}
	}
	return false
}

func (m *GeoModule) ExecuteAction(ctx context.Context, actionID string, payload map[string]interface{}) (interface{}, error) {
	switch actionID {
	case "check_permission":
		moduleName, _ := payload["module"].(string)
		return map[string]interface{}{
			"enabled":          m.config.Enabled,
			"allowed":          m.IsModuleAllowed(moduleName),
			"recommendRadiusM": m.config.RecommendRadiusM,
		}, nil
	case "resolve_context":
		moduleName, _ := payload["module"].(string)
		lat, _ := payload["lat"].(float64)
		lng, _ := payload["lng"].(float64)
		timestampStr, _ := payload["timestamp"].(string)

		if !m.IsModuleAllowed(moduleName) {
			return nil, nil // Or an error
		}

		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			timestamp = time.Now()
		}

		places := FindNearbyPlaces(m.config.DataDir, lat, lng, m.config.RecommendRadiusM)
		landmarks, _ := FetchNearbyLandmarks(ctx, lat, lng, m.config.Provider, m.config.GoogleAPIKey, m.config.AMapAPIKey)
		weather, _ := GetWeatherAtHour(ctx, lat, lng, timestamp)

		return map[string]interface{}{
			"places":    places,
			"landmarks": landmarks,
			"weather":   weather,
		}, nil
	case "list_nearby_places":
		lat, _ := payload["lat"].(float64)
		lng, _ := payload["lng"].(float64)
		radiusM, _ := payload["radiusM"].(float64)
		if radiusM == 0 {
			radiusM = m.config.RecommendRadiusM
		}
		places := FindNearbyPlaces(m.config.DataDir, lat, lng, radiusM)
		return places, nil
	case "create_place":
		name, _ := payload["name"].(string)
		landmark, _ := payload["landmark"].(string)
		lat, _ := payload["lat"].(float64)
		lng, _ := payload["lng"].(float64)
		place := CreatePlace(m.config.DataDir, name, landmark, lat, lng)
		return place, nil
	case "get_weather":
		lat, _ := payload["lat"].(float64)
		lng, _ := payload["lng"].(float64)
		timestampStr, _ := payload["timestamp"].(string)
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			timestamp = time.Now()
		}
		weather, err := GetWeatherAtHour(ctx, lat, lng, timestamp)
		if err != nil {
			return nil, err
		}
		return weather, nil
	}
	return nil, nil
}
