package maint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const flywayImage = "redgate/flyway:13.4.0"

// RunMigrate executa as migrações flyway do profile via container redgate/flyway.
func RunMigrate(p *Profile, migrationsDir string) error {
	if p.Postgres == nil {
		return fmt.Errorf("profile %q não configura postgres", p.Name)
	}

	info, err := os.Stat(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("diretório de migrações %q não encontrado (dica: use --migrations <caminho no host>)", migrationsDir)
		}
		return fmt.Errorf("falha ao verificar %s: %w", migrationsDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q não é um diretório", migrationsDir)
	}

	absMigrations, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("falha ao obter caminho absoluto de %s: %w", migrationsDir, err)
	}

	url := fmt.Sprintf("jdbc:postgresql://%s:%d/%s", p.Postgres.Host, p.Postgres.PortOrDefault(), p.Postgres.Database)
	args := []string{
		"run", "--rm",
		"--network", "host",
		"-v", absMigrations + ":/flyway/sql",
		flywayImage,
		"-url=" + url,
		"-user=" + p.Postgres.User,
		"-password=" + p.Postgres.Password,
		"migrate",
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("flyway migrate falhou: %w", err)
	}
	return nil
}