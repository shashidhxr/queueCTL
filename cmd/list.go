// cmd/list.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shashidhxr/queueCTL/pkg/models"
	"github.com/spf13/cobra"
)

var listState string
var listLimit int
var listJSON bool

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List jobs (optionally by state)",
    RunE: func(cmd *cobra.Command, args []string) error {
        state := models.JobState(listState) // "" means all
        jobs, err := st.ListJobs(context.Background(), state, listLimit)
        if err != nil { return err }
        if listJSON {
            b, _ := json.MarshalIndent(jobs, "", "  ")
            fmt.Println(string(b))
            return nil
        }
        for _, j := range jobs {
            fmt.Printf("%s  %-10s  attempts=%d/%d  cmd=%q  err=%q\n",
                j.ID, j.State, j.Attempts, j.MaxRetries, j.Command, j.Error)
        }
        return nil
    },
}

func init() {
    listCmd.Flags().StringVar(&listState, "state", "", "Filter by state (pending|processing|completed|failed|dead)")
    listCmd.Flags().IntVar(&listLimit, "limit", 50, "Max rows")
    listCmd.Flags().BoolVar(&listJSON, "json", false, "JSON output")
    rootCmd.AddCommand(listCmd)
}
