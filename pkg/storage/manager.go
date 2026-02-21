package storage

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
)

// ─── Manager ──────────────────────────────────────────────────────────────────

var (
	managerMu   sync.RWMutex
	disks       = map[string]Disk{}
	defaultDisk string
)

// Connect boots the storage manager.
// Call once at application startup (e.g. in internal/server/server.go).
func Connect() {
	defaultDisk = config.Get("STORAGE_DISK", "local")

	// Always boot local disk.
	disks["local"] = newLocalDisk()

	// Boot S3 disk only if bucket is configured.
	if config.Get("S3_BUCKET", "") != "" {
		d, err := newS3Disk()
		if err != nil {
			fmt.Printf("⚠️  storage/s3: %v (disk disabled)\n", err)
		} else {
			disks["s3"] = d
		}
	}
}

// Use returns the named disk.
// Use the driver names "local" or "s3".
//
//	storage.Use("s3").Put("backups/dump.sql", data)
func Use(name string) Disk {
	managerMu.RLock()
	d, ok := disks[name]
	managerMu.RUnlock()
	if !ok {
		panic(fmt.Sprintf("storage: disk %q is not configured", name))
	}
	return d
}

// RegisterDisk lets you plug in a custom Disk implementation at boot time.
func RegisterDisk(name string, d Disk) {
	managerMu.Lock()
	disks[name] = d
	managerMu.Unlock()
}

// ─── Default disk helpers ─────────────────────────────────────────────────────
// These proxy to the default disk (STORAGE_DISK env var, default "local").

func defaultD() Disk { return Use(defaultDisk) }

// Put writes content to path on the default disk.
func Put(path string, content []byte) error { return defaultD().Put(path, content) }

// PutStream writes from r to path on the default disk.
func PutStream(path string, r io.Reader) error { return defaultD().PutStream(path, r) }

// Get returns file content from the default disk.
func Get(path string) ([]byte, error) { return defaultD().Get(path) }

// GetStream returns a ReadCloser from the default disk.
func GetStream(path string) (io.ReadCloser, error) { return defaultD().GetStream(path) }

// Exists reports whether path exists on the default disk.
func Exists(path string) bool { return defaultD().Exists(path) }

// Missing reports whether path is absent on the default disk.
func Missing(path string) bool { return defaultD().Missing(path) }

// Delete removes path from the default disk.
func Delete(path string) error { return defaultD().Delete(path) }

// URL returns the public URL for path on the default disk.
func URL(path string) string { return defaultD().URL(path) }

// Copy copies src to dst on the default disk.
func Copy(src, dst string) error { return defaultD().Copy(src, dst) }

// Move moves src to dst on the default disk.
func Move(src, dst string) error { return defaultD().Move(src, dst) }

// Size returns the file size in bytes on the default disk.
func Size(path string) (int64, error) { return defaultD().Size(path) }

// LastModified returns last-modified time on the default disk.
func LastModified(path string) (time.Time, error) { return defaultD().LastModified(path) }

// Files lists files in directory (non-recursive) on the default disk.
func Files(directory string) ([]string, error) { return defaultD().Files(directory) }

// AllFiles lists all files in directory (recursive) on the default disk.
func AllFiles(directory string) ([]string, error) { return defaultD().AllFiles(directory) }

// Directories lists sub-directories of directory on the default disk.
func Directories(directory string) ([]string, error) { return defaultD().Directories(directory) }

// MakeDirectory creates directory on the default disk.
func MakeDirectory(path string) error { return defaultD().MakeDirectory(path) }

// DeleteDirectory removes directory and its contents on the default disk.
func DeleteDirectory(path string) error { return defaultD().DeleteDirectory(path) }
