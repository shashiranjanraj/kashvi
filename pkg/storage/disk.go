// Package storage provides a Laravel-inspired filesystem abstraction for Kashvi.
//
// Two drivers are available out of the box:
//   - "local"  — local filesystem (default)
//   - "s3"     — S3-compatible object storage (AWS S3, MinIO, R2, Spaces)
//
// Quick start:
//
//	// boot once (e.g. in internal/server/server.go):
//	storage.Connect()
//
//	// default disk
//	storage.Put("images/photo.jpg", data)
//	data, _ := storage.Get("images/photo.jpg")
//	url  := storage.URL("images/photo.jpg")
//
//	// named disk
//	storage.Disk("s3").Put("backups/dump.sql.gz", data)
package storage

import (
	"io"
	"time"
)

// Disk is the filesystem driver interface. Every driver must implement this.
type Disk interface {
	// ── Write ──────────────────────────────────────────────────────────────────

	// Put writes content to path, creating parent directories as needed.
	Put(path string, content []byte) error

	// PutStream writes from r to path.
	PutStream(path string, r io.Reader) error

	// ── Read ───────────────────────────────────────────────────────────────────

	// Get returns the full content of the file at path.
	Get(path string) ([]byte, error)

	// GetStream returns a ReadCloser for the file. Caller must close it.
	GetStream(path string) (io.ReadCloser, error)

	// ── Metadata ───────────────────────────────────────────────────────────────

	// Exists reports whether a file exists at path.
	Exists(path string) bool

	// Missing is the inverse of Exists.
	Missing(path string) bool

	// Size returns the byte size of the file.
	Size(path string) (int64, error)

	// LastModified returns the file's last-modified time.
	LastModified(path string) (time.Time, error)

	// URL returns the public URL for path (meaningful for public disks / S3).
	URL(path string) string

	// ── Delete ─────────────────────────────────────────────────────────────────

	// Delete removes a file. Returns nil if the file did not exist.
	Delete(path string) error

	// ── Copy / Move ────────────────────────────────────────────────────────────

	// Copy creates a copy of src at dst.
	Copy(src, dst string) error

	// Move moves (renames) src to dst.
	Move(src, dst string) error

	// ── Directories ────────────────────────────────────────────────────────────

	// Files lists non-recursive filenames directly inside directory.
	Files(directory string) ([]string, error)

	// AllFiles lists all files inside directory, recursively.
	AllFiles(directory string) ([]string, error)

	// Directories lists the immediate sub-directories of directory.
	Directories(directory string) ([]string, error)

	// MakeDirectory creates directory (and any parents).
	MakeDirectory(path string) error

	// DeleteDirectory removes directory and all its contents.
	DeleteDirectory(path string) error
}
