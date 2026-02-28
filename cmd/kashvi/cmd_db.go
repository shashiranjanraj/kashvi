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
	"os"

	"github.com/spf13/cobra"
)

// runInProject executes the given subcommand in the user's project directory
// Note: the implementation is expected to live in cmd_delegate.go or similar

// kashvi migrate
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run all pending database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isFrameworkSelf() {
			return runInProject("migrate")
		}
		fmt.Println("kashvi migrate can only be run inside a Kashvi project directory.")
		os.Exit(1)
		return nil
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
		fmt.Println("kashvi migrate:rollback can only be run inside a Kashvi project directory.")
		os.Exit(1)
		return nil
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
		fmt.Println("kashvi migrate:status can only be run inside a Kashvi project directory.")
		os.Exit(1)
		return nil
	},
}

// kashvi seed
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run all database seeders",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Always delegate to project
		if !isFrameworkSelf() {
			return runInProject("seed")
		}
		fmt.Println("kashvi seed can only be run inside a Kashvi project directory.")
		os.Exit(1)
		return nil
	},
}
