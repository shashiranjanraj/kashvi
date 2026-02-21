package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ─── Scaffold commands ────────────────────────────────────────────────────────

var makeModelCmd = &cobra.Command{
	Use:   "make:model [Name]",
	Short: "Scaffold a new model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return writeStub(fmt.Sprintf("app/models/%s.go", strings.ToLower(name)), modelStub(name))
	},
}

var makeControllerCmd = &cobra.Command{
	Use:   "make:controller [Name]",
	Short: "Scaffold a new controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return writeStub(fmt.Sprintf("app/controllers/%s.go", strings.ToLower(name)), controllerStub(name))
	},
}

var makeServiceCmd = &cobra.Command{
	Use:   "make:service [Name]",
	Short: "Scaffold a new service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return writeStub(fmt.Sprintf("app/services/%s.go", strings.ToLower(name)), serviceStub(name))
	},
}

var makeMigrationCmd = &cobra.Command{
	Use:   "make:migration [name]",
	Short: "Create a new migration file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ts := time.Now().Format("20060102150405")
		slug := strings.ToLower(strings.ReplaceAll(args[0], " ", "_"))
		name := fmt.Sprintf("%s_%s", ts, slug)
		return writeStub(fmt.Sprintf("database/migrations/%s.go", name), migrationStub(name))
	},
}

var makeSeederCmd = &cobra.Command{
	Use:   "make:seeder [Name]",
	Short: "Scaffold a new seeder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return writeStub(fmt.Sprintf("database/seeders/%s.go", strings.ToLower(name)), seederStub(name))
	},
}

// kashvi make:resource — one command to scaffold a complete CRUD resource.
var makeResourceCmd = &cobra.Command{
	Use:   "make:resource [Name]",
	Short: "Scaffold a full CRUD resource (model + controller + migration + seeder)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		lower := strings.ToLower(name)
		ts := time.Now().Format("20060102150405")

		type spec struct{ path, content string }
		files := []spec{
			{fmt.Sprintf("app/models/%s.go", lower), modelStub(name)},
			{fmt.Sprintf("app/controllers/%s_controller.go", lower), resourceControllerStub(name)},
			{fmt.Sprintf("app/services/%s_service.go", lower), serviceStub(name + "Service")},
			{fmt.Sprintf("database/migrations/%s_create_%ss_table.go", ts, lower),
				migrationStub(fmt.Sprintf("%s_create_%ss_table", ts, lower))},
			{fmt.Sprintf("database/seeders/%s_seeder.go", lower), seederStub(name + "Seeder")},
		}
		for _, f := range files {
			if err := writeStub(f.path, f.content); err != nil {
				return err
			}
		}

		fmt.Printf("\n📋  Add to app/routes/api.go:\n\n")
		fmt.Printf("    ctrl := controllers.New%sController()\n", name)
		fmt.Printf("    api.Get(\"/%ss\",         \"%s.index\",   ctx.Wrap(ctrl.Index))\n", lower, lower)
		fmt.Printf("    api.Post(\"/%ss\",        \"%s.store\",   ctx.Wrap(ctrl.Store))\n", lower, lower)
		fmt.Printf("    api.Get(\"/%ss/{id}\",    \"%s.show\",    ctx.Wrap(ctrl.Show))\n", lower, lower)
		fmt.Printf("    api.Put(\"/%ss/{id}\",    \"%s.update\",  ctx.Wrap(ctrl.Update))\n", lower, lower)
		fmt.Printf("    api.Delete(\"/%ss/{id}\", \"%s.destroy\", ctx.Wrap(ctrl.Destroy))\n\n", lower, lower)
		return nil
	},
}

// ─── writeStub ────────────────────────────────────────────────────────────────

func writeStub(path, content string) error {
	dir := path[:strings.LastIndex(path, "/")]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("✅  Created: %s\n", path)
	return nil
}

// ─── Stubs ────────────────────────────────────────────────────────────────────

func modelStub(name string) string {
	return fmt.Sprintf(`package models

import "gorm.io/gorm"

type %s struct {
	gorm.Model
}
`, name)
}

func controllerStub(name string) string {
	return fmt.Sprintf(`package controllers

import "net/http"

type %s struct{}

func New%s() *%s { return &%s{} }

func (c *%s) Index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("TODO: %s.Index"))
}
`, name, name, name, name, name, name)
}

func resourceControllerStub(name string) string {
	lower := strings.ToLower(name)
	return fmt.Sprintf(`package controllers

import (
	"net/http"

	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
)

type %sController struct{}

func New%sController() *%sController { return &%sController{} }

// GET /%ss
func (c *%sController) Index(ctx *appctx.Context) {
	ctx.Success([]map[string]any{})
}

// POST /%ss
func (c *%sController) Store(ctx *appctx.Context) {
	var input struct{}
	if !ctx.BindJSON(&input) { return }
	ctx.Created(map[string]any{"message": "%s created"})
}

// GET /%ss/{id}
func (c *%sController) Show(ctx *appctx.Context) {
	ctx.Success(map[string]any{"id": ctx.Param("id")})
}

// PUT /%ss/{id}
func (c *%sController) Update(ctx *appctx.Context) {
	var input struct{}
	if !ctx.BindJSON(&input) { return }
	ctx.Success(map[string]any{"id": ctx.Param("id"), "updated": true})
}

// DELETE /%ss/{id}
func (c *%sController) Destroy(ctx *appctx.Context) {
	ctx.Status(http.StatusNoContent)
}
`,
		name, name, name, name,
		lower, name,
		lower, name, name,
		lower, name,
		lower, name,
		lower, name,
	)
}

func serviceStub(name string) string {
	return fmt.Sprintf(`package services

type %s struct{}

func New%s() *%s { return &%s{} }
`, name, name, name, name)
}

func migrationStub(name string) string {
	structName := "M_" + name
	return fmt.Sprintf(`package migrations

import (
	"github.com/shashiranjanraj/kashvi/pkg/migration"
	"gorm.io/gorm"
)

func init() { migration.Register("%s", &%s{}) }

type %s struct{}

func (m *%s) Up(db *gorm.DB) error   { return nil }
func (m *%s) Down(db *gorm.DB) error { return nil }
`, name, structName, structName, structName, structName)
}

func seederStub(name string) string {
	return fmt.Sprintf(`package seeders

import "gorm.io/gorm"

func %s(db *gorm.DB) error { return nil }
`, name)
}
