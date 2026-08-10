package brekke

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var now = time.Now

type BackupResult struct {
	Profile string
	Path    string
	Size    int64
}

// RunBackup executa backup completo (postgres + minio) do profile e gera .tar.gz.
func RunBackup(dataDir string, p *Profile) (*BackupResult, error) {
	stamp := now().Format("20060102-150405")
	fileName := fmt.Sprintf("%s-%s.tar.gz", p.Name, stamp)
	destDir := filepath.Join(dataDir, "backups", p.Name)
	dest := filepath.Join(destDir, fileName)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório de backup: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "brekke-"+p.Name+"-")
	if err != nil {
		return nil, fmt.Errorf("falha ao criar diretório temporário: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if p.Backup.Postgres != nil {
		if err := dumpPostgres(tmpDir, p.Backup.Postgres); err != nil {
			return nil, err
		}
	}
	if p.Backup.Minio != nil {
		if err := mirrorMinio(tmpDir, p.Backup.Minio); err != nil {
			return nil, err
		}
	}

	if err := packTarGz(tmpDir, p.Name, dest); err != nil {
		return nil, err
	}

	fi, err := os.Stat(dest)
	if err != nil {
		return nil, fmt.Errorf("falha ao verificar backup gerado: %w", err)
	}

	return &BackupResult{Profile: p.Name, Path: dest, Size: fi.Size()}, nil
}

func dumpPostgres(tmpDir string, pg *PGConfig) error {
	outDir := filepath.Join(tmpDir, "postgres")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	outFile := filepath.Join(outDir, pg.Database+".dump")
	args := []string{
		"-h", pg.Host,
		"-p", fmt.Sprintf("%d", pg.PortOrDefault()),
		"-U", pg.User,
		"-d", pg.Database,
		"--clean",
		"--if-exists",
		"-f", outFile,
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pg.Password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump falhou: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func mirrorMinio(tmpDir string, mi *MinioConfig) error {
	run, err := newMcRunner(mi)
	if err != nil {
		return err
	}

	outDir := filepath.Join(tmpDir, "minio")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	buckets := mi.Buckets
	if len(buckets) == 0 {
		list, err := run("ls", "--json", alias)
		if err != nil {
			return fmt.Errorf("mc ls falhou: %w: %s", err, strings.TrimSpace(list))
		}
		buckets = parseBucketNames(list)
	}

	for _, b := range buckets {
		src := fmt.Sprintf("%s/%s", alias, b)
		dst := filepath.Join(outDir, b)
		if out, err := run("mirror", "--overwrite", "--remove", src, dst); err != nil {
			return fmt.Errorf("mc mirror do bucket %q falhou: %w: %s", b, err, strings.TrimSpace(out))
		}
	}

	return nil
}

// newMcRunner configura o alias no mc via `mc alias set` com creds como args
// separados e um config-dir próprio. Evita falhas de parsing quando a senha
// contém caracteres especiais (ex. @, :), comuns quando embutida em MC_HOST_*.
func newMcRunner(mi *MinioConfig) (func(args ...string) (string, error), error) {
	cfgDir, err := os.MkdirTemp("", "brekke-mc-")
	if err != nil {
		return nil, fmt.Errorf("falha ao criar config-dir do mc: %w", err)
	}

	endpoint := fmt.Sprintf("%s:%s", mi.Host, mi.PortOrDefault())

	setup := exec.Command("mc", "alias", "set", alias, endpoint, mi.AccessKey, mi.SecretKey)
	setup.Env = append(os.Environ(), "MC_CONFIG_DIR="+cfgDir)
	if out, err := setup.CombinedOutput(); err != nil {
		os.RemoveAll(cfgDir)
		return nil, fmt.Errorf("mc alias set falhou: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return func(args ...string) (string, error) {
		cmd := exec.Command("mc", args...)
		cmd.Env = append(os.Environ(), "MC_CONFIG_DIR="+cfgDir)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}, nil
}

func parseBucketNames(jsonOut string) []string {
	var buckets []string
	seen := map[string]bool{}
	for _, line := range strings.Split(jsonOut, "\n") {
		if !strings.Contains(line, `"bucket"`) {
			continue
		}
		i := strings.Index(line, `"bucket"`)
		rest := line[i:]
		open := strings.Index(rest, `"`)
		close := open + 1
		// localiza valor após a chave
		open = strings.Index(rest[close:], `"`)
		if open < 0 {
			continue
		}
		valStart := close + open + 1
		valEnd := strings.Index(rest[valStart:], `"`)
		if valEnd < 0 {
			continue
		}
		name := rest[valStart : valStart+valEnd]
		if name != "" && !seen[name] {
			seen[name] = true
			buckets = append(buckets, name)
		}
	}
	return buckets
}

func packTarGz(srcDir, rootDir, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("falha ao criar %s: %w", dest, err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	base := rootDir

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		name := filepath.Join(base, rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(name)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
		return nil
	})
}
