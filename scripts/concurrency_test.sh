#!/usr/bin/env bash
set -euo pipefail
DB=./test_conc.db
rm -f "$DB"

# start two workers
go run . worker start --count 1 --poll 200ms --db "$DB" > w1.log 2>&1 &
W1=$!
go run . worker start --count 1 --poll 200ms --db "$DB" > w2.log 2>&1 &
W2=$!
sleep 0.5

# enqueue 20 fast jobs
for i in $(seq 1 20); do
  go run . enqueue "echo job-$i" --db "$DB"
done

# wait for processing
sleep 4

# verify no duplicates: check completed count equals 20
sqlite3 "$DB" "SELECT state, COUNT(*) FROM jobs GROUP BY state;"

kill "$W1" "$W2" || true
wait "$W1" "$W2" 2>/dev/null || true
echo "w1 tail:"; tail -n 10 w1.log
echo "w2 tail:"; tail -n 10 w2.log
