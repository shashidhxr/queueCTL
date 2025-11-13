package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/shashidhxr/queueCTL/internal/core"
	"github.com/shashidhxr/queueCTL/pkg/models"
	"github.com/spf13/cobra"
)

var (
	workerCmd = &cobra.Command{
		Use:   "worker",
		Short: "Worker utilities",
	}

	workerStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start worker loop",
		RunE:  runWorkers,
	}

	workerCount    int
	backoffBase    float64
	pollInterval   time.Duration
	printNoJobLogs bool
)

func init() {
	workerCmd.AddCommand(workerStartCmd)
	rootCmd.AddCommand(workerCmd)

	workerStartCmd.Flags().IntVar(&workerCount, "count", 1, "Number of worker goroutines")
	workerStartCmd.Flags().Float64Var(&backoffBase, "backoff-base", 2.0, "Exponential backoff base (e.g., 2 => 2^attempts seconds)")
	workerStartCmd.Flags().DurationVar(&pollInterval, "poll", 300*time.Millisecond, "Polling interval for new jobs")
	workerStartCmd.Flags().BoolVar(&printNoJobLogs, "quiet-empty", true, "Suppress logs when no job is available")
}

func runWorkers(cmd *cobra.Command, args []string) error {
	if workerCount < 1 {
		workerCount = 1
	}
	if pollInterval <= 0 {
		pollInterval = 300 * time.Millisecond
	}
	if backoffBase <= 0 {
		backoffBase = 2.0
	}

	// Graceful shutdown on Ctrl+C / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rm := core.NewRetryManager(*st) // uses default base internally

	fmt.Printf("Worker started: %d goroutine(s). Poll=%s, Backoff base=%.2f. Ctrl+C to stop…\n",
		workerCount, pollInterval, backoffBase)

	// Reaper: periodically requeue expired leases
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := st.RequeueExpiredLeases(context.Background()); err != nil {
					fmt.Println("reaper:", err)
				}
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func(id int) {
			defer wg.Done()
			workerLoop(ctx, id, rm)
		}(i + 1)
	}

	<-ctx.Done()
	fmt.Println("\nShutting down workers gracefully…")
	wg.Wait()
	return nil
}

func workerLoop(ctx context.Context, wid int, rm *core.RetryManager) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// 1) Try to atomically claim a job
			job, err := st.AcquireJob(ctx)
			if err != nil {
				fmt.Printf("[worker %d] acquire error: %v\n", wid, err)
				continue
			}
			if job == nil {
				if !printNoJobLogs {
					fmt.Printf("[worker %d] no job available\n", wid)
				}
				continue
			}

			// 2) Execute the job command with timeout
			effectiveTimeout := time.Duration(job.TimeoutSeconds) * time.Second
			if effectiveTimeout <= 0 {
				effectiveTimeout = 30 * time.Second // default
			}

			fmt.Printf("[worker %d] processing: %s (id=%s, timeout=%s)\n", wid, job.Command, job.ID, effectiveTimeout)

			execCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
			out, execErr := exec.CommandContext(execCtx, "sh", "-c", job.Command).CombinedOutput()
			cancel()

			// record logs
			if len(out) > 0 {
				fmt.Print(string(out))
				_ = st.AppendLog(ctx, job.ID, "combined", string(out))
			}
			if execErr != nil && execCtx.Err() == context.DeadlineExceeded {
				execErr = fmt.Errorf("timeout after %s", effectiveTimeout)
			}

			// 3) Update state based on result
			if execErr == nil {
				if err := st.SetCompleted(ctx, job.ID); err != nil {
					fmt.Printf("[worker %d] update completed error: %v\n", wid, err)
				} else {
					fmt.Printf("[worker %d] completed: id=%s\n", wid, job.ID)
				}
				continue
			}

			// failure path → exponential backoff (delay = base^attempts seconds)
			delay := calculateBackoffWithBase(backoffBase, job.Attempts+1, rm)
			if err := st.FailOrScheduleBackoff(ctx, job, delay, execErr.Error()); err != nil {
				fmt.Printf("[worker %d] update retry/dead error: %v\n", wid, err)
				continue
			}
			if job.Attempts+1 > job.MaxRetries {
				fmt.Printf("[worker %d] job moved to DLQ (dead): id=%s err=%v\n", wid, job.ID, execErr)
			} else {
				fmt.Printf("[worker %d] failed (attempt %d/%d). next in %s. id=%s err=%v\n",
					wid, job.Attempts+1, job.MaxRetries, delay, job.ID, execErr)
			}
		}
	}
}

func calculateBackoffWithBase(base float64, attempts int, rm *core.RetryManager) time.Duration {
	if base <= 0 {
		return rm.CalculateBackoff(attempts)
	}
	// inline calculation with cap at 5 minutes
	seconds := 1.0
	for i := 0; i < attempts; i++ {
		seconds *= base
	}
	if seconds > 300.0 {
		seconds = 300.0
	}
	return time.Duration(seconds) * time.Second
}

// Keep models import considered used if needed
var _ = models.Job{}
