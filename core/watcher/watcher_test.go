package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type mockStorage struct {
	notes       map[string]string
	deleted     map[string]bool
	conflicts   map[string][]string
	mu          sync.Mutex
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		notes:     make(map[string]string),
		deleted:   make(map[string]bool),
		conflicts: make(map[string][]string),
	}
}

func (m *mockStorage) SaveNote(id, data string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notes[id] = data
	return nil
}
func (m *mockStorage) DeleteNote(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted[id] = true
	return nil
}
func (m *mockStorage) GetNote(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notes[id], nil
}
func (m *mockStorage) EmitConflict(id string, paths []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conflicts[id] = paths
}

func TestWatcherExternalNoID(t *testing.T) {
	tempHome := t.TempDir()
	extDir := t.TempDir()
	
	w := &GlobalWatcher{
		stopChan:       make(chan struct{}),
		pending:        make(map[string]*time.Timer),
		fileHashes:     make(map[string]string),
		expectedWrites: make(map[string]ExpectedWrite),
		pathToID:       make(map[string]string),
		idToPath:       make(map[string]string),
		fileMutexes:    make(map[string]*sync.Mutex),
	}
	
	storage := newMockStorage()
	w.Initialize([]string{extDir}, 10, storage, tempHome)
	
	testFile := filepath.Join(extDir, "test.md")
	os.WriteFile(testFile, []byte("no id here"), 0644)
	
	w.processFile(testFile, true)
	
	storage.mu.Lock()
	if len(storage.notes) > 0 {
		t.Errorf("Expected 0 notes saved, got %d", len(storage.notes))
	}
	storage.mu.Unlock()
}

func TestWatcherCustomIDDeleteAndConflict(t *testing.T) {
	tempHome := t.TempDir()
	extDir := t.TempDir()
	
	w := &GlobalWatcher{
		stopChan:       make(chan struct{}),
		pending:        make(map[string]*time.Timer),
		fileHashes:     make(map[string]string),
		expectedWrites: make(map[string]ExpectedWrite),
		pathToID:       make(map[string]string),
		idToPath:       make(map[string]string),
		fileMutexes:    make(map[string]*sync.Mutex),
	}
	
	storage := newMockStorage()
	w.Initialize([]string{extDir}, 10, storage, tempHome)
	
	// Create first file with custom ID
	testFile1 := filepath.Join(extDir, "test1.md")
	content := []byte("---\nid: custom-id-123\n---\nbody")
	os.WriteFile(testFile1, content, 0644)
	w.processFile(testFile1, true)
	
	storage.mu.Lock()
	if _, ok := storage.notes["custom-id-123"]; !ok {
		t.Errorf("Expected note to be saved")
	}
	storage.mu.Unlock()
	
	// Create second file with same custom ID -> Conflict
	testFile2 := filepath.Join(extDir, "test2.md")
	os.WriteFile(testFile2, content, 0644)
	w.processFile(testFile2, true)
	
	storage.mu.Lock()
	if _, ok := storage.conflicts["custom-id-123"]; !ok {
		t.Errorf("Expected conflict to be emitted")
	}
	storage.mu.Unlock()
	
	// Delete first file
	w.handleDelete(testFile1)
	
	storage.mu.Lock()
	if !storage.deleted["custom-id-123"] {
		t.Errorf("Expected note custom-id-123 to be deleted")
	}
	storage.mu.Unlock()
}
