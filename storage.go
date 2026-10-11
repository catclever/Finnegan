package main

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/glebarez/go-sqlite"
)

type RxDBStorage struct {
	db    *sql.DB
	mutex sync.Mutex
}

func NewRxDBStorage(dbPath string) (*RxDBStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS rxdb_docs (
		collection TEXT,
		id TEXT,
		data TEXT,
		PRIMARY KEY(collection, id)
	)`)
	if err != nil {
		return nil, err
	}
	return &RxDBStorage{db: db}, nil
}

func (s *RxDBStorage) GetAllDocs(collection string) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	rows, err := s.db.Query("SELECT data FROM rxdb_docs WHERE collection = ?", collection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		docs = append(docs, data)
	}
	return docs, nil
}

func (s *RxDBStorage) GetDoc(collection, id string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var data string
	err := s.db.QueryRow("SELECT data FROM rxdb_docs WHERE collection = ? AND id = ?", collection, id).Scan(&data)
	if err != nil {
		return "", err
	}
	return data, nil
}

type DocSaveRequest struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

func (s *RxDBStorage) BulkSaveDocs(collection string, docs []DocSaveRequest) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO rxdb_docs (collection, id, data) VALUES (?, ?, ?) ON CONFLICT(collection, id) DO UPDATE SET data = excluded.data")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, req := range docs {
		_, err := stmt.Exec(collection, req.ID, req.Data)
		if err != nil {
			log.Printf("Failed to save doc %s: %v", req.ID, err)
		}
	}
	
	return tx.Commit()
}

func (s *RxDBStorage) DeleteDoc(collection, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, err := s.db.Exec("DELETE FROM rxdb_docs WHERE collection = ? AND id = ?", collection, id)
	return err
}

func (s *RxDBStorage) RemoveCollection(collection string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, err := s.db.Exec("DELETE FROM rxdb_docs WHERE collection = ?", collection)
	return err
}
