package cmd

import (
	"fmt"

	"github.com/rlivdev/maint-cli/internal/maint"
	"github.com/spf13/cobra"
)

var dataDir string

var rootCmd = &cobra.Command{
	Use:   "maint",
	Short: "maint é uma ferramenta de backup, restore e migração para PostgreSQL e MinIO",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", "/data", "Diretório raiz de dados (profiles e backups)")

	rootCmd.AddCommand(newBackupCmd())
	rootCmd.AddCommand(newRestoreCmd())
	rootCmd.AddCommand(newMigrateCmd())
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := maint.CheckDataDir(dataDir); err != nil {
			return fmt.Errorf("data dir inválido: %w", err)
		}
		return nil
	}
}

func getDefaultProfile() string {
	return "default"
}