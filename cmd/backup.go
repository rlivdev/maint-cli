package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rlivdev/maint-cli/internal/maint"
	"github.com/spf13/cobra"
)

func newBackupCmd() *cobra.Command {
	var profileName string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Executa backup completo (postgres e minio) de um profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profileName == "" {
				profileName = getDefaultProfile()
			}

			p, err := maint.LoadProfile(dataDir, profileName)
			if err != nil {
				return err
			}

			res, err := maint.RunBackup(dataDir, p)
			if err != nil {
				return err
			}

			fmt.Printf("Backup concluído: %s (%d bytes)\n", res.Path, res.Size)
			return nil
		},
	}

	cmd.Flags().StringVarP(&profileName, "profile", "p", "", "Nome do profile (nome do arquivo em profiles/)")

	return cmd
}

func newRestoreCmd() *cobra.Command {
	var profileName string
	var file string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restaura um backup de um profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profileName == "" {
				profileName = getDefaultProfile()
			}

			p, err := maint.LoadProfile(dataDir, profileName)
			if err != nil {
				return err
			}

			var backupFile string
			if file != "" {
				backupFile = filepath.Join(dataDir, "backups", profileName, file)
				if _, err := os.Stat(backupFile); err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("arquivo %q não encontrado em %s", file, filepath.Join(dataDir, "backups", profileName))
					}
					return err
				}
			} else {
				backupFile, err = maint.FindLatestBackup(dataDir, profileName)
				if err != nil {
					return err
				}
				fmt.Printf("Usando backup mais recente: %s\n", filepath.Base(backupFile))
			}

			if err := maint.RunRestore(dataDir, p, backupFile); err != nil {
				return err
			}

			fmt.Printf("Restore concluído: %s\n", backupFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&profileName, "profile", "p", "", "Nome do profile (nome do arquivo em profiles/)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Nome do arquivo de backup .tar.gz (default: mais recente)")

	return cmd
}