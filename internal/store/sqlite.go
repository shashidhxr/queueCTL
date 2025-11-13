package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// "errors"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shashidhxr/queueCTL/pkg/models"
)

type SQLiteStorage struct {
	db *sql.DB
}	

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	
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

		CREATE TABLE IF NOT EXISTS config (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStorage) SaveJob(ctx context.Context, j *models.Job) error {
	now := time.Now().UTC()
	if j.State == "" {
		j.State = models.StatePending
	}
	if j.MaxRetries == 0 {
		j.MaxRetries = 3
	}
	j.CreatedAt = now
	j.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
INSERT INTO jobs (id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.Command, j.State, j.Attempts, j.MaxRetries, j.Error, nullableTime(j.NextRetry), j.CreatedAt, j.UpdatedAt)
	return err
}

func (s *SQLiteStorage) GetJob(ctx context.Context, id string) (*models.Job, error) {
    row := s.db.QueryRowContext(ctx, `
SELECT id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at
FROM jobs WHERE id = ?`, id)

    var j models.Job
    var next sql.NullTime
    if err := row.Scan(&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries, &j.Error, &next, &j.CreatedAt, &j.UpdatedAt); err != nil {
        return nil, err
    }
    if next.Valid { t := next.Time; j.NextRetry = &t }
    return &j, nil
}


func (s *SQLiteStorage) UpdateJobState(ctx context.Context, id string, state models.JobState) error {
    _, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET state = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`, state, id)
    return err
}


func (s *SQLiteStorage) ListJobs(ctx context.Context, state models.JobState) ([]*models.Job, error) {
    rows, err := s.db.QueryContext(ctx, `
SELECT id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at
FROM jobs
WHERE state = ?
ORDER BY created_at ASC`, state)
    if err != nil { return nil, err }
    defer rows.Close()

    var out []*models.Job
    for rows.Next() {
        var j models.Job
        var next sql.NullTime
        if err := rows.Scan(&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries, &j.Error, &next, &j.CreatedAt, &j.UpdatedAt); err != nil {
            return nil, err
        }
        if next.Valid { t := next.Time; j.NextRetry = &t }
        out = append(out, &j)
    }
    return out, rows.Err()
}

func (s *SQLiteStorage) GetPendingJobs(ctx context.Context) ([]*models.Job, error) {
    return s.ListJobs(ctx, models.StatePending)
}

func (s *SQLiteStorage) AcquireJob(ctx context.Context) (*models.Job, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// 1) Pick next runnable pending job
	row := tx.QueryRowContext(ctx, `
SELECT id, command, state, attempts, max_retries,
       created_at, updated_at, error, next_retry
FROM jobs
WHERE state = ? AND (next_retry IS NULL OR next_retry <= ?)
ORDER BY created_at ASC
LIMIT 1`,
		models.StatePending, time.Now().UTC(),
	)

	var j models.Job
	var next sql.NullTime
	if err := row.Scan(
		&j.ID,
		&j.Command,
		&j.State,
		&j.Attempts,
		&j.MaxRetries,
		&j.CreatedAt,
		&j.UpdatedAt,
		&j.Error,
		&next,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // nothing to claim
		}
		return nil, err
	}
	if next.Valid {
		t := next.Time
		j.NextRetry = &t
	}

	// 2) Atomically claim it
	res, err := tx.ExecContext(ctx, `
UPDATE jobs
SET state='processing', updated_at=CURRENT_TIMESTAMP
WHERE id = ? AND state='pending'`, j.ID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		// lost race to another worker
		return nil, nil
	}

	// 3) Commit and return the claimed job
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	j.State = models.StateProcessing
	j.UpdatedAt = time.Now().UTC()
	return &j, nil
}


func (s *SQLiteStorage) GetDLQJobs(ctx context.Context) ([]*models.Job, error) {
    return s.ListJobs(ctx, models.StateDead)
}

func (s *SQLiteStorage) RetryDLQJob(ctx context.Context, jobID string) error {
    _, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET state='pending', attempts=0, error=NULL, next_retry=NULL, updated_at=CURRENT_TIMESTAMP
WHERE id = ? AND state='dead'`, jobID)
    return err
}

func (s *SQLiteStorage) GetConfig(ctx context.Context) (*models.Config, error) {
    cfg := &models.Config{ MaxRetries: 3, BackoffBase: 2 } // defaults

    // try override if present
    rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM config`)
    if err != nil { return nil, err }
    defer rows.Close()

    for rows.Next() {
        var k, v string
        if err := rows.Scan(&k, &v); err != nil { return nil, err }
        switch k {
        case "max_retries":
            if n, err := strconv.Atoi(v); err == nil { cfg.MaxRetries = n }
        case "backoff_base":
            if n, err := strconv.Atoi(v); err == nil { cfg.BackoffBase = n }
        }
    }
    return cfg, rows.Err()
}

func (s *SQLiteStorage) SetConfig(ctx context.Context, cfg *models.Config) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer func() { _ = tx.Rollback() }()

    if _, err := tx.ExecContext(ctx, `INSERT INTO config(key,value) VALUES('max_retries',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, strconv.Itoa(cfg.MaxRetries)); err != nil {
        return err
    }
    if _, err := tx.ExecContext(ctx, `INSERT INTO config(key,value) VALUES('backoff_base',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, strconv.Itoa(cfg.BackoffBase)); err != nil {
        return err
    }
    return tx.Commit()
}

func (s *SQLiteStorage) GetJobStats(ctx context.Context) (map[models.JobState]int, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT state, COUNT(*) FROM jobs GROUP BY state`)
    if err != nil { return nil, err }
    defer rows.Close()

    out := map[models.JobState]int{
        models.StatePending: 0,
        models.StateProcessing: 0,
        models.StateCompleted: 0,
        models.StateFailed: 0,
        models.StateDead: 0,
    }
    for rows.Next() {
        var st string; var n int
        if err := rows.Scan(&st, &n); err != nil { return nil, err }
        out[models.JobState(st)] = n
    }
    return out, rows.Err()
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// helpers

func (s *SQLiteStorage) SetCompleted(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='completed', updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

func (s *SQLiteStorage) SetFailedOrRetry(ctx context.Context, j *models.Job, backoffBase int) error {
	j.Attempts++
	if j.Attempts > j.MaxRetries {
		_, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='dead', attempts=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, j.Attempts, j.ID)
		return err
	}
	// delay = base^attempts seconds
	delay := time.Duration(1)
	for i := 0; i < j.Attempts; i++ { delay *= time.Duration(backoffBase) }
	next := time.Now().UTC().Add(delay * time.Second)

	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET state='pending', attempts=?, next_retry=?, error=NULL, updated_at=CURRENT_TIMESTAMP
WHERE id=?`, j.Attempts, next, j.ID)
	return err
}