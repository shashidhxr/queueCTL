package cmd

import (
	"fmt"
	
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "queuectl",
	Short: "A CLI based background job queue system.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
    }
}