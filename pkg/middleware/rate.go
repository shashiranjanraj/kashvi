// Package middleware provides HTTP middleware for the Kashvi framework.
package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// bucket tracks a fixed-window request count for one client.
type bucket struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

func (b *bucket) allow(max int, window time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.After(b.resetAt) {
		b.count = 0
		b.resetAt = now.Add(window)
	}

	b.count++
	return b.count <= max
}

// limiter holds the buckets for one RateLimit middleware instance, so routes
// with different limits do not share counters.
type limiter struct {
	max    int
	window time.Duration

	mu        sync.Mutex
	buckets   map[string]*bucket
	nextSweep time.Time
}

func (l *limiter) getBucket(key string) *bucket {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	// Evict expired buckets once per window; prevents unbounded memory growth
	// on long-running servers.
	if now.After(l.nextSweep) {
		for k, b := range l.buckets {
			b.mu.Lock()
			expired := now.After(b.resetAt)
			b.mu.Unlock()
			if expired {
				delete(l.buckets, k)
			}
		}
		l.nextSweep = now.Add(l.window)
	}

	if b, ok := l.buckets[key]; ok {
		return b
	}

	b := &bucket{resetAt: now.Add(l.window)}
	l.buckets[key] = b
	return b
}

// clientIP returns the client address used as the rate-limit key: the first
// X-Forwarded-For entry when present, otherwise RemoteAddr without its port.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		first, _, _ := strings.Cut(fwd, ",")
		return strings.TrimSpace(first)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// RateLimit returns a middleware that limits each IP to max requests per window.
// Example: middleware.RateLimit(100, time.Minute)
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	l := &limiter{max: max, window: window, buckets: map[string]*bucket{}}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.getBucket(clientIP(r)).allow(l.max, l.window) {
				http.Error(w, `{"status":429,"message":"Too Many Requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
