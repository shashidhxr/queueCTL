package store

import (
	"context"
	"time"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

func (s *SQLiteStorage) SetCompleted(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='completed', updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

func (s *SQLiteStorage) FailOrScheduleFixed(ctx context.Context, id string, attempts, maxRetries int, fixedDelay time.Duration, errStr string) error {
	attempts++
	if attempts > maxRetries {
		_, err := s.db.ExecContext(ctx, `
UPDATE jobs SET state='dead', attempts=?, error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, attempts, errStr, id)
		return err
	}
	next := time.Now().UTC().Add(fixedDelay)
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs SET state='pending', attempts=?, next_retry=?, error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		attempts, next, errStr, id)
	return err
}

func (s *SQLiteStorage) FailOrScheduleBackoff(ctx context.Context, j *models.Job, delay time.Duration, errStr string) error {
	j.Attempts++
	if j.Attempts > j.MaxRetries {
		_, err := s.db.ExecContext(ctx, `
UPDATE jobs SET state='dead', attempts=?, error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, j.Attempts, errStr, j.ID)
		return err
	}
	next := time.Now().UTC().Add(delay)
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs SET state='pending', attempts=?, next_retry=?, error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		j.Attempts, next, errStr, j.ID)
	return err
}
