package cmd

import (
	"fmt"
	"sync"

	"github.com/shashidhxr/queueCTL/pkg/models"
	"github.com/spf13/cobra"
)

var jobQueue = make([]*models.Job, 0)
var queueLock sync.Mutex

var enqueueCmd = &cobra.Command{
	Use: "enqueue [command]",
	Short: "Enqueu a job",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		queueLock.Lock()
		defer queueLock.Unlock()

		job := &models.Job {
			ID: fmt.Sprintf("job_%d", len(jobQueue) + 1),
			Command: args[0],
			State: "pending",
		}
		jobQueue = append(jobQueue, job)
		fmt.Printf("Enqueued: %+v\n", job)
	},
}

func init() {
	rootCmd.AddCommand(enqueueCmd)
}
