package installer

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/joeyb/lpaas-98/internal/catalog"
)

type Installer struct {
	gamesDir string
}

func New(gamesDir string) *Installer {
	return &Installer{
		gamesDir: gamesDir,
	}
}

func (i *Installer) InstallGame(entry *catalog.GameEntry, archivePath string) error {
	if err := verifyChecksum(archivePath, entry.SHA256); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	gameDir := filepath.Join(i.gamesDir, entry.ID)
	tmpDir := gameDir + ".tmp"

	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("cleaning temp directory: %w", err)
	}

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}

	if err := extractTarGz(archivePath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("extracting archive: %w", err)
	}

	if err := os.RemoveAll(gameDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("removing old game directory: %w", err)
	}

	if err := os.Rename(tmpDir, gameDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("installing game: %w", err)
	}

	return nil
}

func (i *Installer) UninstallGame(gameID string) error {
	gameDir := filepath.Join(i.gamesDir, gameID)
	return os.RemoveAll(gameDir)
}

func verifyChecksum(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actual)
	}

	return nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(destDir, filepath.Clean(header.Name))

		if !isPathSafe(path, destDir) {
			return fmt.Errorf("path traversal detected in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}

			f, err := os.Create(path)
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}

			f.Close()
		}
	}

	return nil
}

func isPathSafe(path, baseDir string) bool {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && rel != ".." && len(rel) > 0 && rel[0] != '.'
}
