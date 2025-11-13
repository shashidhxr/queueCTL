# add a quick job that succeeds
go run . enqueue "echo hello" --db ./test.db
# add a job that fails
go run . enqueue "bash -c 'exit 1'" --db ./test.db
# add a job with timeout (if you added flag)
go run . enqueue "sleep 5" --db ./test.db
