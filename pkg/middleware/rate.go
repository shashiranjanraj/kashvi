// Package middleware provides HTTP middleware for the Kashvi framework.
package middleware

import (
	"net/http"
	"sync"
	"time"
)

// bucket tracks a sliding-window request count for one IP.
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

var (
	bucketsMu sync.Mutex
	buckets   = map[string]*bucket{}
)

func init() {
	// Background goroutine: evict buckets whose window has expired.
	// Runs every minute; prevents unbounded memory growth on long-running servers.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			bucketsMu.Lock()
			for ip, b := range buckets {
				b.mu.Lock()
				expired := now.After(b.resetAt)
				b.mu.Unlock()
				if expired {
					delete(buckets, ip)
				}
			}
			bucketsMu.Unlock()
		}
	}()
}

func getBucket(ip string) *bucket {
	bucketsMu.Lock()
	defer bucketsMu.Unlock()

	if b, ok := buckets[ip]; ok {
		return b
	}

	b := &bucket{resetAt: time.Now().Add(time.Minute)}
	buckets[ip] = b
	return b
}

// RateLimit returns a middleware that limits each IP to max requests per window.
// Example: middleware.RateLimit(100, time.Minute)
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = fwd
			}

			if !getBucket(ip).allow(max, window) {
				http.Error(w, `{"status":429,"message":"Too Many Requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
