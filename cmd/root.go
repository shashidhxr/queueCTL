package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shashidhxr/queueCTL/internal/store"
	"github.com/spf13/cobra"
)

var (
	dbPath string
	st     *store.SQLiteStorage
)

var rootCmd = &cobra.Command{
	Use:   "queuectl",
	Short: "A CLI based background job queue system.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if st != nil { return nil }
		if dbPath == "" {
			home, _ := os.UserHomeDir()
			dbPath = filepath.Join(home, ".queuectl", "queue.db")
		}
		s, err := store.NewSQLiteStorage(dbPath)
		if err != nil { return err }
		st = s
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
    }
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to SQLite DB (default $HOME/.queuectl/queue.db)")
}