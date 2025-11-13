package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use: "worker",
	Short: "Worker utilities",
}

var workerStartCmd = &cobra.Command{
	Use: "start",
	Short: "Start worker loop",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	go func() {
	// 		for {
	// 			queueLock.Lock()
	// 			for _, job := range jobQueue {
	// 				if job.State == "pending" {
	// 					fmt.Println("Prcoessing:", job.Command)
	// 					output, err := exec.Command("sh", "-c", job.Command).CombinedOutput()
	// 					fmt.Println("Output:", string(output))

	// 					if err == nil {
	// 						job.State = "completed"
	// 					} else {
	// 						fmt.Println("Error:", err)
	// 					}
	// 				}
	// 			}
	// 			queueLock.Unlock()
	// 			time.Sleep(1 * time.Second)		// avoid CPU hogging
	// 		}
	// 	}()
	// 	fmt.Println("Worker started. Ctrl + C to stop")
	// 	select {}
	// },
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Worker started. Ctrl+C to stop.")
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("\nShutting down…")
				return nil
			case <-ticker.C:
				job, err := st.AcquireJob(ctx)
				if err != nil {
					fmt.Println("acquire error:", err)
					continue
				}
				if job == nil {
					continue // nothing due
				}

				fmt.Println("Processing:", job.Command)
				out, err := exec.Command("sh", "-c", job.Command).CombinedOutput()
				if len(out) > 0 {
					fmt.Print(string(out))
				}
				if err == nil {
					if uerr := st.SetCompleted(ctx, job.ID); uerr != nil {
						fmt.Println("update error:", uerr)
					}
				} else {
					// backoff base 2 for now
					if uerr := st.SetFailedOrRetry(ctx, job, 2); uerr != nil {
						fmt.Println("update error:", uerr)
					}
					fmt.Println("exec error:", err)
				}
			}
		}
	},

}

func init() {
	workerCmd.AddCommand(workerStartCmd)
	rootCmd.AddCommand(workerCmd)
}