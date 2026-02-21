// Package seeders provides a registry of database seed functions.
//
// Usage (define a seeder in any file in this package):
//
//	func init() {
//	    seeders.Register("users", SeedUsers)
//	}
//
//	func SeedUsers(db *gorm.DB) error {
//	    // insert rows …
//	    return nil
//	}
//
// Then run via CLI: kashvi seed
package seeders

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// SeederFunc is the signature for a seed function.
type SeederFunc func(db *gorm.DB) error

type seederEntry struct {
	name string
	fn   SeederFunc
}

var (
	mu      sync.Mutex
	entries []seederEntry
)

// Register adds a seeder to the global registry.
// Call this from init() in your seeder files.
func Register(name string, fn SeederFunc) {
	mu.Lock()
	defer mu.Unlock()
	entries = append(entries, seederEntry{name: name, fn: fn})
}

// RunAll executes every registered seeder in registration order.
// It stops on the first error.
func RunAll(db *gorm.DB) error {
	mu.Lock()
	current := make([]seederEntry, len(entries))
	copy(current, entries)
	mu.Unlock()

	if len(current) == 0 {
		fmt.Println("  (no seeders registered)")
		return nil
	}

	for _, e := range current {
		fmt.Printf("  • Running seeder: %s … ", e.name)
		if err := e.fn(db); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("seeder %q: %w", e.name, err)
		}
		fmt.Println("done")
	}
	return nil
}
