package curio

import (
	"context"
	"testing"
)

func TestCurioModuleRegistration(t *testing.T) {
	m := &CurioModule{}
	if m.Name() != "curio" {
		t.Errorf("Expected name 'curio', got %s", m.Name())
	}
	if len(m.GetCommands()) != 1 {
		t.Errorf("Expected 1 command, got %d", len(m.GetCommands()))
	}
}

func TestCurioModuleExecuteCommand(t *testing.T) {
	m := &CurioModule{}
	res, err := m.ExecuteCommand(context.Background(), "curio:ingest", nil)
	if err != nil {
		t.Fatalf("Unexpected error executing curio:ingest: %v", err)
	}
	if resMap, ok := res.(map[string]interface{}); !ok || resMap["status"] != "triggered" {
		t.Errorf("Unexpected result: %v", res)
	}

	_, err = m.ExecuteCommand(context.Background(), "unknown:cmd", nil)
	if err == nil {
		t.Error("Expected error for unknown command, got none")
	}
}

func TestCurioModule_GetManifest(t *testing.T) {
	m := &CurioModule{}
	manifest := m.GetManifest()

	if manifest.Name != "curio" {
		t.Errorf("Expected Name 'curio', got '%s'", manifest.Name)
	}

	if len(manifest.CardTypes) != 1 || manifest.CardTypes[0].Type != "curio" {
		t.Errorf("Expected CardTypes to have 'curio', got %v", manifest.CardTypes)
	}

	cmds := m.GetCommands()
	if len(manifest.Commands) != len(cmds) {
		t.Errorf("Expected %d commands, got %d", len(cmds), len(manifest.Commands))
	}
}

func TestCurioModule_GetCommands(t *testing.T) {
	m := &CurioModule{}
	cmds := m.GetCommands()
	if len(cmds) == 0 {
		t.Errorf("Expected commands to not be empty")
	}
	foundIngest := false
	for _, cmd := range cmds {
		if cmd.ID == "curio:ingest" {
			foundIngest = true
			break
		}
	}
	if !foundIngest {
		t.Errorf("Expected to find command 'curio:ingest'")
	}
}
