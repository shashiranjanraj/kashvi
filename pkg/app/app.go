// Package app provides the Kashvi application runner.
//
// # Minimal usage (any Go project)
//
//	package main
//
//	import (
//	    "net/http"
//	    "fmt"
//	    "github.com/shashiranjanraj/kashvi/pkg/app"
//	    "github.com/shashiranjanraj/kashvi/pkg/router"
//	    _ "yourproject/database/migrations"
//	    _ "yourproject/database/seeders"
//	)
//
//	func main() {
//	    app.New().
//	        Routes(func(r *router.Router) {
//	            r.Get("/hello", "hello", func(w http.ResponseWriter, req *http.Request) {
//	                fmt.Fprintln(w, "Hello from Kashvi!")
//	            })
//	        }).
//	        AutoMigrate(&User{}).
//	        Run()
//	}
//
// Then run with the global kashvi CLI or directly:
//
//	kashvi serve
//	kashvi migrate
//	kashvi seed
//	kashvi route:list
//
// Or build and run directly:
//
//	go build -o myapp . && ./myapp serve
package app

import (
	"fmt"
	"os"

	"github.com/shashiranjanraj/kashvi/pkg/router"
)

// SeederFunc is a function that seeds the database.
type SeederFunc func()

// global seeders registered via blank-import init() functions.
var globalSeeders []SeederFunc

// RegisterSeeder registers a seeder to be run by `kashvi seed`.
// Call this from an init() in your seeder files.
func RegisterSeeder(name string, fn SeederFunc) {
	globalSeeders = append(globalSeeders, fn)
}

// ─── Application Builder ──────────────────────────────────────────────────────

// Application is the central configuration object for a Kashvi project.
// Build one with New(), attach your configuration, then call Run().
type Application struct {
	routesFns []func(*router.Router)
	models    []interface{}
	seeders   []SeederFunc
}

// New creates a new Application instance with sensible defaults.
func New() *Application {
	return &Application{}
}

// Routes registers a route-registration callback that will be called when
// the HTTP kernel is built. You may call Routes() multiple times; all
// callbacks are executed in order.
func (a *Application) Routes(fn func(*router.Router)) *Application {
	a.routesFns = append(a.routesFns, fn)
	return a
}

// AutoMigrate adds GORM models that will be auto-migrated on server start.
// Pass model pointers: app.New().AutoMigrate(&User{}, &Product{})
func (a *Application) AutoMigrate(models ...interface{}) *Application {
	a.models = append(a.models, models...)
	return a
}

// Seeders registers seeder functions inline (alternative to init()-based
// RegisterSeeder). Can be combined with RegisterSeeder.
func (a *Application) Seeders(fns ...SeederFunc) *Application {
	a.seeders = append(a.seeders, fns...)
	return a
}

// Run reads os.Args and dispatches to the appropriate command.
// This is the ONLY function you need to call from your main().
func (a *Application) Run() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	// Merge globally-registered seeders.
	allSeeders := append(a.seeders, globalSeeders...)

	var err error
	switch cmd {
	case "serve", "start", "run", "s":
		err = cmdServe(a)
	case "migrate":
		err = cmdMigrate()
	case "migrate:rollback", "migrate:down":
		err = cmdMigrateRollback()
	case "migrate:status":
		err = cmdMigrateStatus()
	case "seed":
		err = cmdSeed(allSeeders)
	case "route:list", "routes":
		err = cmdRouteList(a)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %q\n\nRun with --help for usage.\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// ─── Backward-compat free function ────────────────────────────────────────────

// Run is kept for backward compatibility. If you have a plain `app.Run()` call
// in your main.go, it still works — with no custom routes or models.
// Prefer: app.New().Routes(...).AutoMigrate(...).Run()
func Run() {
	New().Run()
}

// ─── Command implementations ──────────────────────────────────────────────────

func printHelp() {
	fmt.Print(`Kashvi — Go Framework CLI

Usage:
  <program> <command>

  (or: kashvi <command>  /  go run . <command>)

Commands:
  serve            Start the HTTP + gRPC server  (aliases: start, run)
  migrate          Run all pending database migrations
  migrate:rollback Rollback the last batch of migrations
  migrate:status   Show migration status
  seed             Run all registered database seeders
  route:list       List registered API routes

`)
}
