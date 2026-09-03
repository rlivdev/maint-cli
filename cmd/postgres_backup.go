package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a PostgreSQL database",
	RunE: func(cmd *cobra.Command, args []string) error {
		host := getFlagWithEnv(cmd, "host", "MAINT_PG_HOST")
		port := getFlagWithEnv(cmd, "port", "MAINT_PG_PORT")
		user := getFlagWithEnv(cmd, "user", "MAINT_PG_USER")
		pass := getFlagWithEnv(cmd, "password", "MAINT_PG_PASSWORD")
		db := getFlagWithEnv(cmd, "database", "MAINT_PG_DATABASE")

		os.Setenv("PGPASSWORD", pass)

		filename := fmt.Sprintf("/data/postgres/%s_%s.dump", db, time.Now().Format("20060102_150405"))
		os.MkdirAll("/data/postgres", 0755)

		cmdExec := exec.Command("pg_dump", "-h", host, "-p", port, "-U", user, "-Fc", "-f", filename, db)
		cmdExec.Stdout = os.Stdout
		cmdExec.Stderr = os.Stderr

		if err := cmdExec.Run(); err != nil {
			return fmt.Errorf("backup failed for %s: %w", filename, err)
		}

		fmt.Printf("Backup completed successfully: %s\n", filename)
		return nil
	},
}

func getFlagWithEnv(cmd *cobra.Command, flag string, env string) string {
	val, _ := cmd.Flags().GetString(flag)
	if val == "" {
		if envVal := os.Getenv(env); envVal != "" {
			return envVal
		}
	}
	return val
}

func init() {
	backupCmd.Flags().String("host", "127.0.0.1", "Database host")
	backupCmd.Flags().String("port", "5432", "Database port")
	backupCmd.Flags().String("user", "", "Database user")
	backupCmd.Flags().String("password", "", "Database password")
	backupCmd.Flags().String("database", "", "Database name")

	backupCmd.MarkFlagRequired("user")
	backupCmd.MarkFlagRequired("password")
	backupCmd.MarkFlagRequired("database")

	postgresCmd.AddCommand(backupCmd)
}
