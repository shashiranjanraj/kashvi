package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shashiranjanraj/kashvi/config"
	kashvigrpc "github.com/shashiranjanraj/kashvi/pkg/grpc"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
)

// kashvi run — in framework-self mode, delegate to go run . serve
// (the framework repo itself also has a sample app, so the CLI
// in project-mode delegates via addProjectDelegateCmds, and in
// framework-self mode this prints guidance rather than hard-coding routes).
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the HTTP server (alias: serve)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInProject("serve")
	},
}

// kashvi route:list — in project mode this delegates; in framework-self mode
// it just explains that routes come from the user project.
var routeListCmd = &cobra.Command{
	Use:   "route:list",
	Short: "List all registered named routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isFrameworkSelf() {
			fmt.Println("route:list requires your project's app.New().Routes(...) to be registered.")
			fmt.Println("Run from a project directory:  kashvi route:list")
			return nil
		}
		return runInProject("route:list")
	},
}

// printRouteTable is a helper used internally when routes are available.
func printRouteTable(infos []routeInfo) {
	if len(infos) == 0 {
		fmt.Println("No named routes registered.")
		return
	}
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
	w.Flush() //nolint:errcheck
}

type routeInfo struct{ Method, Path, Name string }

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

// kashvi serve — alias kept for muscle memory (delegates in project mode).
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInProject("serve")
	},
}

// kashvi grpc:serve — start gRPC server standalone.
var grpcServeCmd = &cobra.Command{
	Use:   "grpc:serve",
	Short: "Start the gRPC server only (health-check + reflection)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Load(); err != nil {
			return err
		}

		grpcSrv, _, err := kashvigrpc.Start(config.GRPCPort())
		if err != nil {
			return err
		}
		fmt.Printf("🔌 gRPC server running on :%s\n", config.GRPCPort())

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("shutting down gRPC server…")
		kashvigrpc.Stop(grpcSrv)
		logger.CloseMongoHandler()
		return nil
	},
}
