package storage

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
)

// localDisk is the local-filesystem driver.
type localDisk struct {
	root    string // absolute root directory
	baseURL string // public URL prefix for URL()
}

func newLocalDisk() *localDisk {
	root := config.Get("STORAGE_LOCAL_ROOT", "storage")
	// Make root absolute relative to working directory.
	if !filepath.IsAbs(root) {
		cwd, _ := os.Getwd()
		root = filepath.Join(cwd, root)
	}
	return &localDisk{
		root:    root,
		baseURL: strings.TrimRight(config.Get("STORAGE_URL", "http://localhost:8080/storage"), "/"),
	}
}

func (d *localDisk) abs(path string) string {
	return filepath.Join(d.root, filepath.FromSlash(path))
}

// ── Write ─────────────────────────────────────────────────────────────────────

func (d *localDisk) Put(path string, content []byte) error {
	return d.PutStream(path, bytes.NewReader(content))
}

func (d *localDisk) PutStream(path string, r io.Reader) error {
	full := d.abs(path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("storage/local: mkdir: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("storage/local: create %s: %w", path, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage/local: write %s: %w", path, err)
	}
	return nil
}

// ── Read ──────────────────────────────────────────────────────────────────────

func (d *localDisk) Get(path string) ([]byte, error) {
	data, err := os.ReadFile(d.abs(path))
	if err != nil {
		return nil, fmt.Errorf("storage/local: get %s: %w", path, err)
	}
	return data, nil
}

func (d *localDisk) GetStream(path string) (io.ReadCloser, error) {
	f, err := os.Open(d.abs(path))
	if err != nil {
		return nil, fmt.Errorf("storage/local: open %s: %w", path, err)
	}
	return f, nil
}

// ── Metadata ──────────────────────────────────────────────────────────────────

func (d *localDisk) Exists(path string) bool {
	_, err := os.Stat(d.abs(path))
	return err == nil
}

func (d *localDisk) Missing(path string) bool { return !d.Exists(path) }

func (d *localDisk) Size(path string) (int64, error) {
	info, err := os.Stat(d.abs(path))
	if err != nil {
		return 0, fmt.Errorf("storage/local: size %s: %w", path, err)
	}
	return info.Size(), nil
}

func (d *localDisk) LastModified(path string) (time.Time, error) {
	info, err := os.Stat(d.abs(path))
	if err != nil {
		return time.Time{}, fmt.Errorf("storage/local: stat %s: %w", path, err)
	}
	return info.ModTime(), nil
}

func (d *localDisk) URL(path string) string {
	return d.baseURL + "/" + strings.TrimLeft(filepath.ToSlash(path), "/")
}

// ── Delete ────────────────────────────────────────────────────────────────────

func (d *localDisk) Delete(path string) error {
	err := os.Remove(d.abs(path))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage/local: delete %s: %w", path, err)
	}
	return nil
}

// ── Copy / Move ───────────────────────────────────────────────────────────────

func (d *localDisk) Copy(src, dst string) error {
	in, err := d.GetStream(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return d.PutStream(dst, in)
}

func (d *localDisk) Move(src, dst string) error {
	if err := d.Copy(src, dst); err != nil {
		return err
	}
	return d.Delete(src)
}

// ── Directories ───────────────────────────────────────────────────────────────

func (d *localDisk) Files(directory string) ([]string, error) {
	absDir := d.abs(directory)
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("storage/local: files %s: %w", directory, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, filepath.Join(directory, e.Name()))
		}
	}
	return out, nil
}

func (d *localDisk) AllFiles(directory string) ([]string, error) {
	absDir := d.abs(directory)
	var out []string
	err := filepath.WalkDir(absDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(d.root, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}

func (d *localDisk) Directories(directory string) ([]string, error) {
	absDir := d.abs(directory)
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("storage/local: directories %s: %w", directory, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(directory, e.Name()))
		}
	}
	return out, nil
}

func (d *localDisk) MakeDirectory(path string) error {
	if err := os.MkdirAll(d.abs(path), 0o755); err != nil {
		return fmt.Errorf("storage/local: mkdir %s: %w", path, err)
	}
	return nil
}

func (d *localDisk) DeleteDirectory(path string) error {
	if err := os.RemoveAll(d.abs(path)); err != nil {
		return fmt.Errorf("storage/local: rmdir %s: %w", path, err)
	}
	return nil
}
