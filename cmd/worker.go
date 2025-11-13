package cmd

import (
	"context"
	"fmt"
	"sync"

	// "os/exec"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/shashidhxr/queueCTL/internal/core"
	"github.com/shashidhxr/queueCTL/pkg/models"
	"github.com/spf13/cobra"
)

var (

	workerCmd = &cobra.Command{
		Use: "worker",
		Short: "Worker utilities",
	}
	
	workerStartCmd = &cobra.Command{
		Use: "start",
		Short: "Start worker loop",
		RunE: runWorkers,
	}
	
	workerCount    int
	backoffBase    float64
	pollInterval   time.Duration
	printNoJobLogs bool
)

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

	rm := core.NewRetryManager(*st)
	// override default base if user set a flag
	if backoffBase > 0 {
		// small setter to avoid exporting field; you can also expose SetBase on RetryManager if you prefer
		// (or recreate with a constructor that accepts base)
		// For now, we’ll just use a tiny hack: calculate with this base inline in the loop where needed.
	}

	fmt.Printf("Worker started: %d goroutine(s). Poll=%s, Backoff base=%.2f. Ctrl+C to stop…\n",
		workerCount, pollInterval, backoffBase)

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

			effectiveTimeout := time.Duration(job.TimeoutSeconds) * time.Second
			if effectiveTimeout <= 0 {
				effectiveTimeout = 30 * time.Second // safe default
			}

			execCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
			out, execErr := exec.CommandContext(execCtx, "sh", "-c", job.Command).CombinedOutput()
			cancel()

			// record logs
			_ = st.AppendLog(ctx, job.ID, "combined", string(out))
			if execErr != nil && execCtx.Err() == context.DeadlineExceeded {
				execErr = fmt.Errorf("timeout after %s", effectiveTimeout)
			}
			
			// 2) Execute the job command
			fmt.Printf("[worker %d] processing: %s (id=%s)\n", wid, job.Command, job.ID)
			out, execErr = exec.CommandContext(ctx, "sh", "-c", job.Command).CombinedOutput()
			if len(out) > 0 {
				fmt.Print(string(out))
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
			// Use attempts+1 for next schedule window
			delay := calculateBackoffWithBase(backoffBase, job.Attempts + 1, rm)
			if err := st.FailOrScheduleBackoff(ctx, job, delay, execErr.Error()); err != nil {
				fmt.Printf("[worker %d] update retry/dead error: %v\n", wid, err)
				continue
			}

			// Fetch latest attempt count to print meaningful info (optional)
			// If you don’t have a quick getter, log using the prior value.
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
		// default behavior from RetryManager (assumes its base is 2.0)
		return rm.CalculateBackoff(attempts)
	}
	// Inline calculation mirroring RetryManager semantics with cap at 5 minutes
	seconds := 1.0
	for i := 0; i < attempts; i++ {
		seconds *= base
	}
	if seconds > 300.0 { // cap at 5 minutes
		seconds = 300.0
	}
	return time.Duration(seconds) * time.Second
}


func init() {
	workerCmd.AddCommand(workerStartCmd)
	rootCmd.AddCommand(workerCmd)
}

var _ = models.Job{}

go func() {
	ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    for {
		select {
			case <-ctx.Done(): return
        case <-ticker.C:
            // any job processing but lease expired -> set pending
            if err := st.RequeueExpiredLeases(context.Background()); err != nil {
				fmt.Println("reaper:", err)
            }
        }
    }
}()