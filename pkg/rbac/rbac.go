// Package rbac provides role-based access control middleware for Kashvi.
package rbac

import (
	"net/http"

	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/response"
)

// HasRole returns middleware that allows access only to users with the given role.
// Requires AuthMiddleware to have already run (role must be in context).
func HasRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := middleware.RoleFromCtx(r)
			if !ok || !allowed[role] {
				response.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Guest returns middleware that blocks authenticated users (useful for login/register).
func Guest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := middleware.UserIDFromCtx(r); ok {
			response.Error(w, http.StatusConflict, "Already authenticated")
			return
		}
		next.ServeHTTP(w, r)
	})
}
