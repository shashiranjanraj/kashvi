package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/app/controllers"
	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/metrics"
	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

// RegisterAPI wires all API routes.
func RegisterAPI(r *router.Router) {
	authCtrl := controllers.NewAuthController()

	// Prometheus metrics endpoint — no auth, no rate limit.
	r.HandleFunc("/metrics", metrics.Handler())

	// Serve local storage files at GET /storage/{path...}
	if config.StorageDefault() == "local" {
		root := config.StorageLocalRoot()
		r.Mount("/storage", http.StripPrefix("/storage", http.FileServer(http.Dir(root))))
	}

	api := r.Group("/api", middleware.RateLimit(120, time.Minute))

	// Public routes
	api.Post("/register", "auth.register", authCtrl.Register)
	api.Post("/login", "auth.login", authCtrl.Login)

	// Health-check — pings DB and Redis, returns 503 if either is down.
	api.Get("/health", "health", healthHandler)

	// Protected routes — require valid JWT
	protected := api.Group("", middleware.AuthMiddleware)
	protected.Get("/profile", "auth.profile", authCtrl.Profile)
	protected.Post("/profile", "auth.profile.update", authCtrl.UpdateProfile)
}

// healthHandler pings the database and Redis, returning a structured status.
// Returns HTTP 200 when all services are healthy, 503 when any are degraded.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	type serviceStatus struct {
		Status  string `json:"status"`
		Latency string `json:"latency,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	type healthResponse struct {
		Status   string                   `json:"status"`
		Services map[string]serviceStatus `json:"services"`
	}

	services := make(map[string]serviceStatus)
	allOK := true

	// ── Database
	if database.DB != nil {
		start := time.Now()
		sqlDB, err := database.DB.DB()
		if err == nil {
			err = sqlDB.PingContext(r.Context())
		}
		latency := time.Since(start)
		if err != nil {
			allOK = false
			services["database"] = serviceStatus{Status: "down", Error: err.Error()}
		} else {
			services["database"] = serviceStatus{Status: "up", Latency: latency.Round(time.Millisecond).String()}
		}
	} else {
		allOK = false
		services["database"] = serviceStatus{Status: "down", Error: "not connected"}
	}

	// ── Redis / Cache
	if cache.RDB != nil {
		start := time.Now()
		err := cache.RDB.Ping(cache.Ctx).Err()
		latency := time.Since(start)
		if err != nil {
			allOK = false
			services["cache"] = serviceStatus{Status: "down", Error: err.Error()}
		} else {
			services["cache"] = serviceStatus{Status: "up", Latency: latency.Round(time.Millisecond).String()}
		}
	} else {
		services["cache"] = serviceStatus{Status: "unavailable"}
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !allOK {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(healthResponse{
		Status:   status,
		Services: services,
	})
}
