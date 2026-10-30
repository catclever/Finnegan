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
