package main

// cmd_db.go — database sub-commands for the global kashvi CLI.
//
// In project mode (running from a user's project directory), these commands
// delegate to the user's own binary via `go run . <cmd>` so that the
// project's migrations and seeders are properly registered.
//
// In framework-self mode (running inside the kashvi source tree), the commands
// operate on the framework's own database/migrations package directly.

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shashiranjanraj/kashvi/config"
	_ "github.com/shashiranjanraj/kashvi/database/migrations"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/migration"
)

// bootDBDirect loads config and opens the database connection.
// Used only in framework-self mode.
func bootDBDirect() error {
	if err := config.Load(); err != nil {
		return err
	}
	// The blank import of database/migrations registers all migrations via init().
	return database.Connect()
}

// kashvi migrate
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run all pending database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isFrameworkSelf() {
			return runInProject("migrate")
		}
		if err := bootDBDirect(); err != nil {
			return err
		}
		fmt.Println("Running migrations…")
		return migration.New(database.DB).Run()
	},
}

// kashvi migrate:rollback
var migrateRollbackCmd = &cobra.Command{
	Use:   "migrate:rollback",
	Short: "Rollback the last batch of migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isFrameworkSelf() {
			return runInProject("migrate:rollback")
		}
		if err := bootDBDirect(); err != nil {
			return err
		}
		fmt.Println("Rolling back last batch…")
		return migration.New(database.DB).Rollback()
	},
}

// kashvi migrate:status
var migrateStatusCmd = &cobra.Command{
	Use:   "migrate:status",
	Short: "Show the status of each migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isFrameworkSelf() {
			return runInProject("migrate:status")
		}
		if err := bootDBDirect(); err != nil {
			return err
		}
		return migration.New(database.DB).Status()
	},
}

// kashvi seed
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run all database seeders",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Always delegate to project — the global CLI binary has no project seeders.
		return runInProject("seed")
	},
}
