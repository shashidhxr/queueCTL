// cmd/queue.go
package cmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/shashidhxr/queueCTL/pkg/models"
	"github.com/spf13/cobra"
)

var jobQueue = make([]*models.Job, 0)
var queueLock sync.Mutex

var enqueueCmd = &cobra.Command{
	Use:   "enqueue [command]",
	Short: "Enqueue a job",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		j := &models.Job{
			ID:         uuid.NewString(),
			Command:    args[0],
			State:      models.StatePending,
			MaxRetries: 3,
		}
		if err := st.SaveJob(context.Background(), j); err != nil {
			return err
		}
		fmt.Printf("Enqueued: %+v\n", *j)
		return nil
	},
}

func init() { rootCmd.AddCommand(enqueueCmd) }
