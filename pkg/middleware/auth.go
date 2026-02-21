package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/shashiranjanraj/kashvi/pkg/auth"
	"github.com/shashiranjanraj/kashvi/pkg/response"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxRole   ctxKey = "role"
)

// AuthMiddleware validates the Bearer token and injects user_id + role into ctx.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		token := strings.TrimPrefix(raw, "Bearer ")

		if token == "" {
			response.Unauthorized(w)
			return
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			response.Unauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxRole, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromCtx retrieves the authenticated user's ID from the context.
func UserIDFromCtx(r *http.Request) (uint, bool) {
	id, ok := r.Context().Value(ctxUserID).(uint)
	return id, ok
}

// RoleFromCtx retrieves the authenticated user's role from the context.
func RoleFromCtx(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(ctxRole).(string)
	return role, ok
}
