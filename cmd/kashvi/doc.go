// Package cmd/kashvi provides the global Kashvi framework CLI.
//
// Install once globally:
//
//	go install github.com/shashiranjanraj/kashvi/cmd/kashvi@latest
//
// Then from ANY project directory that uses the Kashvi framework:
//
//	kashvi serve           # start server
//	kashvi migrate         # run migrations
//	kashvi migrate:rollback
//	kashvi migrate:status
//	kashvi seed            # seed data
//	kashvi route:list      # list API routes
//
// The CLI detects whether it is running:
//
//	a) Inside the kashvi framework repo itself → uses direct Go imports
//	b) Inside a user project → delegates to `go run . <command>`
//
// User projects just need this in their main.go:
//
//	import "github.com/shashiranjanraj/kashvi/pkg/app"
//	func main() { app.Run() }
package main
