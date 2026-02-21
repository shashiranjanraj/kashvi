package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// Import migrations so their init() funcs run and register themselves.
	_ "github.com/shashiranjanraj/kashvi/database/migrations"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kashvi",
	Short: "Kashvi — Go framework CLI",
	Long:  "Kashvi is a Laravel-inspired Go framework. Use this CLI to scaffold and manage your project.",
}

func init() {
	// Server
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(routeListCmd)

	// Database
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(migrateRollbackCmd)
	rootCmd.AddCommand(migrateStatusCmd)
	rootCmd.AddCommand(seedCmd)

	// Workers
	rootCmd.AddCommand(queueWorkCmd)
	rootCmd.AddCommand(scheduleRunCmd)

	// Scaffolding
	rootCmd.AddCommand(makeModelCmd)
	rootCmd.AddCommand(makeControllerCmd)
	rootCmd.AddCommand(makeServiceCmd)
	rootCmd.AddCommand(makeMigrationCmd)
	rootCmd.AddCommand(makeSeederCmd)
	rootCmd.AddCommand(makeResourceCmd)
}
