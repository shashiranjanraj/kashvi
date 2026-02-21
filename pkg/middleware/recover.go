package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"github.com/shashiranjanraj/kashvi/pkg/response"
)

// Recovery catches any panic in downstream handlers, logs the stack trace,
// and returns a 500 Internal Server Error to the client.
// Always add this as the innermost middleware (last in the chain) so it wraps
// all other middleware and handlers.
//
//	r.Use(metrics.Middleware())
//	r.Use(reqid.Middleware())
//	r.Use(middleware.Recovery)   // ← catches panics from all below
//	r.Use(middleware.Logger)
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				logger.Error("panic recovered",
					"error", fmt.Sprintf("%v", err),
					"stack", string(stack),
					"method", r.Method,
					"path", r.URL.Path,
				)
				response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
