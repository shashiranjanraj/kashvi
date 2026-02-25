package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
	if isFrameworkSelf() {
		// ── Framework dev mode: direct imports used, no delegation.
		rootCmd.AddCommand(runCmd)
		rootCmd.AddCommand(buildCmd)
		rootCmd.AddCommand(serveCmd)
		rootCmd.AddCommand(routeListCmd)
		rootCmd.AddCommand(grpcServeCmd)

		// Database commands (direct — only useful inside framework repo)
		rootCmd.AddCommand(migrateCmd)
		rootCmd.AddCommand(migrateRollbackCmd)
		rootCmd.AddCommand(migrateStatusCmd)
		rootCmd.AddCommand(seedCmd)

		// Workers (direct)
		rootCmd.AddCommand(queueWorkCmd)
		rootCmd.AddCommand(scheduleRunCmd)
	} else {
		// ── Project mode: delegate ALL runtime commands to the user's
		// own main.go (which calls app.Run()) via `go run . <cmd>`.
		// This ensures the project's own migrations, seeders and routes
		// are properly registered.
		addProjectDelegateCmds(rootCmd)
	}

	// Scaffolding generators — always available, they only create files.
	rootCmd.AddCommand(makeModelCmd)
	rootCmd.AddCommand(makeControllerCmd)
	rootCmd.AddCommand(makeServiceCmd)
	rootCmd.AddCommand(makeMigrationCmd)
	rootCmd.AddCommand(makeSeederCmd)
	rootCmd.AddCommand(makeResourceCmd)
}
