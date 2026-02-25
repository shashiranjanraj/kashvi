// Package main is an example of a minimal project using the Kashvi framework.
//
// To run this example:
//
//	cd /Users/shashi/devlopment/kashvi/example
//	go run . serve
//	# Then: curl http://localhost:8080/hello
package main

import (
	"encoding/json"
	"net/http"

	"github.com/shashiranjanraj/kashvi/pkg/app"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

func main() {
	app.New().
		// Register your routes — get a *router.Router, add your handlers.
		Routes(func(r *router.Router) {
			r.Get("/hello", "hello", helloHandler)
			r.Get("/ping", "ping", pingHandler)
		}).
		// Auto-migrate any GORM models (pass pointer values).
		// AutoMigrate(&User{}).
		Run()
}

// ─── Example Handlers ─────────────────────────────────────────────────────────

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"message": "Hello from Kashvi! 🚀",
		"status":  "ok",
	})
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"pong": "true",
	})
}
