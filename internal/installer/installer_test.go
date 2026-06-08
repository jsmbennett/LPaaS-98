package installer

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/joeyb/lpaas-98/internal/catalog"
)

func createTestArchive(t *testing.T, filePath, fileContent string) (string, string) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}

	h := sha256.New()
	w := io.MultiWriter(f, h)

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	header := &tar.Header{
		Name: filePath,
		Mode: 0644,
		Size: int64(len(fileContent)),
	}

	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}

	if _, err := io.WriteString(tw, fileContent); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	tw.Close()
	gz.Close()
	f.Close()

	return archivePath, fmt.Sprintf("%x", h.Sum(nil))
}

func TestInstallGame(t *testing.T) {
	tmpDir := t.TempDir()
	gamesDir := filepath.Join(tmpDir, "games")
	os.MkdirAll(gamesDir, 0755)

	archivePath, checksum := createTestArchive(t, "manifest.json", `{"id":"test"}`)

	entry := &catalog.GameEntry{
		ID:      "test",
		SHA256:  checksum,
	}

	inst := New(gamesDir)
	if err := inst.InstallGame(entry, archivePath); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	manifestPath := filepath.Join(gamesDir, "test", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest not found: %v", err)
	}
}

func TestInstallGameBadChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	gamesDir := filepath.Join(tmpDir, "games")
	os.MkdirAll(gamesDir, 0755)

	archivePath, _ := createTestArchive(t, "manifest.json", `{"id":"test"}`)

	entry := &catalog.GameEntry{
		ID:     "test",
		SHA256: "wrongchecksum",
	}

	inst := New(gamesDir)
	if err := inst.InstallGame(entry, archivePath); err == nil {
		t.Fatal("expected checksum verification to fail")
	}
}

func TestUninstallGame(t *testing.T) {
	tmpDir := t.TempDir()
	gamesDir := filepath.Join(tmpDir, "games")
	gameDir := filepath.Join(gamesDir, "test")
	os.MkdirAll(gameDir, 0755)

	os.WriteFile(filepath.Join(gameDir, "test.txt"), []byte("test"), 0644)

	inst := New(gamesDir)
	if err := inst.UninstallGame("test"); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	if _, err := os.Stat(gameDir); err == nil {
		t.Fatal("game directory still exists")
	}
}
