// Package metrics provides Prometheus instrumentation for Kashvi.
//
// It pre-defines the standard HTTP metrics that every app needs and gives
// you helpers to register your own custom metrics.
//
// Wire it up once in internal/kernel/http.go:
//
//	r.Use(metrics.Middleware())
//	r.Get("/metrics", "metrics", metrics.Handler())
//
// Then scrape http://localhost:8080/metrics from Prometheus.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ─────────────────────────────────────────────
// Built-in HTTP metrics
// ─────────────────────────────────────────────

var (
	// RequestDuration tracks how long each HTTP request takes,
	// broken down by method, route path, and status code.
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kashvi",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds.",
			Buckets:   prometheus.DefBuckets, // .005 .01 .025 .05 .1 .25 .5 1 2.5 5 10
		},
		[]string{"method", "path", "status"},
	)

	// RequestTotal counts all HTTP requests.
	RequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kashvi",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	// RequestInFlight tracks how many requests are currently being served.
	RequestInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "kashvi",
		Subsystem: "http",
		Name:      "requests_in_flight",
		Help:      "Number of HTTP requests currently being served.",
	})

	// ResponseSize tracks the response body size in bytes.
	ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kashvi",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "Response body sizes in bytes.",
			Buckets:   []float64{100, 1_000, 10_000, 100_000, 1_000_000},
		},
		[]string{"method", "path"},
	)

	// DBQueryDuration tracks ORM query latency.
	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kashvi",
			Subsystem: "db",
			Name:      "query_duration_seconds",
			Help:      "Duration of database queries in seconds.",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .5, 1},
		},
		[]string{"operation"}, // "select" | "insert" | "update" | "delete"
	)

	// QueueJobsProcessed counts processed queue jobs by status.
	QueueJobsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kashvi",
			Subsystem: "queue",
			Name:      "jobs_processed_total",
			Help:      "Total queue jobs processed.",
		},
		[]string{"status"}, // "success" | "failed"
	)

	// QueueJobDuration tracks how long queue jobs take.
	QueueJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kashvi",
			Subsystem: "queue",
			Name:      "job_duration_seconds",
			Help:      "Duration of queue job processing in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"job_type"},
	)

	// CacheHits / CacheMisses track cache effectiveness.
	CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kashvi",
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Total cache hits.",
		},
		[]string{"driver"}, // "redis" | "memory"
	)
	CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kashvi",
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Total cache misses.",
		},
		[]string{"driver"},
	)
)

// ─────────────────────────────────────────────
// Registry
// ─────────────────────────────────────────────

// DefaultRegistry is the Prometheus registry used by Kashvi.
// Register your own metrics against this.
var DefaultRegistry = prometheus.NewRegistry()

func init() {
	// Go runtime metrics (GC, goroutines, memory)
	DefaultRegistry.MustRegister(collectors.NewGoCollector())
	// OS process metrics (CPU, open FDs)
	DefaultRegistry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Kashvi built-in metrics
	DefaultRegistry.MustRegister(
		RequestDuration,
		RequestTotal,
		RequestInFlight,
		ResponseSize,
		DBQueryDuration,
		QueueJobsProcessed,
		QueueJobDuration,
		CacheHits,
		CacheMisses,
	)
}

// Register lets you add your own prometheus.Collector to the Kashvi registry.
func Register(c prometheus.Collector) error {
	return DefaultRegistry.Register(c)
}

// MustRegister panics if registration fails.
func MustRegister(c ...prometheus.Collector) {
	DefaultRegistry.MustRegister(c...)
}

// ─────────────────────────────────────────────
// Custom metric constructors
// ─────────────────────────────────────────────

// NewCounter creates and registers a Counter with the given name and labels.
func NewCounter(namespace, name, help string, labels []string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
	}, labels)
	DefaultRegistry.MustRegister(c)
	return c
}

// NewHistogram creates and registers a Histogram with the given name and labels.
func NewHistogram(namespace, name, help string, buckets []float64, labels []string) *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labels)
	DefaultRegistry.MustRegister(h)
	return h
}

// NewGauge creates and registers a Gauge.
func NewGauge(namespace, name, help string, labels []string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
	}, labels)
	DefaultRegistry.MustRegister(g)
	return g
}

// ─────────────────────────────────────────────
// HTTP middleware
// ─────────────────────────────────────────────

// responseRecorder wraps http.ResponseWriter to capture status code and size.
type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// Middleware returns an http.Handler middleware that records Prometheus metrics
// for every request: duration histogram, total counter, in-flight gauge, response size.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := r.URL.Path // raw path; normalize in high-cardinality APIs

			RequestInFlight.Inc()
			defer RequestInFlight.Dec()

			rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rr, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rr.status)

			RequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
			RequestTotal.WithLabelValues(r.Method, path, status).Inc()
			ResponseSize.WithLabelValues(r.Method, path).Observe(float64(rr.size))
		})
	}
}

// ─────────────────────────────────────────────
// /metrics endpoint handler
// ─────────────────────────────────────────────

// Handler returns an http.HandlerFunc that exposes the Prometheus metrics page.
// Mount it on GET /metrics in your router.
func Handler() http.HandlerFunc {
	h := promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true, // enables text/plain AND OpenMetrics formats
	})
	return h.ServeHTTP
}

// ─────────────────────────────────────────────
// Helpers for app code
// ─────────────────────────────────────────────

// ObserveDBQuery records a DB query duration with a simple timer:
//
//	defer metrics.ObserveDBQuery("select", time.Now())
func ObserveDBQuery(operation string, start time.Time) {
	DBQueryDuration.WithLabelValues(operation).Observe(time.Since(start).Seconds())
}

// RecordQueueJob records a queue job result.
func RecordQueueJob(jobType, status string, start time.Time) {
	QueueJobsProcessed.WithLabelValues(status).Inc()
	QueueJobDuration.WithLabelValues(jobType).Observe(time.Since(start).Seconds())
}
