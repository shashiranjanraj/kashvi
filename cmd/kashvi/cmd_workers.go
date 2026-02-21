package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/shashiranjanraj/kashvi/pkg/queue"
	"github.com/shashiranjanraj/kashvi/pkg/schedule"
)

var queueWorkersFlag int

// kashvi queue:work
var queueWorkCmd = &cobra.Command{
	Use:   "queue:work",
	Short: "Start the queue worker",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		workers := queueWorkersFlag
		if workers < 1 {
			workers = 5
		}

		fmt.Printf("🚀 Queue worker started (%d workers). Press Ctrl+C to stop.\n", workers)
		queue.StartWorkers(ctx, workers)

		<-ctx.Done()
		fmt.Println("\n⚡ Queue worker stopped.")
		return nil
	},
}

// kashvi schedule:run
var scheduleRunCmd = &cobra.Command{
	Use:   "schedule:run",
	Short: "Start the task scheduler",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		tasks := schedule.List()
		if len(tasks) == 0 {
			fmt.Println("No scheduled tasks registered.")
		} else {
			fmt.Println("Registered scheduled tasks:")
			for _, t := range tasks {
				fmt.Println("  •", t)
			}
		}

		fmt.Println("🕐 Scheduler started. Press Ctrl+C to stop.")
		schedule.Start(ctx)

		<-ctx.Done()
		fmt.Println("\n⚡ Scheduler stopped.")
		return nil
	},
}

func init() {
	queueWorkCmd.Flags().IntVarP(&queueWorkersFlag, "workers", "w", 5, "Number of concurrent workers")
}
