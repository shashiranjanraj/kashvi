// Package migration provides a database migration runner for Kashvi.
//
// Usage (in database/migrations/register.go):
//
//	func init() {
//	    migration.Register("20240101000000_create_users_table", &CreateUsersTable{})
//	}
//
//	type CreateUsersTable struct{}
//	func (m *CreateUsersTable) Up(db *gorm.DB) error {
//	    return db.AutoMigrate(&models.User{})
//	}
//	func (m *CreateUsersTable) Down(db *gorm.DB) error {
//	    return db.Migrator().DropTable("users")
//	}
//
// Run from CLI:
//
//	kashvi migrate             // run all pending
//	kashvi migrate:rollback    // rollback last batch
package migration

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"gorm.io/gorm"
)

// Migration is the interface every migration must implement.
type Migration interface {
	// Up applies the migration.
	Up(db *gorm.DB) error
	// Down reverses the migration.
	Down(db *gorm.DB) error
}

// migrationRecord is the GORM model stored in the tracking table.
type migrationRecord struct {
	ID    uint      `gorm:"primaryKey;autoIncrement"`
	Name  string    `gorm:"uniqueIndex;size:255;not null"`
	Batch int       `gorm:"not null"`
	RunAt time.Time `gorm:"autoCreateTime"`
}

func (migrationRecord) TableName() string { return "kashvi_migrations" }

// ------------------- Registry -------------------

type registeredMigration struct {
	name string
	m    Migration
}

var registry []registeredMigration

// Register adds a migration to the global registry.
// name should be a timestamp-prefixed string, e.g. "20240101000000_create_users_table".
// Migrations are run in the order they are registered, so call Register in
// chronological order (use an init() in each migration file).
func Register(name string, m Migration) {
	registry = append(registry, registeredMigration{name: name, m: m})
}

// ------------------- Runner -------------------

// Runner executes and tracks migrations.
type Runner struct {
	db *gorm.DB
}

// New creates a Runner backed by the provided gorm.DB.
func New(db *gorm.DB) *Runner {
	return &Runner{db: db}
}

// EnsureTable creates the tracking table if it does not exist.
func (r *Runner) EnsureTable() error {
	return r.db.AutoMigrate(&migrationRecord{})
}

// Pending returns the names of migrations that have not yet been run.
func (r *Runner) Pending() ([]registeredMigration, error) {
	var ran []migrationRecord
	if err := r.db.Find(&ran).Error; err != nil {
		return nil, err
	}

	ranSet := make(map[string]bool, len(ran))
	for _, rec := range ran {
		ranSet[rec.Name] = true
	}

	var pending []registeredMigration
	for _, reg := range registry {
		if !ranSet[reg.name] {
			pending = append(pending, reg)
		}
	}

	// Ensure stable ordering by name (timestamps sort lexicographically).
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].name < pending[j].name
	})

	return pending, nil
}

// Run executes all pending migrations in a single batch.
func (r *Runner) Run() error {
	if err := r.EnsureTable(); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}

	pending, err := r.Pending()
	if err != nil {
		return fmt.Errorf("migration: fetch pending: %w", err)
	}

	if len(pending) == 0 {
		logger.Info("migration: nothing to migrate")
		fmt.Println("Nothing to migrate.")
		return nil
	}

	batch := r.nextBatch()

	for _, reg := range pending {
		logger.Info("migration: running", "name", reg.name)
		fmt.Printf("  ▶ Migrating: %s\n", reg.name)

		if err := reg.m.Up(r.db); err != nil {
			return fmt.Errorf("migration: %s up: %w", reg.name, err)
		}

		record := migrationRecord{Name: reg.name, Batch: batch}
		if err := r.db.Create(&record).Error; err != nil {
			return fmt.Errorf("migration: record %s: %w", reg.name, err)
		}

		fmt.Printf("  ✅ Migrated:  %s\n", reg.name)
	}

	logger.Info("migration: done", "ran", len(pending), "batch", batch)
	return nil
}

// Rollback reverses all migrations from the most recent batch.
func (r *Runner) Rollback() error {
	if err := r.EnsureTable(); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}

	// Find the last batch number.
	var maxBatch struct{ Max int }
	r.db.Model(&migrationRecord{}).Select("MAX(batch) as max").Scan(&maxBatch)
	if maxBatch.Max == 0 {
		fmt.Println("Nothing to roll back.")
		return nil
	}

	// Get all migrations in that batch, descending order.
	var records []migrationRecord
	if err := r.db.Where("batch = ?", maxBatch.Max).
		Order("id desc").
		Find(&records).Error; err != nil {
		return err
	}

	// Find corresponding Migration implementations.
	regMap := make(map[string]Migration, len(registry))
	for _, reg := range registry {
		regMap[reg.name] = reg.m
	}

	for _, rec := range records {
		m, ok := regMap[rec.Name]
		if !ok {
			return fmt.Errorf("migration: cannot rollback %s — not registered", rec.Name)
		}

		fmt.Printf("  ◀ Rolling back: %s\n", rec.Name)
		logger.Info("migration: rolling back", "name", rec.Name)

		if err := m.Down(r.db); err != nil {
			return fmt.Errorf("migration: %s down: %w", rec.Name, err)
		}

		if err := r.db.Delete(&rec).Error; err != nil {
			return err
		}

		fmt.Printf("  ✅ Rolled back:  %s\n", rec.Name)
	}

	return nil
}

// Status prints all migrations and whether each has been run.
func (r *Runner) Status() error {
	if err := r.EnsureTable(); err != nil {
		return err
	}

	var ran []migrationRecord
	if err := r.db.Find(&ran).Error; err != nil {
		return err
	}

	ranMap := make(map[string]migrationRecord, len(ran))
	for _, rec := range ran {
		ranMap[rec.Name] = rec
	}

	fmt.Printf("%-60s  %-8s  %s\n", "Migration", "Status", "Batch")
	fmt.Println(string(make([]byte, 80)))
	for _, reg := range registry {
		if rec, ok := ranMap[reg.name]; ok {
			fmt.Printf("%-60s  %-8s  %d\n", reg.name, "Ran", rec.Batch)
		} else {
			fmt.Printf("%-60s  %-8s  -\n", reg.name, "Pending")
		}
	}
	return nil
}

func (r *Runner) nextBatch() int {
	var maxBatch struct{ Max int }
	r.db.Model(&migrationRecord{}).Select("MAX(batch) as max").Scan(&maxBatch)
	return maxBatch.Max + 1
}

// ErrNoMigrations is returned when Run is called but no migrations are registered.
var ErrNoMigrations = errors.New("no migrations registered")
