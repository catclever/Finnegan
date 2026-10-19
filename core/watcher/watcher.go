package watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/markdown"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Storage interface {
	SaveNote(id, data string) error
	DeleteNote(id string) error
	GetNote(id string) (string, error)
	EmitConflict(id string, paths []string)
}

type ExpectedWrite struct {
	Hash   string
	Origin string
}

type GlobalWatcher struct {
	watcher        *fsnotify.Watcher
	debounceMs     int
	storage        Storage
	stopChan       chan struct{}
	stopped        bool
	
	pending      map[string]*time.Timer
	pendingMutex sync.Mutex
	
	fileHashes     map[string]string
	expectedWrites map[string]ExpectedWrite
	pathToID       map[string]string
	idToPath       map[string]string
	globalMutex    sync.Mutex
	
	fileMutexes    map[string]*sync.Mutex
	fileMutexesMut sync.Mutex

	homeDir string
}

var instance *GlobalWatcher
var once sync.Once

func GetWatcher() *GlobalWatcher {
	once.Do(func() {
		instance = &GlobalWatcher{
			stopChan:       make(chan struct{}),
			pending:        make(map[string]*time.Timer),
			fileHashes:     make(map[string]string),
			expectedWrites: make(map[string]ExpectedWrite),
			pathToID:       make(map[string]string),
			idToPath:       make(map[string]string),
			fileMutexes:    make(map[string]*sync.Mutex),
		}
	})
	return instance
}

func (w *GlobalWatcher) getFileMutex(filePath string) *sync.Mutex {
	w.fileMutexesMut.Lock()
	defer w.fileMutexesMut.Unlock()
	if m, ok := w.fileMutexes[filePath]; ok {
		return m
	}
	m := &sync.Mutex{}
	w.fileMutexes[filePath] = m
	return m
}

func (w *GlobalWatcher) Initialize(dirs []string, debounceMs int, storage Storage, homeDir string) error {
	w.debounceMs = debounceMs
	w.storage = storage
	w.homeDir = homeDir

	if w.debounceMs <= 0 {
		w.debounceMs = 300
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = watcher

	for _, dir := range dirs {
		w.addDirRecursive(dir, true)
	}

	go w.watchRoutine()
	return nil
}

func (w *GlobalWatcher) addDirRecursive(dir string, isStartup bool) {
	if filepath.Base(dir) == ".finnegan" {
		return // never watch .finnegan
	}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		
		if d.IsDir() {
			if name == ".finnegan" {
				return fs.SkipDir
			}
			w.watcher.Add(path)
		} else if isMarkdown(name) {
			if isStartup {
				w.processFile(path, true)
			}
		}
		return nil
	})
}

func isMarkdown(name string) bool {
	if !strings.HasSuffix(name, ".md") {
		return false
	}
	if strings.HasSuffix(name, ".swp") || strings.HasSuffix(name, "~") || strings.HasSuffix(name, ".tmp") {
		return false
	}
	if strings.HasPrefix(filepath.Base(name), ".") {
		return false
	}
	return true
}

func (w *GlobalWatcher) Close() {
	if w.watcher != nil {
		w.watcher.Close()
	}
	w.globalMutex.Lock(); if !w.stopped { close(w.stopChan); w.stopped = true }; w.globalMutex.Unlock()
}

func (w *GlobalWatcher) watchRoutine() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			
			if event.Op&fsnotify.Create == fsnotify.Create {
				if stat, err := os.Stat(event.Name); err == nil && stat.IsDir() {
					w.addDirRecursive(event.Name, false)
					continue
				}
			}
			
			if isMarkdown(event.Name) {
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
					w.debounceDelete(event.Name)
				} else if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
					w.debounceProcess(event.Name)
				}
			}
			
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Println("watcher error:", err)
		case <-w.stopChan:
			return
		}
	}
}

func (w *GlobalWatcher) debounceProcess(filePath string) {
	w.pendingMutex.Lock()
	defer w.pendingMutex.Unlock()

	if timer, ok := w.pending[filePath]; ok {
		timer.Stop()
	}

	w.pending[filePath] = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
		w.pendingMutex.Lock()
		delete(w.pending, filePath)
		w.pendingMutex.Unlock()
		w.processFile(filePath, false)
	})
}

func (w *GlobalWatcher) debounceDelete(filePath string) {
	w.pendingMutex.Lock()
	defer w.pendingMutex.Unlock()

	if timer, ok := w.pending[filePath]; ok {
		timer.Stop()
	}
	
	w.pending[filePath] = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
		w.pendingMutex.Lock()
		delete(w.pending, filePath)
		w.pendingMutex.Unlock()
		
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			w.handleDelete(filePath)
		} else {
			w.processFile(filePath, false)
		}
	})
}

type CardData struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Meta      map[string]any `json:"meta"`
	FilePath  string         `json:"filePath"`
}

