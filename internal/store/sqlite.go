package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
	 return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
	 return nil, err
	}
	s := &SQLiteStorage{db: db}
	if err := s.runMigrations(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStorage) Close() error { return s.db.Close() }
