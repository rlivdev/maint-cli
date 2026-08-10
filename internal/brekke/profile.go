package brekke

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	supportedVersion = "1"
	defaultPGPort    = 5432
	defaultMinioPort = 9000
)

var profileNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Profile struct {
	Version string `yaml:"version"`
	Name    string `yaml:"name"`

	Backup struct {
		Postgres *PGConfig `yaml:"postgres"`
		Minio    *MinioConfig `yaml:"minio"`
	} `yaml:"backup"`
}

type PGConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type MinioConfig struct {
	Host      string   `yaml:"host"`
	Port      string   `yaml:"port"`
	AccessKey string   `yaml:"access_key"`
	SecretKey string   `yaml:"secret_key"`
	Buckets   []string `yaml:"buckets"`
}

func (p *PGConfig) PortOrDefault() int {
	if p.Port == 0 {
		return defaultPGPort
	}
	return p.Port
}

func (m *MinioConfig) PortOrDefault() string {
	if m.Port == "" {
		return fmt.Sprintf("%d", defaultMinioPort)
	}
	return m.Port
}

func CheckDataDir(dataDir string) error {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return fmt.Errorf("diretório %q não encontrado", dataDir)
	}
	return nil
}

// LoadProfile carrega ~/.brekke/profiles/<name>.yaml dentro de dataDir.
func LoadProfile(dataDir, name string) (*Profile, error) {
	if !profileNameRe.MatchString(name) {
		return nil, fmt.Errorf("nome de profile inválido %q", name)
	}

	profilesDir := filepath.Join(dataDir, "profiles")
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("diretório de profiles %q não encontrado", profilesDir)
	}

	path := filepath.Join(profilesDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q não encontrado em %s", name, profilesDir)
		}
		return nil, fmt.Errorf("falha ao ler profile %q: %w", name, err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("profile %q possui YAML inválido: %w", name, err)
	}

	if p.Version == "" {
		return nil, fmt.Errorf("profile %q não define o campo 'version'", name)
	}
	if p.Version != supportedVersion {
		return nil, fmt.Errorf("profile %q usa versão %q não suportada (suportada: %q)", name, p.Version, supportedVersion)
	}

	if p.Name == "" {
		p.Name = name
	}
	if p.Name != name {
		return nil, fmt.Errorf("profile %q: campo 'name' (%q) não bate com nome do arquivo", name, p.Name)
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("profile %q inválido: %w", name, err)
	}

	return &p, nil
}

func (p *Profile) Validate() error {
	if p.Backup.Postgres == nil && p.Backup.Minio == nil {
		return fmt.Errorf("nenhum recurso configurado (backup.postgres ou backup.minio)")
	}

	if pg := p.Backup.Postgres; pg != nil {
		missing := []string{}
		if strings.TrimSpace(pg.Host) == "" {
			missing = append(missing, "backup.postgres.host")
		}
		if strings.TrimSpace(pg.User) == "" {
			missing = append(missing, "backup.postgres.user")
		}
		if strings.TrimSpace(pg.Password) == "" {
			missing = append(missing, "backup.postgres.password")
		}
		if strings.TrimSpace(pg.Database) == "" {
			missing = append(missing, "backup.postgres.database")
		}
		if len(missing) > 0 {
			return fmt.Errorf("campos obrigatórios de postgres ausentes: %s", strings.Join(missing, ", "))
		}
	}

	if mi := p.Backup.Minio; mi != nil {
		missing := []string{}
		if strings.TrimSpace(mi.Host) == "" {
			missing = append(missing, "backup.minio.host")
		}
		if strings.TrimSpace(mi.AccessKey) == "" {
			missing = append(missing, "backup.minio.access_key")
		}
		if strings.TrimSpace(mi.SecretKey) == "" {
			missing = append(missing, "backup.minio.secret_key")
		}
		if len(missing) > 0 {
			return fmt.Errorf("campos obrigatórios de minio ausentes: %s", strings.Join(missing, ", "))
		}
	}

	return nil
}