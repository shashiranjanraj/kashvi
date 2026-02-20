package middleware

import (
	"net/http"
	"strings"

	"github.com/shashiranjanraj/kashvi/pkg/auth"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)

		if token == "" {
			http.Error(w, "Unauthorized", 401)
			return
		}

		_, err := auth.ValidateToken(token)
		if err != nil {
			http.Error(w, "Invalid token", 401)
			return
		}

		next.ServeHTTP(w, r)
	})
}
