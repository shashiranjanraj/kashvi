package app

// pkg/app/server.go — bridges Application → internal/server.
// The only job of this file is to build the HTTP handler (via kernel.go)
// and pass it to the internal server that actually binds the port.

import (
	"fmt"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/internal/server"
)

// startServer builds the HTTP handler from the application config and
// hands it to internal/server.Start for the actual listen+serve lifecycle.
// DB is connected before buildHandler so that kernel can run AutoMigrate.
func startServer(a *Application) error {
	if err := config.Load(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := database.Connect(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	handler := buildHandler(a)
	return server.Start(handler)
}
