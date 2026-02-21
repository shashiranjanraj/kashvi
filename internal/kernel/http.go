package kernel

import (
	"net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/app/models"
	"github.com/shashiranjanraj/kashvi/app/routes"
	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/metrics"
	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/orm"
	"github.com/shashiranjanraj/kashvi/pkg/reqid"
	"github.com/shashiranjanraj/kashvi/pkg/router"
	"github.com/shashiranjanraj/kashvi/pkg/session"
)

// HTTPKernel bootstraps the HTTP layer.
type HTTPKernel struct{}

func NewHTTPKernel() *HTTPKernel {
	return &HTTPKernel{}
}

func (k *HTTPKernel) Handler() http.Handler {
	// Wire cache into ORM (breaks the import cycle).
	orm.CacheStore = &ormCache{}

	// Auto-migrate models on startup.
	if database.DB != nil {
		database.DB.AutoMigrate(
			&models.User{},
			&models.Order{},
			&models.Product{},
		)
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

	routes.RegisterAPI(r)

	return r.Handler()
}

// ormCache bridges pkg/cache.Get/Set to the orm.Cacher interface.
// It lives here (kernel) so neither orm nor cache imports each other.
type ormCache struct{}

func (c *ormCache) Get(key string, dest interface{}) bool {
	return cache.Get(key, dest)
}

func (c *ormCache) Set(key string, value interface{}, ttl time.Duration) error {
	return cache.Set(key, value, ttl)
}
