package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

type SQLiteStorage struct {
	db *sql.DB
}	

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &SQLiteStorage{db: db}
	if err := s.runMigrations(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStorage) Close() error { return s.db.Close() }

func (s *SQLiteStorage) runMigrations() error {
	schema := `
CREATE TABLE IF NOT EXISTS jobs (
  id           TEXT PRIMARY KEY,
  command      TEXT NOT NULL,
  state        TEXT NOT NULL,          -- pending|processing|completed|failed|dead
  attempts     INTEGER NOT NULL DEFAULT 0,
  max_retries  INTEGER NOT NULL DEFAULT 3,
  error        TEXT,
  next_retry   TIMESTAMP,
  created_at   TIMESTAMP NOT NULL,
  updated_at   TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_jobs_state_next ON jobs(state, next_retry, created_at);
`
	_, err := s.db.Exec(schema)
	return err
}

// func (s *SQLiteStorage) SaveJob(ctx context.Context, j *models.Job) error {
// 	now := time.Now().UTC()
// 	if j.State == "" {
// 		j.State = string(models.StatePending)
// 	}
// 	if j.MaxRetries == 0 {
// 		j.MaxRetries = 3
// 	}
// 	j.CreatedAt = now
// 	j.UpdatedAt = now

// 	_, err := s.db.ExecContext(ctx, `
// INSERT INTO jobs (id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at)
// VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
// 		j.ID, j.Command, j.State, j.Attempts, j.MaxRetries, j.Error, nullableTime(j.NextRetry), j.CreatedAt, j.UpdatedAt)
// 	return err
// }


func (s *SQLiteStorage) AcquireJob(ctx context.Context) (*models.Job, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, command, state, attempts, max_retries,
				created_at, updated_at, error, next_retry
		FROM jobs
		WHERE state = ? AND (next_retry IS NULL OR next_retry <= ?)
		ORDER BY created ASC
		LIMIT 1
	`
	
	var job models.Job
	err = tx.QueryRowContext(ctx, query, models.StatePending, time.Now()).Scan(
		&job.ID,
		&job.Command,
		&job.State, 
		&job.Attempts,
		&job.MaxRetries,
		&job,job.CreatedAt,
		&job.UpdatedAt,
		&job.Error,
		&job.NextRetry,
	)
	if err != nil {
		return nil, err
	}

	updateQuery := "UPDATE jobs SET state = ?, updated_at = ? WHERE id = ?"
	_, err = tx.ExecContext(ctx, updateQuery, models.StateProcessing, time.Now(), job.ID)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	job.State = models.StateProcessing
	return &job, nil
}