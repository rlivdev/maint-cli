package brekke

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// FindLatestBackup retorna caminho do backup mais recente do profile em backups/<name>/.
func FindLatestBackup(dataDir, name string) (string, error) {
	backupsDir := filepath.Join(dataDir, "backups", name)

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("nenhum backup encontrado para o profile %q em %s", name, backupsDir)
		}
		return "", fmt.Errorf("falha ao listar backups de %q: %w", name, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		files = append(files, e.Name())
	}

	if len(files) == 0 {
		return "", fmt.Errorf("nenhum backup encontrado para o profile %q em %s", name, backupsDir)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return filepath.Join(backupsDir, files[0]), nil
}

// RunRestore descompacta o backup e importa postgres e minio do profile.
func RunRestore(dataDir string, p *Profile, backupFile string) error {
	tmpDir, err := os.MkdirTemp("", "brekke-restore-"+p.Name+"-")
	if err != nil {
		return fmt.Errorf("falha ao criar diretório temporário: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	unpackedDir := filepath.Join(tmpDir, "data")
	if err := unpackTarGz(backupFile, unpackedDir); err != nil {
		return err
	}

	restoredPostgres := false
	restoredMinio := false

	if p.Backup.Postgres != nil {
		if err := restorePostgres(unpackedDir, p.Backup.Postgres); err != nil {
			return err
		}
		restoredPostgres = true
	}
	if p.Backup.Minio != nil {
		if err := restoreMinio(unpackedDir, p.Backup.Minio); err != nil {
			return err
		}
		restoredMinio = true
	}

	if restoredPostgres {
		fmt.Printf("PostgreSQL restaurado com sucesso (%s)\n", p.Backup.Postgres.Database)
	}
	if restoredMinio {
		fmt.Printf("MinIO restaurado com sucesso (%s)\n", p.Name)
	}

	return nil
}

func unpackTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("falha ao abrir backup %s: %w", archivePath, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("backup %s inválido (gzip): %w", archivePath, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("backup %s inválido (tar): %w", archivePath, err)
		}

		rel := filepath.ToSlash(hdr.Name)
		if rel == "" || !isSafeRel(rel) {
			return fmt.Errorf("caminho inseguro no backup: %q", hdr.Name)
		}

		target := filepath.Join(destDir, filepath.FromSlash(rel))

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			return fmt.Errorf("tipo de entrada não suportado no backup: %q", hdr.Name)
		}
	}

	return nil
}

func isSafeRel(rel string) bool {
	if rel == "" || rel == "." {
		return false
	}
	if strings.HasPrefix(rel, "/") {
		return false
	}
	// bloqueia subida de diretório e prefixos ".."
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

// findResourceDir localiza diretório relativo (ex. "postgres", "minio") no tar descompactado,
// ignorando o nome arbitrário da raiz (que pode ser o profile ou um dir temporário).
func findResourceDir(root, name string) (string, error) {
	if d := filepath.Join(root, name); dirExists(d) {
		return d, nil
	}

	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || d.Name() != name {
			return nil
		}
		found = path
		return filepath.SkipAll
	})
	if err != nil {
		return "", fmt.Errorf("falha ao localizar %q no backup: %w", name, err)
	}
	if found == "" {
		return "", fmt.Errorf("backup não contém diretório %q", name)
	}
	return found, nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func restorePostgres(root string, pg *PGConfig) error {
	dir, err := findResourceDir(root, "postgres")
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	dumpFile := filepath.Join(dir, pg.Database+".dump")
	if _, err := os.Stat(dumpFile); os.IsNotExist(err) {
		return fmt.Errorf("backup não contém o dump do banco %q", pg.Database)
	}

	args := []string{
		"-h", pg.Host,
		"-p", fmt.Sprintf("%d", pg.PortOrDefault()),
		"-U", pg.User,
		"-d", pg.Database,
		"-v", "ON_ERROR_STOP=1",
		"-f", dumpFile,
	}

	cmd := exec.Command("psql", args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pg.Password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql falhou: %w", err)
	}

	return nil
}

func restoreMinio(root string, mi *MinioConfig) error {
	dir, err := findResourceDir(root, "minio")
	if err != nil {
		return fmt.Errorf("minio: %w", err)
	}
	srcDir := dir

	run, err := newMcRunner(mi)
	if err != nil {
		return err
	}

	// Condição: buckets vazios/ausentes = restore de todos diretórios presentes no backup.
	buckets, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("falha ao listar buckets do backup: %w", err)
	}

	restoredAny := false
	for _, b := range buckets {
		if !b.IsDir() {
			continue
		}

		if !bucketAllowed(mi.Buckets, b.Name()) {
			fmt.Printf("Ignorando bucket %q (fora da lista do profile)\n", b.Name())
			continue
		}

		src := filepath.Join(srcDir, b.Name())
		dst := fmt.Sprintf("%s/%s", alias, b.Name())

		if out, err := run("mb", "--ignore-existing", fmt.Sprintf("%s/%s", alias, b.Name())); err != nil {
			return fmt.Errorf("mc mb do bucket %q falhou: %w: %s", b.Name(), err, strings.TrimSpace(out))
		}

		if out, err := run("mirror", "--overwrite", "--remove", src, dst); err != nil {
			return fmt.Errorf("mc mirror do bucket %q falhou: %w: %s", b.Name(), err, strings.TrimSpace(out))
		}
		restoredAny = true
	}

	if !restoredAny {
		return fmt.Errorf("nenhum bucket elegível para restore no backup")
	}

	return nil
}

func bucketAllowed(configured []string, name string) bool {
	if len(configured) == 0 {
		return true
	}
	for _, c := range configured {
		if c == name {
			return true
		}
	}
	return false
}

var alias = "brekke-backup"
