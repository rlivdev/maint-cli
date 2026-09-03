package cmd

import (
	"github.com/spf13/cobra"
)

var minioCmd = &cobra.Command{
	Use:   "minio",
	Short: "MinIO maintenance commands",
}

func init() {
	rootCmd.AddCommand(minioCmd)
}
