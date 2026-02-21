package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORSOptions configures the CORS middleware.
type CORSOptions struct {
	AllowedOrigins []string // e.g. ["https://app.example.com"] or ["*"]
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int // seconds for preflight cache
}

// DefaultCORSOptions returns permissive options suited for local development.
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		MaxAge:         300,
	}
}

// CORS returns a middleware that adds Cross-Origin Resource Sharing headers.
func CORS(opts CORSOptions) func(http.Handler) http.Handler {
	methods := strings.Join(opts.AllowedMethods, ", ")
	headers := strings.Join(opts.AllowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine if origin is allowed.
			allowed := ""
			for _, o := range opts.AllowedOrigins {
				if o == "*" || o == origin {
					allowed = o
					break
				}
			}

			if allowed != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowed)
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				if opts.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", opts.MaxAge))
				}
			}

			// Handle preflight.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
