package main

// cmd_delegate.go provides the project-delegation mechanism.
//
// When `kashvi <cmd>` is run inside a user's project directory (not the
// kashvi framework source), it executes `go run . <cmd>` so the user's
// own main.go (which calls app.Run()) handles the command with the project's
// migrations, seeders and routes registered.

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// runInProject runs `go run <dir> <subcommand>` in the current working directory.
// It is used when the kashvi CLI is acting as an external driver for a
// user project rather than the framework's own internal server.
func runInProject(subcommand string) error {
	cwd, _ := os.Getwd()
	dir := findEntrypoint(cwd)
	args := []string{"run", dir, subcommand}

	c := exec.Command("go", args...)
	c.Dir = cwd
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()
	return c.Run()
}

// findEntrypoint returns the Go package path to pass to `go run`.
// It checks whether the cwd itself has Go files; if not it probes
// common subdirectory conventions used by Go projects.
func findEntrypoint(cwd string) string {
	// If there are Go files in the cwd, use "." (standard layout)
	entries, err := os.ReadDir(cwd)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && len(e.Name()) > 3 && e.Name()[len(e.Name())-3:] == ".go" {
				return "."
			}
		}
	}

	// Probe common entrypoint subdirectories in priority order
	candidates := []string{
		"cmd/server",
		"cmd/app",
		"cmd/main",
		"main",
		"cmd",
	}
	for _, sub := range candidates {
		subEntries, err := os.ReadDir(cwd + "/" + sub)
		if err != nil {
			continue
		}
		for _, e := range subEntries {
			if !e.IsDir() && len(e.Name()) > 3 && e.Name()[len(e.Name())-3:] == ".go" {
				return "./" + sub
			}
		}
	}

	// Fallback: let `go run .` produce the original error message
	return "."
}

// isProjectMode returns true when the CLI is being used outside the kashvi
// framework source tree. We detect this by looking for go.mod in the cwd.
// When running inside the kashvi repo itself, direct package imports are used.
func isFrameworkSelf() bool {
	_, err := os.Stat("pkg/app/app.go")
	return err == nil
}

// projectCmds are commands that delegate to the user's project when the
// CLI is used externally. They wrap the framework-internal commands so
// `kashvi serve`, `kashvi migrate` etc. work from any project directory.
func addProjectDelegateCmds(root *cobra.Command) {
	if isFrameworkSelf() {
		return // Kashvi framework dev mode — direct imports already registered
	}

	// Override run/serve/start to delegate to user project
	for _, name := range []string{"run", "serve", "start"} {
		cmd := name
		root.AddCommand(&cobra.Command{
			Use:   cmd,
			Short: "Start the HTTP + gRPC server (delegates to your project)",
			RunE: func(c *cobra.Command, args []string) error {
				return runInProject("serve")
			},
		})
	}

	root.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run pending migrations (delegates to your project)",
		RunE: func(c *cobra.Command, args []string) error {
			return runInProject("migrate")
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "migrate:rollback",
		Short: "Rollback last batch of migrations",
		RunE: func(c *cobra.Command, args []string) error {
			return runInProject("migrate:rollback")
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "migrate:status",
		Short: "Show migration status",
		RunE: func(c *cobra.Command, args []string) error {
			return runInProject("migrate:status")
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "seed",
		Short: "Seed the database (delegates to your project)",
		RunE: func(c *cobra.Command, args []string) error {
			return runInProject("seed")
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "route:list",
		Short: "List registered API routes",
		RunE: func(c *cobra.Command, args []string) error {
			return runInProject("route:list")
		},
	})
}

func printQuickStart() {
	fmt.Println(`
  kashvi – Go Web Framework  ⚡

  Install globally:
    go install github.com/shashiranjanraj/kashvi/cmd/kashvi@latest

  Your project's main.go:
    package main
    import (
        "github.com/shashiranjanraj/kashvi/pkg/app"
        _ "yourproject/database/migrations"
        _ "yourproject/database/seeders"
    )
    func main() { app.Run() }

  Commands (run from your project directory):
    kashvi serve            Start HTTP + gRPC server
    kashvi migrate          Run pending migrations
    kashvi migrate:rollback Rollback last batch
    kashvi migrate:status   Show migration status
    kashvi seed             Seed the database
    kashvi route:list       List all API routes
`)
}
