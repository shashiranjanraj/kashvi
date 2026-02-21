// Package ctx provides a gin.Context-inspired request context for Kashvi handlers.
//
// Instead of accepting (http.ResponseWriter, *http.Request), your handler
// receives a single *Context with helper methods for everything:
//
//	// Before: standard handler
//	func GetUser(w http.ResponseWriter, r *http.Request) {
//	    id := chi.URLParam(r, "id")
//	    json.NewEncoder(w).Encode(...)
//	}
//
//	// After: Kashvi context handler
//	func GetUser(c *ctx.Context) {
//	    id := c.Param("id")
//	    c.JSON(http.StatusOK, user)
//	}
//
//	// Register with ctx.Wrap:
//	router.Get("/users/{id}", "users.show", ctx.Wrap(GetUser))
package ctx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/shashiranjanraj/kashvi/pkg/bind"
	"github.com/shashiranjanraj/kashvi/pkg/validate"
)

// HandlerFunc is the Kashvi context-aware handler signature.
type HandlerFunc func(c *Context)

// Wrap converts a HandlerFunc to a standard http.HandlerFunc so it can be
// passed to any router method.
//
//	router.Get("/users/{id}", "users.show", ctx.Wrap(func(c *ctx.Context) {
//	    c.JSON(200, map[string]any{"id": c.Param("id")})
//	}))
func Wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := acquire(w, r)
		defer release(c)
		h(c)
	}
}

// ─── Context ──────────────────────────────────────────────────────────────────

// Context wraps a request/response pair and provides a rich helper API.
type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	mu     sync.RWMutex
	store  map[string]any
	status int // written status code (0 = not written yet)
}

// pool recycles Context objects to reduce GC pressure.
var pool = sync.Pool{
	New: func() any { return &Context{store: make(map[string]any)} },
}

func acquire(w http.ResponseWriter, r *http.Request) *Context {
	c := pool.Get().(*Context)
	c.W = w
	c.R = r
	c.status = 0
	for k := range c.store {
		delete(c.store, k)
	}
	return c
}

func release(c *Context) {
	c.W = nil
	c.R = nil
	pool.Put(c)
}

// ─── Request helpers ──────────────────────────────────────────────────────────

// Param returns a URL path parameter (e.g. "/users/{id}" → c.Param("id")).
func (c *Context) Param(key string) string {
	return chi.URLParam(c.R, key)
}

// Query returns a query-string value. Returns "" if not present.
func (c *Context) Query(key string) string {
	return c.R.URL.Query().Get(key)
}

// DefaultQuery returns a query-string value, or def if it is empty.
func (c *Context) DefaultQuery(key, def string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	return def
}

// PostForm returns a form field from an application/x-www-form-urlencoded body.
func (c *Context) PostForm(key string) string {
	return c.R.FormValue(key)
}

// Header returns the value of a request header.
func (c *Context) Header(key string) string {
	return c.R.Header.Get(key)
}

// Cookie returns the value of a named cookie.
func (c *Context) Cookie(name string) (string, error) {
	cookie, err := c.R.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// Body reads and returns the raw request body bytes.
// The body can only be read once; use BindJSON for structured data.
func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(c.R.Body)
}

// Method returns the HTTP method of the request.
func (c *Context) Method() string { return c.R.Method }

// Path returns the request URL path.
func (c *Context) Path() string { return c.R.URL.Path }

// FullPath returns method + path (e.g. "GET /api/users").
func (c *Context) FullPath() string {
	return c.R.Method + " " + c.R.URL.Path
}

// ClientIP returns the real client IP, respecting X-Forwarded-For.
func (c *Context) ClientIP() string {
	if fwd := c.R.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.SplitN(fwd, ",", 2)[0]
	}
	if real := c.R.Header.Get("X-Real-Ip"); real != "" {
		return real
	}
	// Strip port from RemoteAddr.
	ip := c.R.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// IsXHR reports whether the request was made via XMLHttpRequest.
func (c *Context) IsXHR() bool {
	return strings.EqualFold(c.R.Header.Get("X-Requested-With"), "XMLHttpRequest")
}

// Context returns the underlying request context.
func (c *Context) Context() context.Context { return c.R.Context() }

// ─── Per-request store ────────────────────────────────────────────────────────

// Set stores a value in the per-request key-value store.
// Useful for passing data between middleware and handlers.
func (c *Context) Set(key string, val any) {
	c.mu.Lock()
	c.store[key] = val
	c.mu.Unlock()
}

// Get retrieves a value from the per-request store.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.store[key]
	c.mu.RUnlock()
	return v, ok
}

