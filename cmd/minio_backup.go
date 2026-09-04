package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var minioBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a MinIO bucket",
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint, _ := cmd.Flags().GetString("endpoint")
		accessKey, _ := cmd.Flags().GetString("access-key")
		secretKey, _ := cmd.Flags().GetString("secret-key")
		bucket, _ := cmd.Flags().GetString("bucket")

		alias := "maint-minio"

		// Configure mc alias
		configCmd := exec.Command("mc", "alias", "set", alias, endpoint, accessKey, secretKey)
		configCmd.Stdout = os.Stdout
		configCmd.Stderr = os.Stderr
		if err := configCmd.Run(); err != nil {
			return fmt.Errorf("failed to configure mc alias: %w", err)
		}

		// Prepare backup directory named after the bucket
		backupPath := fmt.Sprintf("/data/minio/%s", bucket)
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}

		// Execute mirror
		source := fmt.Sprintf("%s/%s", alias, bucket)
		mirrorCmd := exec.Command("mc", "mirror", "--overwrite", source, backupPath)
		mirrorCmd.Stdout = os.Stdout
		mirrorCmd.Stderr = os.Stderr

		if err := mirrorCmd.Run(); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}

		// Compress backup (internal folder keeps the bucket name)
		tarFile := fmt.Sprintf("/data/minio/%s_%s.tar.gz", bucket, time.Now().Format("20060102_150405"))
		tarCmd := exec.Command("tar", "-czf", tarFile, "-C", filepath.Dir(backupPath), filepath.Base(backupPath))
		tarCmd.Stdout = os.Stdout
		tarCmd.Stderr = os.Stderr

		if err := tarCmd.Run(); err != nil {
			return fmt.Errorf("compression failed: %w", err)
		}

		// Cleanup uncompressed bucket directory
		os.RemoveAll(backupPath)

		absPath, _ := filepath.Abs(tarFile)
		fmt.Printf("Backup completed successfully: %s\n", absPath)
		return nil
	},
}

func init() {
	minioBackupCmd.Flags().String("endpoint", "http://127.0.0.1:9000", "MinIO endpoint")
	minioBackupCmd.Flags().String("access-key", "", "MinIO access key")
	minioBackupCmd.Flags().String("secret-key", "", "MinIO secret key")
	minioBackupCmd.Flags().String("bucket", "", "Bucket name")

	minioBackupCmd.MarkFlagRequired("access-key")
	minioBackupCmd.MarkFlagRequired("secret-key")
	minioBackupCmd.MarkFlagRequired("bucket")

	minioCmd.AddCommand(minioBackupCmd)
}
