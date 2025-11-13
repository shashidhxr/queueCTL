package store

import (
	"context"
	"database/sql"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

func (s *SQLiteStorage) ListJobs(ctx context.Context, state models.JobState, limit int) ([]models.Job, error) {
	if limit <= 0 { limit = 50 }
	rows, err := s.db.QueryContext(ctx, `
SELECT id, command, state, attempts, max_retries, error, next_retry, created_at, updated_at
FROM jobs
WHERE (? = '' OR state = ?)
ORDER BY created_at DESC
LIMIT ?`, state, state, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []models.Job
	for rows.Next() {
		var j models.Job
		var next sql.NullTime
		if err := rows.Scan(&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries, &j.Error, &next, &j.CreatedAt, &j.UpdatedAt); err != nil {
		 return nil, err
		}
		if next.Valid { t := next.Time; j.NextRetry = &t }
		out = append(out, j)
	}
	return out, rows.Err()
}
