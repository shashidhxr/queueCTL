package cmd

import (
	"context"
	"fmt"

	"github.com/shashidhxr/queueCTL/pkg/models"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Print counts of jobs by state",
    RunE: func(cmd *cobra.Command, args []string) error {
        m, err := st.GetJobStats(context.Background())
        if err != nil { return err }
        fmt.Printf("pending=%d processing=%d completed=%d failed=%d dead=%d\n",
            m[models.StatePending], m[models.StateProcessing], m[models.StateCompleted], m[models.StateFailed], m[models.StateDead])
        return nil
    },
}

func init() { rootCmd.AddCommand(statusCmd) }
