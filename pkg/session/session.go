// Package session provides HTTP session management backed by Redis (or memory).
//
// Usage (middleware):
//
//	r.Use(session.Middleware(session.DefaultOptions()))
//
// Usage (handler):
//
//	sess := session.FromCtx(r)
//	sess.Set("user_id", 42)
//	sess.Save(w)
//	val, _ := sess.Get("user_id")
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/cache"
)

// ------------------- Options -------------------

// Options configures session behaviour.
type Options struct {
	CookieName string
	TTL        time.Duration
	HTTPOnly   bool
	Secure     bool
	SameSite   http.SameSite
	Path       string
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		CookieName: "kashvi_session",
		TTL:        2 * time.Hour,
		HTTPOnly:   true,
		Secure:     false, // set true in production
		SameSite:   http.SameSiteLaxMode,
		Path:       "/",
	}
}

// ------------------- Session -------------------

type ctxKey struct{}

// Session is an in-request session handle.
type Session struct {
	id      string
	data    map[string]interface{}
	opts    Options
	changed bool
}

// newID generates a cryptographically random 32-byte hex session ID.
func newID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func redisKey(id string) string { return "kashvi:session:" + id }

// load fetches session data from Redis.
func load(id string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if cache.Get(redisKey(id), &data) {
		return data, nil
	}
	return map[string]interface{}{}, nil
}

// Set stores a value under key in the session.
func (s *Session) Set(key string, value interface{}) {
	s.data[key] = value
	s.changed = true
}

// Get retrieves a value from the session.
func (s *Session) Get(key string) (interface{}, bool) {
	v, ok := s.data[key]
	return v, ok
}

// GetString is a typed convenience getter.
func (s *Session) GetString(key string) (string, bool) {
	v, ok := s.data[key]
	if !ok {
		return "", false
	}
	s2, ok := v.(string)
	return s2, ok
}

// GetInt is a typed convenience getter.
func (s *Session) GetInt(key string) (int, bool) {
	v, ok := s.data[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64: // JSON numbers unmarshal as float64
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

// Delete removes a key from the session.
func (s *Session) Delete(key string) {
	delete(s.data, key)
	s.changed = true
}

// Flash stores a value that is auto-deleted after the next Get.
func (s *Session) Flash(key string, value interface{}) {
	s.Set("_flash_"+key, value)
}

// GetFlash retrieves and removes a flash value.
func (s *Session) GetFlash(key string) (interface{}, bool) {
	v, ok := s.Get("_flash_" + key)
	if ok {
		s.Delete("_flash_" + key)
	}
	return v, ok
}

// Invalidate destroys the session (logout).
func (s *Session) Invalidate() {
	s.data = map[string]interface{}{}
	s.changed = true
}

// ID returns the session ID.
func (s *Session) ID() string { return s.id }

// Save persists the session to Redis and writes the cookie to the response.
func (s *Session) Save(w http.ResponseWriter) error {
	if !s.changed {
		return nil
	}

	raw, err := json.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}

	if err := cache.Set(redisKey(s.id), json.RawMessage(raw), s.opts.TTL); err != nil {
		return fmt.Errorf("session: redis save: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.opts.CookieName,
		Value:    s.id,
		Path:     s.opts.Path,
		MaxAge:   int(s.opts.TTL.Seconds()),
		HttpOnly: s.opts.HTTPOnly,
		Secure:   s.opts.Secure,
		SameSite: s.opts.SameSite,
	})

	s.changed = false
	return nil
}

// ------------------- Middleware -------------------

// Middleware loads (or creates) the session for every request and injects it
// into the request context. Handlers call session.FromCtx(r) to access it.
func Middleware(opts Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := &Session{opts: opts}

			if cookie, err := r.Cookie(opts.CookieName); err == nil {
				sess.id = cookie.Value
				sess.data, _ = load(sess.id)
			} else {
				id, _ := newID()
				sess.id = id
				sess.data = map[string]interface{}{}
			}

			ctx := context.WithValue(r.Context(), ctxKey{}, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromCtx retrieves the session from the request context.
// Returns an empty (unsaved) session if none is present.
func FromCtx(r *http.Request) *Session {
	if s, ok := r.Context().Value(ctxKey{}).(*Session); ok {
		return s
	}
	id, _ := newID()
	return &Session{id: id, data: map[string]interface{}{}, opts: DefaultOptions()}
}
