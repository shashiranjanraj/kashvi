package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shashiranjanraj/kashvi/pkg/router"
	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
)

// BenchmarkRouterRawHandler measures routing + raw http.HandlerFunc (no context).
func BenchmarkRouterRawHandler(b *testing.B) {
	r := router.New()
	r.Get("/", "home", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	h := r.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		rec.Code = 0
		h.ServeHTTP(rec, req)
	}
}

// BenchmarkRouterWithParam measures routing with a path parameter.
func BenchmarkRouterWithParam(b *testing.B) {
	r := router.New()
	r.Get("/users/{id}", "users.show", func(w http.ResponseWriter, r *http.Request) {
		_ = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	h := r.Handler()
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		rec.Code = 0
		h.ServeHTTP(rec, req)
	}
}

// BenchmarkRouterWithCtx measures routing + ctx.Wrap handler (context pool + JSON response).
func BenchmarkRouterWithCtx(b *testing.B) {
	r := router.New()
	r.Get("/", "home", appctx.Wrap(func(c *appctx.Context) {
		c.Success("ok")
	}))
	h := r.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		rec.Code = 0
		h.ServeHTTP(rec, req)
	}
}
