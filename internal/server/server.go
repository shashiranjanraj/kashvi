package server

import (
	"fmt"
	"net/http"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/internal/kernel"
	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
)

func Start() error {
	if err := config.Load(); err != nil {
		return err
	}

	database.Connect()
	cache.Connect()

	httpKernel := kernel.NewHTTPKernel()

	fmt.Println("Kashvi running on :8080")
	return http.ListenAndServe(":8080", httpKernel.Handler())
}
