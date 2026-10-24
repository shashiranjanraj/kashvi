// Package middleware provides HTTP middleware for the Kashvi framework.
package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
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

// trustedProxies decides whose X-Forwarded-For header may be believed.
// With no prefixes configured, loopback and private-network addresses are trusted.
type trustedProxies []netip.Prefix

func parseTrustedProxies(entries []string) trustedProxies {
	var out trustedProxies
	for _, e := range entries {
		if p, err := netip.ParsePrefix(e); err == nil {
			out = append(out, p.Masked())
		} else if a, err := netip.ParseAddr(e); err == nil {
			out = append(out, netip.PrefixFrom(a, a.BitLen()))
		} else {
			logger.Warn("rate limit: ignoring invalid TRUSTED_PROXIES entry", "entry", e)
		}
	}
	return out
}

func (t trustedProxies) contains(ip string) bool {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	a = a.Unmap()
	if len(t) == 0 {
		return a.IsLoopback() || a.IsPrivate()
	}
	for _, p := range t {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// clientIP returns the client address used as the rate-limit key. The
// connection address (without port) is used unless it belongs to a trusted
// proxy; then X-Forwarded-For is read right to left, skipping trusted hops.
func clientIP(r *http.Request, trusted trustedProxies) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !trusted.contains(host) {
		return host
	}

	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if hop == "" {
			continue
		}
		if !trusted.contains(hop) || i == 0 {
			return hop
		}
	}
	return host
}

// RateLimit returns a middleware that limits each client IP to max requests per window.
// X-Forwarded-For is only honoured from TRUSTED_PROXIES (default: loopback and
// private networks).
// Example: middleware.RateLimit(100, time.Minute)
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	l := &limiter{max: max, window: window, buckets: map[string]*bucket{}}
	trusted := parseTrustedProxies(config.TrustedProxies())

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.getBucket(clientIP(r, trusted)).allow(l.max, l.window) {
				http.Error(w, `{"status":429,"message":"Too Many Requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
