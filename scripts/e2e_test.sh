#!/usr/bin/env bash
set -euo pipefail

DB=./test_e2e.db
rm -f "$DB"
echo "Using DB: $DB"

# 1. start worker in background
go run . worker start --count 1 --poll 200ms --db "$DB" > worker.log 2>&1 &
WPID=$!
sleep 0.5

# 2. enqueue jobs
echo "Enqueue success job"
go run . enqueue "echo ok" --db "$DB"
echo "Enqueue fail job"
go run . enqueue "bash -c 'exit 1'" --db "$DB"

# 3. wait some time for processing and retries
sleep 6

# 4. inspect DB summary
sqlite3 "$DB" "SELECT state, COUNT(*) FROM jobs GROUP BY state;"

# 5. show worker logs and kill
kill "$WPID"
wait "$WPID" 2>/dev/null || true
echo "Worker log:"
tail -n 50 worker.log
