package cmd

import (
	"fmt"

	"github.com/rlivdev/maint-cli/internal/maint"
	"github.com/spf13/cobra"
)

const defaultMigrationsDir = "./infra/flyway/migrations"

func newMigrateCmd() *cobra.Command {
	var profileName string
	var migrationsDir string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Executa migrações flyway do profile via container redgate/flyway",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profileName == "" {
				profileName = getDefaultProfile()
			}
			if migrationsDir == "" {
				migrationsDir = defaultMigrationsDir
			}

			p, err := maint.LoadProfile(dataDir, profileName)
			if err != nil {
				return err
			}

			if err := maint.RunMigrate(p, migrationsDir); err != nil {
				return err
			}

			fmt.Printf("Migração concluída (%s)\n", migrationsDir)
			return nil
		},
	}

	cmd.Flags().StringVarP(&profileName, "profile", "p", "", "Nome do profile (nome do arquivo em profiles/)")
	cmd.Flags().StringVar(&migrationsDir, "migrations", defaultMigrationsDir, "Diretório com migrações flyway (padrão: ./infra/flyway/migrations)")

	return cmd
}