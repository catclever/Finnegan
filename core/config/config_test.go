package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSkeletonAndPointer(t *testing.T) {
	// Setup temp home
	tempHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	cfg, home, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expectedHome := filepath.Join(tempHome, "Finnegan")
	if home != expectedHome {
		t.Errorf("Expected home %s, got %s", expectedHome, home)
	}

	// Check skeleton
	dirs := []string{
		filepath.Join(expectedHome, "Scattering"),
		filepath.Join(expectedHome, "Weaving"),
		filepath.Join(expectedHome, ".finnegan", "bases"),
	}
	for _, d := range dirs {
		if stat, err := os.Stat(d); err != nil || !stat.IsDir() {
			t.Errorf("Expected directory %s to exist", d)
		}
	}

	if cfg.DebounceMs != 300 {
		t.Errorf("Expected debounce 300, got %d", cfg.DebounceMs)
	}
}

func TestDeriveWatchDirs(t *testing.T) {
	home := t.TempDir()
	ext1 := filepath.Join(t.TempDir(), "ext1")
	ext2 := filepath.Join(t.TempDir(), "ext2")
	os.MkdirAll(ext1, 0755)
	os.MkdirAll(ext2, 0755)
	
	// Create marker in ext1, not ext2
	os.WriteFile(filepath.Join(ext1, ".finnegan-project"), []byte(""), 0644)

	dirs := DeriveWatchDirs(home, []string{ext1, ext2})
	
	expected := []string{
		filepath.Join(home, "Scattering"),
		filepath.Join(home, "Weaving"),
		ext1,
	}
	
	if len(dirs) != len(expected) {
		t.Errorf("Expected %d watch dirs, got %d", len(expected), len(dirs))
	}
	for i, d := range expected {
		if i < len(dirs) && dirs[i] != d {
			t.Errorf("Expected %s at %d, got %s", d, i, dirs[i])
		}
	}
}
