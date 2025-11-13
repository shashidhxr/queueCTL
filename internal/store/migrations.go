package store

func (s *SQLiteStorage) runMigrations() error {
	schema := `
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  command TEXT NOT NULL,
  state TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  max_retries INTEGER NOT NULL DEFAULT 3,
  error TEXT,
  next_retry TIMESTAMP,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  timeout_seconds INTEGER NOT NULL DEFAULT 30,
  worker_id TEXT,
  lease_until TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_jobs_state_next ON jobs(state, next_retry, created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_lease ON jobs(state, lease_until);

CREATE TABLE IF NOT EXISTS config (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS job_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL,
  ts TIMESTAMP NOT NULL,
  stream TEXT NOT NULL,
  chunk TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_job_logs_job_ts ON job_logs(job_id, ts);
`
	_, err := s.db.Exec(schema)
	return err
}
