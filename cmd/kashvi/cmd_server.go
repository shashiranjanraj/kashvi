package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shashiranjanraj/kashvi/app/routes"
	"github.com/shashiranjanraj/kashvi/internal/server"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

// kashvi run — start the HTTP server.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the HTTP server (alias: serve)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Start()
	},
}

// kashvi route:list — print all registered routes.
var routeListCmd = &cobra.Command{
	Use:   "route:list",
	Short: "List all registered named routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := router.New()
		routes.RegisterAPI(r)

		infos := r.Routes()
		if len(infos) == 0 {
			fmt.Println("No named routes registered.")
			return nil
		}

		// Sort by path then method.
		sort.Slice(infos, func(i, j int) bool {
			if infos[i].Path != infos[j].Path {
				return infos[i].Path < infos[j].Path
			}
			return infos[i].Method < infos[j].Method
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "METHOD\tPATH\tNAME")
		fmt.Fprintln(w, "------\t----\t----")
		for _, ri := range infos {
			fmt.Fprintf(w, "%s\t%s\t%s\n", ri.Method, ri.Path, ri.Name)
		}
		return w.Flush()
	},
}

// kashvi build — compile the server binary.
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the kashvi server binary (outputs ./kashvi)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Building kashvi…")
		c := exec.Command("go", "build", "-o", "kashvi", "./cmd/server")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
		fmt.Println("✅  Built: ./kashvi")
		return nil
	},
}

// kashvi serve — alias kept for muscle memory.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Start()
	},
}
