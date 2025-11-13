package store

import (
	"context"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

func (s *SQLiteStorage) GetJobStats(ctx context.Context) (map[models.JobState]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT state, COUNT(*) FROM jobs GROUP BY state`)
	if err != nil { return nil, err }
	defer rows.Close()

	out := map[models.JobState]int{
		models.StatePending: 0, models.StateProcessing: 0,
		models.StateCompleted: 0, models.StateFailed: 0, models.StateDead: 0,
	}
	for rows.Next() {
		var st string; var n int
		if err := rows.Scan(&st, &n); err != nil { return nil, err }
		out[models.JobState(st)] = n
	}
	return out, rows.Err()
}
