
# QueueCTL

QueueCTL is a command-line background job queue system written in Go.  
It provides a simple way to enqueue shell commands for asynchronous execution,  
with persistence, retry handling with exponential backoff, and a dead letter queue (DLQ).

---

## Overview

QueueCTL consists of three main components:

1. **CLI Interface** – provides commands to enqueue jobs, start workers, and manage configuration.  
2. **Worker System** – executes jobs concurrently, manages state transitions, and handles retries.  
3. **Storage Layer** – persists jobs and their metadata using SQLite for durability and atomic operations.

This system demonstrates the design of a minimal, production-style background job queue.

---

## Features

- Persistent job storage using SQLite
- Multiple worker support
- Automatic retry with exponential backoff (`delay = base^attempts` seconds)
- Dead Letter Queue (DLQ) for permanently failed jobs
- Configurable backoff base and polling interval
- Graceful worker shutdown

---

## Architecture

```
+------------+         +----------------+         +---------------+
|  queuectl  |  --->   |   SQLite DB    |  --->   |    Worker(s)  |
|   CLI      |         |  (Persistent)  |         |  Executes cmd |
+------------+         +----------------+         +---------------+

Job Lifecycle:
 pending → processing → completed
               ↘
              failed → pending (backoff) → dead (DLQ)
```

**Storage**  
Implements an atomic queue using SQLite transactions to safely claim and update jobs.

**Worker**  
Continuously polls the queue, executes pending jobs, and updates state based on execution results.

**Retry Manager**  
Computes backoff delays using an exponential formula and schedules next retry times.

---

## Installation

### Prerequisites
- Go 1.22+
- SQLite (installed automatically via Go driver)

### Build
```bash
git clone https://github.com/shashidhxr/queueCTL.git
cd queueCTL
go build -o queuectl
```

### Run Without Build
```bash
go run . <command>
```

---

## Usage

### Enqueue a Job
```bash
go run . enqueue "echo 'Hello World'"
```
Adds a new job to the SQLite queue (default path: `~/.queuectl/queue.db`).

### Start Worker(s)
```bash
go run . worker start --count 2
```
Starts two concurrent workers that process jobs.

### Example Output
```
Worker started: 2 goroutine(s). Poll=300ms, Backoff base=2.00.
[worker 1] processing: echo 'Hello World'
Hello World
[worker 1] completed: id=...
```

### Test Retry Behavior
```bash
go run . enqueue "bash -c 'exit 1'"
```
Expected behavior:
- Retries with increasing delays (`2s`, `4s`, `8s`, …)
- Moves to DLQ (`state='dead'`) after exceeding retry limit

---

## Configuration

Available flags:

| Flag | Description | Default |
|------|--------------|----------|
| `--count` | Number of worker goroutines | `1` |
| `--poll` | Polling interval for new jobs | `300ms` |
| `--backoff-base` | Base for exponential backoff | `2.0` |
| `--db` | Path to SQLite database | `~/.queuectl/queue.db` |

Example:
```bash
go run . worker start --count 4 --backoff-base 3.0 --poll 500ms
```

---

## Inspecting Jobs

To inspect the current queue:
```bash
sqlite3 ~/.queuectl/queue.db "SELECT id, state, attempts, datetime(next_retry), error FROM jobs ORDER BY created_at;"
```

Typical states:
- `pending` – waiting to be executed  
- `processing` – currently being executed  
- `completed` – executed successfully  
- `failed` – temporary failure (scheduled for retry)  
- `dead` – permanently failed (in DLQ)

---

## Project Structure

```
queueCTL/
├── cmd/                # CLI commands (enqueue, worker )
├── internal/core/      # Retry logic and core utilities
├── internal/store/     # SQLite persistence layer
├── pkg/models/         # Job and config models
└── main.go             # Entry point
```

---

## Testing the System

1. Start a worker process:
   ```bash
   go run . worker start
   ```

2. Enqueue multiple jobs:
   ```bash
   go run . enqueue "echo Job 1"
   go run . enqueue "bash -c 'exit 1'"
   ```

3. Observe job transitions:
   - Success: `pending → processing → completed`
   - Failure: `pending → processing → failed → pending (backoff) → dead(DLQ)`

4. Restart the worker to verify job persistence across runs.

---

## Future Improvements

- Job execution timeout and cancellation
- Job priority queues
- Scheduled and delayed jobs
- Job output logging
- Prometheus metrics and monitoring
- Minimal web dashboard for job inspection
- Worker lease and visibility timeout handling

---

## License

MIT License © 2025 Shashidhxr
