package middleware

import (
	"net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"github.com/shashiranjanraj/kashvi/pkg/reqid"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger logs each request with method, path, status, duration, IP, and
// the unique request_id injected by reqid.Middleware.
//
// Wire reqid.Middleware() BEFORE this middleware so the ID is available
// in the context when Logger runs.
//
//	r.Use(reqid.Middleware())
//	r.Use(middleware.Logger)
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := reqid.FromCtx(r.Context())

		// Build a per-request logger pre-tagged with the request_id.
		// Every downstream call to logger.WithCtx(ctx) returns this logger.
		reqLog := logger.L.With("request_id", rid)
		ctx := logger.InjectLogger(r.Context(), reqLog)
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		reqLog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start).String(),
			"ip", r.RemoteAddr,
		)
	})
}
