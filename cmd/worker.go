// cmd/worker.go (minimal)
package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/shashidhxr/queueCTL/internal/core"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{Use: "worker"}
var workerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start worker",
	RunE: func(cmd *cobra.Command, args []string) error {
		rm := core.NewRetryManager(*st) // default base=2
		fmt.Println("Worker started. Ctrl+C to stop")
		ctx := context.Background()

		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			<-ticker.C
			job, err := st.AcquireJob(ctx)
			if err != nil {
				fmt.Println("acquire error:", err)
				continue
			}
			if job == nil {
				continue // nothing due
			}

			fmt.Println("Processing:", job.Command)
			out, execErr := exec.Command("sh", "-c", job.Command).CombinedOutput()
			if len(out) > 0 {
				fmt.Print(string(out))
			}

			if execErr == nil {
				if err := st.SetCompleted(ctx, job.ID); err != nil {
					fmt.Println("complete err:", err)
				}
				continue
			}

			// retry path
			delay := rm.CalculateBackoff(job.Attempts + 1) // base^attempts (cap 5m)
			if err := st.FailOrScheduleBackoff(ctx, job, delay, execErr.Error()); err != nil {
				fmt.Println("retry/dead update err:", err)
				continue
			}
			if job.Attempts+1 > job.MaxRetries {
				fmt.Printf("Moved to DLQ (dead): %s\n", job.ID)
			} else {
				fmt.Printf("Failed (attempt %d/%d), next in %s: %s\n",
					job.Attempts+1, job.MaxRetries, delay, job.ID)
			}
		}
	},
}

func init() {
	workerCmd.AddCommand(workerStartCmd)
	rootCmd.AddCommand(workerCmd)
}
