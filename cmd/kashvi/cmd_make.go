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
		content, err := renderStub("model", StubData{Name: name, Lower: strings.ToLower(name)})
		if err != nil {
			return err
		}
		return writeStub(fmt.Sprintf("app/models/%s.go", strings.ToLower(name)), content)
	},
}

var makeControllerCmd = &cobra.Command{
	Use:   "make:controller [Name]",
	Short: "Scaffold a new controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content, err := renderStub("controller", StubData{Name: name, Lower: strings.ToLower(name)})
		if err != nil {
			return err
		}
		return writeStub(fmt.Sprintf("app/controllers/%s.go", strings.ToLower(name)), content)
	},
}

var makeServiceCmd = &cobra.Command{
	Use:   "make:service [Name]",
	Short: "Scaffold a new service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content, err := renderStub("service", StubData{Name: name, Lower: strings.ToLower(name)})
		if err != nil {
			return err
		}
		return writeStub(fmt.Sprintf("app/services/%s.go", strings.ToLower(name)), content)
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
		structName := "M_" + name
		content, err := renderStub("migration", StubData{Name: name, StructName: structName})
		if err != nil {
			return err
		}
		return writeStub(fmt.Sprintf("database/migrations/%s.go", name), content)
	},
}

var makeSeederCmd = &cobra.Command{
	Use:   "make:seeder [Name]",
	Short: "Scaffold a new seeder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content, err := renderStub("seeder", StubData{Name: name, Lower: strings.ToLower(name)})
		if err != nil {
			return err
		}
		return writeStub(fmt.Sprintf("database/seeders/%s.go", strings.ToLower(name)), content)
	},
}

// kashvi make:resource — one command to scaffold a complete CRUD resource.
// Users requested `kashvi make:crud` alias with flags. We update this resource command to match.
var makeResourceCmd = &cobra.Command{
	Use:     "make:resource [Name]",
	Aliases: []string{"make:crud"},
	Short:   "Scaffold a full CRUD resource (model + controller + service + test + migration + seeder)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		lower := strings.ToLower(name)
		ts := time.Now().Format("20060102150405")

		// Parse boolean flags added via init
		authorize, _ := cmd.Flags().GetBool("authorize")
		cache, _ := cmd.Flags().GetBool("cache")

		data := StubData{
			Name:      name,
			Lower:     lower,
			Authorize: authorize,
			Cache:     cache,
		}

		// Pre-render content
		mdl, _ := renderStub("model", data)
		ctrl, _ := renderStub("controller", data)
		svc, _ := renderStub("service", StubData{Name: name + "Service", Lower: lower + "service"})

		migName := fmt.Sprintf("%s_create_%ss_table", ts, lower)
		mig, _ := renderStub("migration", StubData{Name: migName, StructName: "M_" + migName})
		sdr, _ := renderStub("seeder", StubData{Name: name + "Seeder"})

		// Add automated testcase generator
		testScen, _ := renderStub("test_scenario", data)

		type spec struct{ path, content string }
		files := []spec{
			{fmt.Sprintf("app/models/%s.go", lower), mdl},
			{fmt.Sprintf("app/controllers/%s_controller.go", lower), ctrl},
			{fmt.Sprintf("app/services/%s_service.go", lower), svc},
			{fmt.Sprintf("database/migrations/%s.go", migName), mig},
			{fmt.Sprintf("database/seeders/%s_seeder.go", lower), sdr},
			// Test scenarios standard layout
			{fmt.Sprintf("testdata/%s_scenarios.json", lower), testScen},
		}

		for _, f := range files {
			if err := writeStub(f.path, f.content); err != nil {
				return err
			}
		}

		fmt.Printf("\n📋  Add to app/routes/api.go:\n\n")
		fmt.Printf("    ctrl := controllers.New%sController()\n", name)

		middle := ""
		if authorize {
			middle = ", middlewares.Auth()"
		}

		fmt.Printf("    api.Get(\"/%ss\",         \"%s.index\",   ctx.Wrap(ctrl.Index)%s)\n", lower, lower, middle)
		fmt.Printf("    api.Post(\"/%ss\",        \"%s.store\",   ctx.Wrap(ctrl.Store)%s)\n", lower, lower, middle)
		fmt.Printf("    api.Get(\"/%ss/{id}\",    \"%s.show\",    ctx.Wrap(ctrl.Show)%s)\n", lower, lower, middle)
		fmt.Printf("    api.Put(\"/%ss/{id}\",    \"%s.update\",  ctx.Wrap(ctrl.Update)%s)\n", lower, lower, middle)
		fmt.Printf("    api.Delete(\"/%ss/{id}\", \"%s.destroy\", ctx.Wrap(ctrl.Destroy)%s)\n\n", lower, lower, middle)
		return nil
	},
}

func init() {
	makeResourceCmd.Flags().Bool("authorize", false, "Add authentication middleware placeholders")
	makeResourceCmd.Flags().Bool("cache", false, "Add caching mechanisms to generated boilerplate")
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
