package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a PostgreSQL database",
	RunE: func(cmd *cobra.Command, args []string) error {
		host := getFlagWithEnv(cmd, "host", "MAINT_PG_HOST")
		port := getFlagWithEnv(cmd, "port", "MAINT_PG_PORT")
		user := getFlagWithEnv(cmd, "user", "MAINT_PG_USER")
		pass := getFlagWithEnv(cmd, "password", "MAINT_PG_PASSWORD")
		db := getFlagWithEnv(cmd, "database", "MAINT_PG_DATABASE")
		file, _ := cmd.Flags().GetString("file")

		os.Setenv("PGPASSWORD", pass)

		if file == "" {
			var err error
			file, err = findMostRecentForDB(db)
			if err != nil {
				return fmt.Errorf("no file specified and failed to find recent dump: %w", err)
			}
			fmt.Printf("No file specified, using most recent: %s\n", file)
		}

		cmdExec := exec.Command("pg_restore", "-h", host, "-p", port, "-U", user, "-d", db, "--no-owner", "--no-acl", file)
		cmdExec.Stdout = os.Stdout
		cmdExec.Stderr = os.Stderr

		if err := cmdExec.Run(); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		fmt.Printf("Restore completed successfully from: %s\n", file)
		return nil
	},
}

func findMostRecentForDB(db string) (string, error) {
	searchPath := "/data/postgres"
	if _, err := os.Stat(searchPath); err != nil {
		return "", fmt.Errorf("no backup directory for database %q", db)
	}

	var files []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), db+"_") && strings.HasSuffix(info.Name(), ".dump") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no dump files found for database %q", db)
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return files[0], nil
}

func init() {
	restoreCmd.Flags().String("host", "127.0.0.1", "Database host")
	restoreCmd.Flags().String("port", "5432", "Database port")
	restoreCmd.Flags().String("user", "", "Database user")
	restoreCmd.Flags().String("password", "", "Database password")
	restoreCmd.Flags().String("database", "", "Database name")
	restoreCmd.Flags().String("file", "", "Dump file to restore (optional, defaults to most recent)")

	restoreCmd.MarkFlagRequired("user")
	restoreCmd.MarkFlagRequired("password")
	restoreCmd.MarkFlagRequired("database")

	postgresCmd.AddCommand(restoreCmd)
}
