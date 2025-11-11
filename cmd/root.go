package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "queuectl",
	Short: "A CLI based background job queue system.",
	Long: `QueueCTL is a production-grade job queue system
that manages background jobs with workers, retries, and a DLQ.
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}