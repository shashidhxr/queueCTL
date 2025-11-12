package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	// "os"
	// "time"
)

type Store interface {
	Init() error
}

type sqliteStore struct {
	db *sql.DB
}

func NewStore(dbPath string) (Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("Failed to create db dir: %w", err)
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode= WAL", dbPath))
	if err != nil {
		return nil; fmt.Errorf("Failed to open db: %w", err)
	}

	db.SetMaxOpenConns(1)		// WAL for best SQLite perf
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute * 5)

	return &sqliteStore{db: db}, nil
}

func (s *sqliteStore) Init() error {
	jobTableSQL := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		state TEXT NOT NULL DEFAULT 'pending',
		attempts INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		available_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_state_available ON jobs (state, available_at);
	`
	
	dlqTableSQL := `
	CREATE TABLE IF NOT EXISTS dlq (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		state TEXT NOT NULL DEFAULT 'dead',
		attempts INTEGER NOT NULL,
		max_retries INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`

	if _, err := s.db.Exec(jobTableSQL); err != nil {
		return fmt.Errorf("Failed to create jobs table: %w", err)
	}

	if _, err := s.db.Exec(dlqTableSQL); err != nil {
		return fmt.Errorf("Failed to create dlq table: %w", err)
	}
	return nil
}