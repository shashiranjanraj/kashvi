package routes

import (
	"net/http"

	"github.com/shashiranjanraj/kashvi/app/controllers"
	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

func RegisterAPI(r *router.Router) {
	authController := controllers.NewAuthController()

	api := r.Group("/api")
	api.Post("/login", "auth.login", authController.Login)

	protected := api.Group("", middleware.AuthMiddleware)
	protected.Get("/profile", "auth.profile", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("Protected route ✅"))
	})
}
