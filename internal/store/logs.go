package store

import (
	"context"
	"time"
)

type LogLine struct {
	TS     time.Time
	Stream string
	Chunk  string
}

func (s *SQLiteStorage) AppendLog(ctx context.Context, jobID, stream, chunk string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO job_logs(job_id, ts, stream, chunk) VALUES(?, CURRENT_TIMESTAMP, ?, ?)`, jobID, stream, chunk)
	return err
}

func (s *SQLiteStorage) GetLogs(ctx context.Context, jobID string, limit int) ([]LogLine, error) {
	if limit <= 0 { limit = 200 }
	rows, err := s.db.QueryContext(ctx, `
SELECT ts, stream, chunk FROM job_logs WHERE job_id = ? ORDER BY ts ASC LIMIT ?`, jobID, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []LogLine
	for rows.Next() {
		var l LogLine
		if err := rows.Scan(&l.TS, &l.Stream, &l.Chunk); err != nil { return nil, err }
		out = append(out, l)
	}
	return out, rows.Err()
}
