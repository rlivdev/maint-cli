package maint

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProfile(t *testing.T, dir, name, content string) {
	t.Helper()
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, name+".yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadProfile_OK(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "sample-service", `
version: "1"
name: "sample-service"
postgres:
  host: "db.local"
  user: "postgres"
  password: "secret"
  database: "app"
`)

	p, err := LoadProfile(dir, "sample-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Postgres.Host != "db.local" {
		t.Errorf("wrong host: %s", p.Postgres.Host)
	}
}

func TestLoadProfile_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "other", `version: "1"`+"\nname: other\n")

	_, err := LoadProfile(dir, "missing")
	if err == nil || !strings.Contains(err.Error(), "não encontrado") {
		t.Fatalf("want not-found error, got: %v", err)
	}
}

func TestLoadProfile_UnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "future", `
version: "999"
name: "future"
`)

	_, err := LoadProfile(dir, "future")
	if err == nil || !strings.Contains(err.Error(), "não suportada") {
		t.Fatalf("want version error, got: %v", err)
	}
}

func TestLoadProfile_NameMismatch(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "file-a", `
version: "1"
name: "file-b"
`)

	_, err := LoadProfile(dir, "file-a")
	if err == nil || !strings.Contains(err.Error(), "não bate") {
		t.Fatalf("want name mismatch error, got: %v", err)
	}
}

func TestLoadProfile_InvalidName(t *testing.T) {
	_, err := LoadProfile(t.TempDir(), "../../etc")
	if err == nil || !strings.Contains(err.Error(), "inválido") {
		t.Fatalf("want invalid name error, got: %v", err)
	}
}

func TestPackTarGz_RoundTrip(t *testing.T) {
	src := t.TempDir() + "/content"
	os.MkdirAll(filepath.Join(src, "postgres"), 0o755)
	os.WriteFile(filepath.Join(src, "postgres", "a.dump"), []byte("hello"), 0o600)

	dest := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := packTarGz(src, "content", dest); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("empty archive")
	}

	out := t.TempDir()
	if err := unpackTarGz(dest, out); err != nil {
		t.Fatal(err)
	}
	dir, err := findResourceDir(out, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "a.dump"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("wrong content: %q", data)
	}
}

func TestUnpackTarGz_PathTraversal(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "evil.tar.gz")
	f, _ := os.Create(dest)
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	hdr := &tar.Header{Name: "content/../../evil", Mode: 0o644, Size: 4, Typeflag: tar.TypeReg}
	tw.WriteHeader(hdr)
	tw.Write([]byte("evil"))
	tw.Close()
	gzw.Close()
	f.Close()

	if err := unpackTarGz(dest, t.TempDir()); err == nil {
		t.Fatal("want path traversal error")
	}
}

func TestRunMigrate_NoPostgres(t *testing.T) {
	p := &Profile{Name: "svc"}
	if err := RunMigrate(p, "./infra/flyway/migrations"); err == nil {
		t.Fatal("want error when postgres not configured")
	}
}

func TestRunMigrate_MissingDir(t *testing.T) {
	p := &Profile{Name: "svc", Postgres: &PGConfig{Host: "db", User: "u", Password: "p", Database: "d"}}
	if err := RunMigrate(p, "/nonexistent-migrations"); err == nil {
		t.Fatal("want error when migrations dir missing")
	}
}

func TestFindLatestBackup(t *testing.T) {
	dir := t.TempDir()
	backupsDir := filepath.Join(dir, "backups", "svc")
	os.MkdirAll(backupsDir, 0o755)
	os.WriteFile(filepath.Join(backupsDir, "svc-20260810-120000.tar.gz"), []byte("a"), 0o600)
	os.WriteFile(filepath.Join(backupsDir, "svc-20260810-130000.tar.gz"), []byte("b"), 0o600)

	latest, err := FindLatestBackup(dir, "svc")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(latest) != "svc-20260810-130000.tar.gz" {
		t.Fatalf("wrong latest: %s", latest)
	}
}

func TestFindLatestBackup_None(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "backups", "svc"), 0o755)
	if _, err := FindLatestBackup(dir, "svc"); err == nil {
		t.Fatal("want error when no backups")
	}
}