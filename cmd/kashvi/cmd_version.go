package main

import (
	"fmt"

	"github.com/shashiranjanraj/kashvi/pkg/app"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the framework version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kashvi Framework v%s\n", app.Version)
	},
}
