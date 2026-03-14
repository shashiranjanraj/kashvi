package ctx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
)

// BenchmarkWrap measures ctx.Wrap handler: pool acquire, handler run, pool release, JSON response.
func BenchmarkWrap(b *testing.B) {
	h := appctx.Wrap(func(c *appctx.Context) {
		c.Success(map[string]string{"status": "ok"})
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		rec.Code = 0
		h.ServeHTTP(rec, req)
	}
}

// BenchmarkWrapWithParam measures context handler that reads a URL param.
func BenchmarkWrapWithParam(b *testing.B) {
	h := appctx.Wrap(func(c *appctx.Context) {
		_ = c.Param("id")
		c.Success("ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	// Chi would set URL params; without the router we only get path. Param("id") will be empty.
	// This still benchmarks context acquire/release + Success.
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		rec.Code = 0
		h.ServeHTTP(rec, req)
	}
}
