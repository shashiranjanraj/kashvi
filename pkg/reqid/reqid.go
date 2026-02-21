// Package reqid provides request ID generation and context propagation.
//
// A unique ID is generated for every HTTP request, stored in the request
// context, forwarded via the X-Request-ID header, and included in every
// structured log line via logger.WithCtx(ctx).
//
// Middleware wiring in kernel/http.go:
//
//	r.Use(reqid.Middleware())
//
// Reading inside a handler or service:
//
//	id := reqid.FromCtx(r.Context())
//
// Logging with the ID automatically attached:
//
//	log := logger.WithCtx(r.Context())
//	log.Info("user created", "user_id", user.ID)
//	// → time=... level=INFO msg="user created" request_id=abc123 user_id=1
package reqid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// ctxKey is the unexported key used to store the request ID in context.
type ctxKey struct{}

// Header is the HTTP header name used to propagate the request ID.
const Header = "X-Request-ID"

// New generates a cryptographically random 16-byte (32 hex char) request ID.
func New() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// WithValue stores id in ctx and returns the new context.
func WithValue(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromCtx extracts the request ID from ctx.
// Returns an empty string if none is present.
func FromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKey{}).(string); ok {
		return id
	}
	return ""
}

// Middleware injects a unique request ID into every request context and
// response header:
//
//   - If the client sends X-Request-ID, that value is reused (useful for
//     tracing across microservices).
//   - Otherwise a new cryptographically random ID is generated.
//
// The ID is available downstream via reqid.FromCtx(r.Context()).
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Honour an upstream request ID (e.g. from an API gateway or proxy).
			id := r.Header.Get(Header)
			if id == "" {
				id = New()
			}

			// Propagate forward in the response so clients can correlate.
			w.Header().Set(Header, id)

			// Store in context for downstream use.
			ctx := WithValue(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
