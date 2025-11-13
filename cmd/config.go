package cmd

import (
    "context"
    "fmt"
    "github.com/spf13/cobra"
)

var configCmd = &cobra.Command{ Use: "config", Short: "Get or set configuration" }

var configGetCmd = &cobra.Command{
    Use:   "get",
    Short: "Get all config keys",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := st.GetConfig(context.Background())
        if err != nil { 
			return err
		}
        // for k, v := range m { fmt.Printf("%s=%s\n", k, v) }
		fmt.Printf("max_retries=%d\n", cfg.MaxRetries)
		fmt.Printf("backoff_base=%d\n", cfg.BackoffBase)
        return nil
    },
}

var configSetCmd = &cobra.Command{
    Use:   "set <key> <value>",
    Short: "Set a config key",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        return st.SetConfig(context.Background(), args[0], args[1])
    },
}

func init() {
    configCmd.AddCommand(configGetCmd, configSetCmd)
    rootCmd.AddCommand(configCmd)
}
