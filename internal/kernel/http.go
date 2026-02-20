package kernel

import (
	"net/http"

	"github.com/shashiranjanraj/kashvi/app/routes"
	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

type HTTPKernel struct{}

func NewHTTPKernel() *HTTPKernel {
	return &HTTPKernel{}
}

func (k *HTTPKernel) Handler() http.Handler {
	r := router.New()
	r.Use(middleware.Logger)
	routes.RegisterAPI(r)

	return r.Handler()
}
