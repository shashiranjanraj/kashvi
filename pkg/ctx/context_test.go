package ctx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
)

func newCtx(method, path, body string) (*appctx.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	var c *appctx.Context
	appctx.Wrap(func(cx *appctx.Context) { c = cx })(rec, req)
	return c, rec
}

func TestWrapAndJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	appctx.Wrap(func(c *appctx.Context) {
		c.JSON(http.StatusOK, map[string]any{"ok": true})
	})(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	appctx.Wrap(func(c *appctx.Context) {
		c.Success(map[string]any{"id": 1})
	})(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSetAndGet(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	appctx.Wrap(func(c *appctx.Context) {
		c.Set("user_id", uint(42))
		uid := c.GetUint("user_id")
		if uid != 42 {
			t.Errorf("expected 42, got %d", uid)
		}
		c.Success(nil)
	})(rec, req)
}

func TestBindJSONValid(t *testing.T) {
	rec := httptest.NewRecorder()
	body := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	appctx.Wrap(func(c *appctx.Context) {
		var input struct {
			Name  string `json:"name"  validate:"required"`
			Email string `json:"email" validate:"required,email"`
		}
		if !c.BindJSON(&input) {
			t.Error("expected BindJSON to succeed")
			return
		}
		if input.Name != "John" {
			t.Errorf("expected John, got %s", input.Name)
		}
		c.Success(nil)
	})(rec, req)

	if rec.Code == http.StatusUnprocessableEntity {
		t.Errorf("unexpected validation failure: %s", rec.Body.String())
	}
}

func TestBindJSONInvalid(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")

	appctx.Wrap(func(c *appctx.Context) {
		var input struct {
			Name string `json:"name" validate:"required"`
		}
		if c.BindJSON(&input) {
			t.Error("expected BindJSON to fail")
		}
	})(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestClientIP(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	appctx.Wrap(func(c *appctx.Context) {
		ip := c.ClientIP()
		if ip != "1.2.3.4" {
			t.Errorf("expected 1.2.3.4, got %s", ip)
		}
		c.Success(nil)
	})(rec, req)
}

func TestErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	appctx.Wrap(func(c *appctx.Context) {
		c.NotFound("Resource missing")
	})(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
