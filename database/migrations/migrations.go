// Package migrations contains all database migration files.
// Each migration file uses init() to call migration.Register().
// This package is imported by cmd/kashvi/main.go to ensure all
// migrations are registered at CLI startup.
package migrations
