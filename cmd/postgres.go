package cmd

import (
	"github.com/spf13/cobra"
)

var postgresCmd = &cobra.Command{
	Use:   "postgres",
	Short: "PostgreSQL maintenance commands",
}

func init() {
	rootCmd.AddCommand(postgresCmd)
}