// MustGet retrieves a value from the store and panics if the key is absent.
func (c *Context) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic(fmt.Sprintf("ctx: key %q not found in store", key))
	}
	return v
}

// GetString returns a string value from the store, or "" if absent/wrong type.
func (c *Context) GetString(key string) string {
	v, _ := c.Get(key)
	s, _ := v.(string)
	return s
}

// GetUint returns a uint value from the store, or 0 if absent/wrong type.
func (c *Context) GetUint(key string) uint {
	v, _ := c.Get(key)
	u, _ := v.(uint)
	return u
}

// ─── Binding / Validation ─────────────────────────────────────────────────────

// BindJSON decodes the JSON body into dest and runs validation.
// On validation failure it automatically sends a 422 response and returns false.
// On JSON decode error it sends a 400 and returns false.
// Returns true only when dest is valid and ready to use.
//
//	var input RegisterInput
//	if !c.BindJSON(&input) {
//	    return // response already sent
//	}
func (c *Context) BindJSON(dest any) bool {
	errs, err := bind.JSON(c.R, dest)
	if err != nil {
		c.Error(http.StatusBadRequest, err.Error())
		return false
	}
	if validate.HasErrors(errs) {
		c.ValidationError(errs)
		return false
	}
	return true
}

// ShouldBindJSON decodes the JSON body into dest and runs validation.
// Unlike BindJSON, it does NOT write a response — the caller handles errors.
func (c *Context) ShouldBindJSON(dest any) (map[string]string, error) {
	return bind.JSON(c.R, dest)
}

// Validate runs validation rules on an already-populated struct.
// Returns the error map (nil map = no errors).
func (c *Context) Validate(v any) map[string]string {
	return validate.Struct(v)
}

// ─── Response helpers ─────────────────────────────────────────────────────────

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) {
	c.W.Header().Set(key, value)
}

// SetCookie sets a cookie on the response.
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	http.SetCookie(c.W, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

// Status writes just the HTTP status code with an empty body.
func (c *Context) Status(code int) {
	c.status = code
	c.W.WriteHeader(code)
}

// JSON writes a JSON response with the given status code.
func (c *Context) JSON(code int, v any) {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	c.status = code
	json.NewEncoder(c.W).Encode(v) //nolint:errcheck
}

// Success sends a 200 JSON envelope: {"status":200,"data":...}
func (c *Context) Success(data any) {
	c.JSON(http.StatusOK, envelope{Status: http.StatusOK, Data: data})
}

// Created sends a 201 JSON envelope.
func (c *Context) Created(data any) {
	c.JSON(http.StatusCreated, envelope{Status: http.StatusCreated, Data: data})
}

// Error sends a JSON error envelope with the given status and message.
func (c *Context) Error(code int, message string) {
	c.JSON(code, envelope{Status: code, Message: message})
}

// ValidationError sends a 422 Unprocessable Entity with field-level errors.
func (c *Context) ValidationError(errs map[string]string) {
	c.JSON(http.StatusUnprocessableEntity, envelope{
		Status:  http.StatusUnprocessableEntity,
		Message: "Validation failed",
		Errors:  errs,
	})
}

// Unauthorized sends a 401.
func (c *Context) Unauthorized(message ...string) {
	msg := "Unauthorized"
	if len(message) > 0 {
		msg = message[0]
	}
	c.Error(http.StatusUnauthorized, msg)
}

// Forbidden sends a 403.
func (c *Context) Forbidden(message ...string) {
	msg := "Forbidden"
	if len(message) > 0 {
		msg = message[0]
	}
	c.Error(http.StatusForbidden, msg)
}

// NotFound sends a 404.
func (c *Context) NotFound(message ...string) {
	msg := "Not found"
	if len(message) > 0 {
		msg = message[0]
	}
	c.Error(http.StatusNotFound, msg)
}

// String writes a plain-text response.
func (c *Context) String(code int, format string, args ...any) {
	c.W.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.W.WriteHeader(code)
	c.status = code
	fmt.Fprintf(c.W, format, args...)
}

// Redirect sends an HTTP redirect response.
func (c *Context) Redirect(code int, url string) {
	http.Redirect(c.W, c.R, url, code)
}

// File serves a file from the local filesystem.
func (c *Context) File(filepath string) {
	http.ServeFile(c.W, c.R, filepath)
}

// Abort sends an error response. By convention, the handler should return
// immediately after calling Abort.
func (c *Context) Abort(code int, message string) {
	c.Error(code, message)
}

// WrittenStatus returns the HTTP status code that was written to the response,
// or 0 if no response has been written yet.
func (c *Context) WrittenStatus() int { return c.status }

// ─── JSON envelope (mirrors pkg/response) ────────────────────────────────────

type envelope struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}
