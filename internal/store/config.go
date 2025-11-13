package store

import (
	"context"
	"strconv"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

func (s *SQLiteStorage) GetConfig(ctx context.Context) (*models.Config, error) {
	cfg := &models.Config{ MaxRetries: 3, BackoffBase: 2 }
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

func (s *SQLiteStorage) SetConfig(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO config(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
