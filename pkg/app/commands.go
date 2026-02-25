package app

// pkg/app/commands.go — implementations for all CLI sub-commands.
// These are called from Application.Run() and use only framework packages.

import (
	"fmt"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/migration"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

// cmdServe boots the HTTP + gRPC servers using the Application's handler.
func cmdServe(a *Application) error {
	return startServer(a)
}

// cmdMigrate runs all pending migrations.
func cmdMigrate() error {
	if err := bootDB(); err != nil {
		return err
	}
	return migration.New(database.DB).Run()
}

// cmdMigrateRollback reverses the last migration batch.
func cmdMigrateRollback() error {
	if err := bootDB(); err != nil {
		return err
	}
	return migration.New(database.DB).Rollback()
}

// cmdMigrateStatus prints migration status.
func cmdMigrateStatus() error {
	if err := bootDB(); err != nil {
		return err
	}
	return migration.New(database.DB).Status()
}

// cmdSeed runs all registered seeders (global + per-application).
func cmdSeed(seeders []SeederFunc) error {
	if err := bootDB(); err != nil {
		return err
	}
	if len(seeders) == 0 {
		fmt.Println("No seeders registered. Use app.RegisterSeeder() or .Seeders() on Application.")
		return nil
	}
	for _, fn := range seeders {
		fn()
	}
	fmt.Printf("✅ Seeding complete (%d seeders ran)\n", len(seeders))
	return nil
}

// cmdRouteList prints all registered routes.
func cmdRouteList(a *Application) error {
	r := router.New()
	for _, fn := range a.routesFns {
		fn(r)
	}

	routes := r.Routes()
	if len(routes) == 0 {
		fmt.Println("No routes registered.")
		return nil
	}

	fmt.Printf("%-8s  %-50s  %s\n", "METHOD", "PATH", "NAME")
	fmt.Println(func(n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = '-'
		}
		return string(b)
	}(80))
	for _, ri := range routes {
		fmt.Printf("%-8s  %-50s  %s\n", ri.Method, ri.Path, ri.Name)
	}
	return nil
}

// bootDB loads config and connects to the database.
func bootDB() error {
	if err := config.Load(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return database.Connect()
}
