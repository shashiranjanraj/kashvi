package main

// cmd/server/main.go is the entry point for when a user builds their own
// project binary. It is NOT the global kashvi CLI binary.
//
// For user projects, replace this file with:
//
//	package main
//
//	import (
//	    "github.com/shashiranjanraj/kashvi/pkg/app"
//	    _ "yourproject/database/migrations"
//	    _ "yourproject/database/seeders"
//	)
//
//	func main() {
//	    app.New().Routes(myRoutes).AutoMigrate(&User{}).Run()
//	}
//
// This file exists only as a fallback for the framework repository itself.

import (
	"log"

	"github.com/shashiranjanraj/kashvi/pkg/app"
)

func main() {
	if err := runApp(); err != nil {
		log.Fatal(err)
	}
}

func runApp() error {
	done := make(chan error, 1)
	go func() {
		// app.New().Run() calls os.Exit on error, so we wrap it.
		app.New().Run()
		done <- nil
	}()
	return <-done
}