func (w *GlobalWatcher) processFile(filePath string, isStartup bool) {
	mutex := w.getFileMutex(filePath)
	mutex.Lock()
	defer mutex.Unlock()

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 
		}
		log.Printf("Failed to read file %s: %v", filePath, err)
		return
	}

	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	w.globalMutex.Lock()
	lastHash := w.fileHashes[filePath]
	expected, hasExpected := w.expectedWrites[filePath]
	origin := "external"
	if hasExpected {
		if expected.Hash == hash {
			origin = expected.Origin
		}
		delete(w.expectedWrites, filePath)
	}
	w.globalMutex.Unlock()

	if lastHash == hash {
		return // dedupe
	}

	abs, _ := filepath.Abs(filePath)
	docID := ""
	
	strContent := string(content)
	if strings.HasPrefix(strContent, "---\n") || strings.HasPrefix(strContent, "---\r\n") {
		parts := strings.SplitN(strContent, "---", 3)
		if len(parts) >= 3 {
			var meta map[string]any
			if err := yaml.Unmarshal([]byte(parts[1]), &meta); err == nil && meta != nil {
				if id, ok := meta["id"].(string); ok && id != "" {
					docID = id
				}
			}
		}
	}

	// External project check: must have ID
	isInternal := strings.HasPrefix(abs, filepath.Join(w.homeDir, "Scattering")) || strings.HasPrefix(abs, filepath.Join(w.homeDir, "Weaving"))
	if !isInternal && docID == "" {
		return // Ignore md without ID in external projects
	}

	if docID == "" {
		docID = markdown.FallbackID(abs)
	}

	w.globalMutex.Lock()
	if existingPath, ok := w.idToPath[docID]; ok && existingPath != filePath {
		log.Printf("Warning: Duplicate id %s found in %s (existing in %s)", docID, filePath, existingPath)
		w.globalMutex.Unlock()
		if w.storage != nil {
			w.storage.EmitConflict(docID, []string{existingPath, filePath})
		}
		return // Never silently overwrite
	}
	w.pathToID[filePath] = docID
	w.idToPath[docID] = filePath
	w.globalMutex.Unlock()

	var existingCreatedAt time.Time
	var oldMeta map[string]any

	if w.storage != nil {
		if noteStr, err := w.storage.GetNote(docID); err == nil && noteStr != "" {
			var oldCard CardData
			if err := json.Unmarshal([]byte(noteStr), &oldCard); err == nil {
				existingCreatedAt = oldCard.CreatedAt
				oldMeta = oldCard.Meta
			}
		}
	}

	doc, err := markdown.Parse(filePath, content, existingCreatedAt)
	if err != nil {
		log.Printf("Failed to parse markdown %s: %v", filePath, err)
		return
	}

	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	
	card := CardData{
		ID:        doc.ID,
		Type:      "local",
		Title:     doc.Title,
		Content:   doc.Body,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: time.Now(),
		Meta:      doc.Meta,
		FilePath:  filePath,
	}

	dataBytes, err := json.Marshal(card)
	if err != nil {
		log.Printf("Failed to marshal card data: %v", err)
		return
	}

	if w.storage != nil {
		err = w.storage.SaveNote(doc.ID, string(dataBytes))
		if err != nil {
			log.Printf("Failed to save note to db: %v", err)
		}
	}

	w.globalMutex.Lock()
	w.fileHashes[filePath] = hash
	w.globalMutex.Unlock()

	if !isStartup {
		eventType := bus.EventFileModified
		if lastHash == "" {
			eventType = bus.EventFileCreated
		}
		
		bus.GetBus().Publish(bus.Event{
			Type: eventType,
			Data: bus.EventFileData{
				FilePath: filePath,
				DocID:    doc.ID,
				NewMeta:  doc.Meta,
				OldMeta:  oldMeta,
				Body:     doc.Body,
				Origin:   origin,
			},
		})
	}
}

func (w *GlobalWatcher) handleDelete(filePath string) {
	w.globalMutex.Lock()
	delete(w.fileHashes, filePath)
	delete(w.expectedWrites, filePath)
	
	docID := w.pathToID[filePath]
	if docID != "" {
		delete(w.idToPath, docID)
		delete(w.pathToID, filePath)
	}
	w.globalMutex.Unlock()

	abs, _ := filepath.Abs(filePath)
	if docID == "" {
		docID = markdown.FallbackID(abs)
	}

	if w.storage != nil {
		if noteStr, err := w.storage.GetNote(docID); err == nil && noteStr != "" {
			var oldCard CardData
			if err := json.Unmarshal([]byte(noteStr), &oldCard); err == nil {
				docID = oldCard.ID
			}
		}
		
		err := w.storage.DeleteNote(docID)
		if err != nil {
			log.Printf("Failed to delete note from db: %v", err)
		}
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventFileDeleted,
		Data: bus.EventFileData{
			FilePath: filePath,
			DocID:    docID,
			Origin:   "external",
		},
	})
}

func (w *GlobalWatcher) WriteFile(filePath string, content []byte, origin string) error {
	mutex := w.getFileMutex(filePath)
	mutex.Lock()
	defer mutex.Unlock()

	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	w.globalMutex.Lock()
	w.expectedWrites[filePath] = ExpectedWrite{Hash: hash, Origin: origin}
	w.globalMutex.Unlock()

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, ".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	if err := os.Rename(tmpName, filePath); err != nil {
		return err
	}
	
	return nil
}

// GetPathForID returns the mapped file path for a docID, or empty string if not found.
func (w *GlobalWatcher) GetPathForID(id string) string {
	w.globalMutex.Lock()
	defer w.globalMutex.Unlock()
	return w.idToPath[id]
}
