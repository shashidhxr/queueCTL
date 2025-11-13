package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var logsLimit int

var logsCmd = &cobra.Command{
    Use:   "logs <job_id>",
    Short: "Print stored logs for a job",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        lines, err := st.GetLogs(context.Background(), args[0], logsLimit)
        if err != nil { return err }
        for _, l := range lines {
            fmt.Printf("%s [%s] %s\n", l.TS.Format(time.RFC3339), l.Stream, l.Chunk)
        }
        return nil
    },
}

func init() {
    logsCmd.Flags().IntVar(&logsLimit, "limit", 200, "Max lines")
    rootCmd.AddCommand(logsCmd)
}
