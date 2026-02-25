package app

// pkg/app/kernel.go — builds an http.Handler from the Application config.
// This file has NO imports of project-specific code (models, routes, etc).
// All project dependencies are injected via the Application builder methods.

import (
	"net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/metrics"
	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/orm"
	"github.com/shashiranjanraj/kashvi/pkg/reqid"
	"github.com/shashiranjanraj/kashvi/pkg/router"
	"github.com/shashiranjanraj/kashvi/pkg/session"
)

// buildHandler constructs the HTTP handler from the Application config.
// This is pure framework code — it sets up global middleware, runs
// auto-migrations, then calls the user's route-registration callbacks.
func buildHandler(a *Application) http.Handler {
	// Wire cache into ORM (breaks the import cycle).
	orm.CacheStore = &ormCache{}

	// Auto-migrate user-supplied models (if DB is available).
	if database.DB != nil && len(a.models) > 0 {
		database.DB.AutoMigrate(a.models...)
	}

	r := router.New()

	// Global middleware stack (outermost → innermost):
	//  1. Prometheus metrics — outermost for accurate total latency
	//  2. Recovery          — catches panics before they kill the goroutine
	//  3. Request ID        — inject unique ID before anything logs
	//  4. Logger            — logs request_id from context
	//  5. Session           — load/create session cookie via Redis
	//  6. CORS              — set CORS headers
	//  7. Rate limiter      — reject abusers early
	r.Use(metrics.Middleware())
	r.Use(middleware.Recovery)
	r.Use(reqid.Middleware())
	r.Use(middleware.Logger)
	r.Use(session.Middleware(session.DefaultOptions()))
	r.Use(middleware.CORS(middleware.DefaultCORSOptions()))
	r.Use(middleware.RateLimit(200, time.Minute))

	// Prometheus /metrics endpoint — no auth, no rate limit.
	r.HandleFunc("/metrics", metrics.Handler())

	// Call every route-registration callback the user supplied.
	for _, fn := range a.routesFns {
		fn(r)
	}

	return r.Handler()
}

// ormCache bridges pkg/cache.Get/Set to the orm.Cacher interface.
// Lives here so neither orm nor cache imports each other.
type ormCache struct{}

func (c *ormCache) Get(key string, dest interface{}) bool {
	return cache.Get(key, dest)
}

func (c *ormCache) Set(key string, value interface{}, ttl time.Duration) error {
	return cache.Set(key, value, ttl)
}
