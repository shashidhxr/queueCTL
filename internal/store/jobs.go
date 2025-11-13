package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shashidhxr/queueCTL/pkg/models"
)

func (s *SQLiteStorage) SaveJob(ctx context.Context, j *models.Job) error {
	now := time.Now().UTC()
	if j.State == "" {
		j.State = models.StatePending
	}
	if j.MaxRetries == 0 {
		j.MaxRetries = 3
	}
	j.CreatedAt, j.UpdatedAt = now, now
	_, err := s.db.ExecContext(ctx, `
INSERT INTO jobs (id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at, timeout_seconds)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.Command, j.State, j.Attempts, j.MaxRetries, j.Error, nullableTime(j.NextRetry), j.CreatedAt, j.UpdatedAt, j.TimeoutSeconds)
	return err
}

func (s *SQLiteStorage) GetJob(ctx context.Context, id string) (*models.Job, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at, timeout_seconds, worker_id, lease_until
FROM jobs WHERE id = ?`, id)
	var j models.Job
	var next, lease sql.NullTime
	var workerID sql.NullString
	if err := row.Scan(&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries, &j.Error, &next, &j.CreatedAt, &j.UpdatedAt, &j.TimeoutSeconds, &workerID, &lease); err != nil {
		return nil, err
	}
	if next.Valid { t := next.Time; j.NextRetry = &t }
	if lease.Valid { t := lease.Time; j.LeaseUntil = &t }
	if workerID.Valid { v := workerID.String; j.WorkerID = &v }
	return &j, nil
}

func (s *SQLiteStorage) UpdateJobState(ctx context.Context, id string, state models.JobState) error {
	_, err := s.db.ExecContext(ctx, `UPDATE jobs SET state=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, state, id)
	return err
}


func (s *SQLiteStorage) AcquireJob(ctx context.Context) (*models.Job, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil { return nil, err }
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
SELECT id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at, timeout_seconds
FROM jobs
WHERE state='pending' AND (next_retry IS NULL OR next_retry <= CURRENT_TIMESTAMP)
ORDER BY created_at ASC LIMIT 1`)
	var j models.Job
	if err := row.Scan(&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries, &j.Error, &j.NextRetry, &j.CreatedAt, &j.UpdatedAt, &j.TimeoutSeconds); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return nil, nil }
		return nil, err
	}

	workerID := uuid.NewString()
	leaseUntil := time.Now().UTC().Add(time.Duration(j.TimeoutSeconds+10) * time.Second)

	res, err := tx.ExecContext(ctx, `
UPDATE jobs
SET state='processing', worker_id=?, lease_until=?, updated_at=CURRENT_TIMESTAMP
WHERE id=? AND state='pending'`, workerID, leaseUntil, j.ID)
	if err != nil { return nil, err }
	if n, _ := res.RowsAffected(); n != 1 { return nil, nil }

	if err := tx.Commit(); err != nil { return nil, err }
	j.State, j.UpdatedAt = models.StateProcessing, time.Now().UTC()
	j.WorkerID, j.LeaseUntil = &workerID, &leaseUntil
	return &j, nil
}

func (s *SQLiteStorage) RequeueExpiredLeases(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET state='pending', worker_id=NULL, lease_until=NULL, updated_at=CURRENT_TIMESTAMP
WHERE state='processing' AND lease_until IS NOT NULL AND lease_until <= CURRENT_TIMESTAMP`)
	return err
}