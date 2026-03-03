package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [ProjectName]",
	Short: "Create a new Kashvi project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		projectPath := filepath.Join(".", projectName)

		if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
			return fmt.Errorf("directory %s already exists", projectPath)
		}

		fmt.Printf("🚀 Scaffoldling new Kashvi project: %s\n", projectName)

		// Create directories
		dirs := []string{
			"app/controllers",
			"app/models",
			"app/routes",
			"app/services",
			"config",
			"database/migrations",
			"database/seeders",
		}

		for _, dir := range dirs {
			if err := os.MkdirAll(filepath.Join(projectPath, dir), 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}

		// Run go mod init
		fmt.Println("📦 Initializing Go module...")
		cArg := exec.Command("go", "mod", "init", projectName)
		cArg.Dir = projectPath
		if err := cArg.Run(); err != nil {
			return fmt.Errorf("failed to run go mod init: %w", err)
		}

		// Create basic files
		if err := createInitialFiles(projectPath, projectName); err != nil {
			return err
		}

		fmt.Println("📦 Downloading Kashvi dependencies...")
		cArg = exec.Command("go", "get", "github.com/shashiranjanraj/kashvi@latest")
		cArg.Dir = projectPath
		if err := cArg.Run(); err != nil {
			return fmt.Errorf("failed to get kashvi dependency: %w", err)
		}

		cArg = exec.Command("go", "mod", "tidy")
		cArg.Dir = projectPath
		_ = cArg.Run() // Ignore errors on tidy

		fmt.Println("\n✅ Project created successfully!")
		fmt.Printf("\nNext steps:\n  cd %s\n  kashvi serve\n\n", projectName)
		return nil
	},
}

func createInitialFiles(projectPath, projectName string) error {
	mainData := `package main

import (
	"github.com/shashiranjanraj/kashvi/pkg/app"

	_ "` + projectName + `/app/routes"
)

func main() {
	app.Run()
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "main.go"), []byte(mainData), 0o644); err != nil {
		return err
	}

	routesData := `package routes

import (
	"github.com/shashiranjanraj/kashvi/pkg/core"
	"github.com/shashiranjanraj/kashvi/pkg/ctx"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

func init() {
	core.RegisterRoutes(func(r *router.Router) {
		r.Get("/", "home", ctx.Wrap(func(c *ctx.Context) {
			c.Success(map[string]any{
				"message": "Welcome to Kashvi! Fast like Go, Elegant like Laravel.",
				"version": "1.0",
			})
		}))
		
		api := r.Group("/api")
		api.Get("/ping", "api.ping", ctx.Wrap(func(c *ctx.Context) {
			c.Success(map[string]any{"status": "ok"})
		}))
	})
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "app", "routes", "api.go"), []byte(routesData), 0o644); err != nil {
		return err
	}

	envData := `APP_ENV=local
APP_PORT=8080
JWT_SECRET=changelater

DB_DRIVER=sqlite
DATABASE_DSN=database.sqlite

CACHE_DRIVER=memory
QUEUE_DRIVER=sync
`
	if err := os.WriteFile(filepath.Join(projectPath, ".env.example"), []byte(envData), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectPath, ".env"), []byte(envData), 0o644); err != nil {
		return err
	}

	return nil
}
