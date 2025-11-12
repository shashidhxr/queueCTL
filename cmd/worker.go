package cmd

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use: "worker",
}

var workerStartCmd = &cobra.Command{
	Use: "start",
	Short: "Start workers",
	Run: func(cmd *cobra.Command, args []string) {
		go func() {
			for {
				queueLock.Lock()
				for _, job := range jobQueue {
					if job.State == "pending" {
						fmt.Println("Prcoessing:", job.Command)
						output, err := exec.Command("sh", "-c", job.Command).CombinedOutput()
						fmt.Println("Output:", string(output))

						if err == nil {
							job.State = "completed"
						} else {
							fmt.Println("Error:", err)
						}
					}
				}
				queueLock.Unlock()
				time.Sleep(1 * time.Second)		// avoid CPU hogging
			}
		}()
		fmt.Println("Worker started. Ctrl + C to stop")
		select {}
	},
}

func init() {
	workerCmd.AddCommand(workerStartCmd)
	rootCmd.AddCommand(workerCmd)
}