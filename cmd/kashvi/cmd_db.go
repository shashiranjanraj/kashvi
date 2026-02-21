package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/database/seeders"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/migration"
)

// bootDB loads config and opens the database connection.
func bootDB() error {
	if err := config.Load(); err != nil {
		return err
	}
	database.Connect()
	return nil
}

// kashvi migrate
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run all pending database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := bootDB(); err != nil {
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
		if err := bootDB(); err != nil {
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
		if err := bootDB(); err != nil {
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
		if err := bootDB(); err != nil {
			return err
		}
		fmt.Println("Running seeders…")
		return seeders.RunAll(database.DB)
	},
}
