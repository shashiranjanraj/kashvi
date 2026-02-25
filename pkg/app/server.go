package app

// pkg/app/server.go — bridges Application → internal/server.
// The only job of this file is to build the HTTP handler (via kernel.go)
// and pass it to the internal server that actually binds the port.

import "github.com/shashiranjanraj/kashvi/internal/server"

// startServer builds the HTTP handler from the application config and
// hands it to internal/server.Start for the actual listen+serve lifecycle.
func startServer(a *Application) error {
	handler := buildHandler(a)
	return server.Start(handler)
}
