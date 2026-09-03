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

var minioRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a MinIO bucket",
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint, _ := cmd.Flags().GetString("endpoint")
		accessKey, _ := cmd.Flags().GetString("access_key")
		secretKey, _ := cmd.Flags().GetString("secret_key")
		bucket, _ := cmd.Flags().GetString("bucket")
		file, _ := cmd.Flags().GetString("file")

		alias := "maint-minio"

		// Configure mc alias
		configCmd := exec.Command("mc", "alias", "set", alias, endpoint, accessKey, secretKey)
		configCmd.Stdout = os.Stdout
		configCmd.Stderr = os.Stderr
		if err := configCmd.Run(); err != nil {
			return fmt.Errorf("failed to configure mc alias: %w", err)
		}

		if file == "" {
			var err error
			file, err = findMostRecentMinioBackup(bucket)
			if err != nil {
				return fmt.Errorf("no file specified and failed to find recent backup: %w", err)
			}
			fmt.Printf("No file specified, using most recent: %s\n", file)
		}

		// Create bucket if it does not exist
		bucketCmd := exec.Command("mc", "mb", "--ignore-existing", fmt.Sprintf("%s/%s", alias, bucket))
		bucketCmd.Stdout = os.Stdout
		bucketCmd.Stderr = os.Stderr
		if err := bucketCmd.Run(); err != nil {
			return fmt.Errorf("failed to ensure bucket exists: %w", err)
		}

		// Extract archive to a temporary directory
		tmpDir, err := os.MkdirTemp("/tmp", "minio-restore-")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		tarCmd := exec.Command("tar", "-xzf", file, "-C", tmpDir)
		tarCmd.Stdout = os.Stdout
		tarCmd.Stderr = os.Stderr
		if err := tarCmd.Run(); err != nil {
			return fmt.Errorf("failed to extract backup: %w", err)
		}

		// Mirror extracted files back to the bucket
		source := fmt.Sprintf("%s/%s/%s", tmpDir, bucket, ".")
		mirrorCmd := exec.Command("mc", "mirror", "--overwrite", source, fmt.Sprintf("%s/%s", alias, bucket))
		mirrorCmd.Stdout = os.Stdout
		mirrorCmd.Stderr = os.Stderr

		if err := mirrorCmd.Run(); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		fmt.Printf("Restore completed successfully from: %s\n", file)
		return nil
	},
}

func findMostRecentMinioBackup(bucket string) (string, error) {
	searchPath := "/data/minio"
	if _, err := os.Stat(searchPath); err != nil {
		return "", fmt.Errorf("no backup directory for bucket %q", bucket)
	}

	var files []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), bucket+"_") && strings.HasSuffix(info.Name(), ".tar.gz") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no backup files found for bucket %q", bucket)
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return files[0], nil
}

func init() {
	minioRestoreCmd.Flags().String("endpoint", "http://127.0.0.1:9000", "MinIO endpoint")
	minioRestoreCmd.Flags().String("access_key", "", "MinIO access key")
	minioRestoreCmd.Flags().String("secret_key", "", "MinIO secret key")
	minioRestoreCmd.Flags().String("bucket", "", "Bucket name")
	minioRestoreCmd.Flags().String("file", "", "Backup file to restore (optional, defaults to most recent)")

	minioRestoreCmd.MarkFlagRequired("access_key")
	minioRestoreCmd.MarkFlagRequired("secret_key")
	minioRestoreCmd.MarkFlagRequired("bucket")

	minioCmd.AddCommand(minioRestoreCmd)
}
