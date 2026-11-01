package bugu

import (
	"testing"


)

func TestBuguModule_GetManifest(t *testing.T) {
	m := &BuguModule{}
	manifest := m.GetManifest()

	if manifest.Name != "bugu" {
		t.Errorf("Expected Name 'bugu', got '%s'", manifest.Name)
	}
	if manifest.ThemeColor != "#f59e0b" {
		t.Errorf("Expected ThemeColor '#f59e0b', got '%s'", manifest.ThemeColor)
	}
	if manifest.BeaconType != "radar" {
		t.Errorf("Expected BeaconType 'radar', got '%s'", manifest.BeaconType)
	}
	if manifest.PromotionStrategy == nil {
		t.Fatalf("Expected PromotionStrategy to not be nil")
	}
	if !manifest.PromotionStrategy.HasCommentPrompt {
		t.Errorf("Expected HasCommentPrompt to be true")
	}
	if !manifest.PromotionStrategy.BindMotherCard {
		t.Errorf("Expected BindMotherCard to be true")
	}

	cmds := m.GetCommands()
	if len(manifest.Commands) != len(cmds) {
		t.Errorf("Expected %d commands, got %d", len(cmds), len(manifest.Commands))
	}
}

func TestBuguModule_GetCommands(t *testing.T) {
	m := &BuguModule{}
	cmds := m.GetCommands()
	if len(cmds) == 0 {
		t.Errorf("Expected commands to not be empty")
	}
	foundPing := false
	for _, cmd := range cmds {
		if cmd.ID == "bugu:radar" {
			foundPing = true
			break
		}
	}
	if !foundPing {
		t.Errorf("Expected to find command 'bugu:ping'")
	}
}
