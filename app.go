package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"Finnegan/bugu/api"
	"Finnegan/bugu/config"
	"Finnegan/bugu/models"
)

// App struct
type App struct {
	ctx        context.Context
	buguConfig *config.Config
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load Bugu radar targets configuration
	configPaths := []string{
		"bugu/radar_targets.yml",
		"radar_targets.yml",
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			cfg, err := config.LoadConfig(p)
			if err == nil {
				a.buguConfig = cfg
				log.Printf("[Finnegan] Bugu radar config loaded with %d targets from %s", len(cfg.Targets), p)
				break
			}
		}
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetRadarTargets returns the list of configured radar targets
func (a *App) GetRadarTargets() []models.Target {
	if a.buguConfig == nil {
		return []models.Target{}
	}
	return a.buguConfig.Targets
}

// RunRadar triggers a Bugu deep radar search for the given intent
func (a *App) RunRadar(intent string) api.RadarResponse {
	if a.buguConfig == nil {
		return api.RadarResponse{
			Status: "error",
			Error:  "Bugu configuration is not loaded",
		}
	}
	return api.ExecuteRadar(a.buguConfig, intent)
}
