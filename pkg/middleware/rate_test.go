package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func doRequest(h http.Handler, remoteAddr, xff string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func TestRateLimitIgnoresClientPort(t *testing.T) {
	h := RateLimit(2, time.Minute)(okHandler)

	for i, addr := range []string{"10.0.0.1:1000", "10.0.0.1:1001"} {
		if code := doRequest(h, addr, ""); code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, code)
		}
	}
	if code := doRequest(h, "10.0.0.1:1002", ""); code != http.StatusTooManyRequests {
		t.Fatalf("third request from same IP: got %d, want 429", code)
	}
	if code := doRequest(h, "10.0.0.2:1000", ""); code != http.StatusOK {
		t.Fatalf("other IP: got %d, want 200", code)
	}
}

func TestRateLimitForwardedFor(t *testing.T) {
	h := RateLimit(1, time.Minute)(okHandler)

	if code := doRequest(h, "127.0.0.1:1", "203.0.113.5, 10.0.0.1"); code != http.StatusOK {
		t.Fatalf("got %d, want 200", code)
	}
	if code := doRequest(h, "127.0.0.1:2", "203.0.113.5"); code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", code)
	}
}

func TestRateLimitInstancesAreIndependent(t *testing.T) {
	a := RateLimit(1, time.Minute)(okHandler)
	b := RateLimit(1, time.Minute)(okHandler)

	if code := doRequest(a, "10.0.0.1:1", ""); code != http.StatusOK {
		t.Fatalf("a: got %d, want 200", code)
	}
	if code := doRequest(b, "10.0.0.1:1", ""); code != http.StatusOK {
		t.Fatalf("b: got %d, want 200", code)
	}
}

func TestRateLimitWindowResets(t *testing.T) {
	h := RateLimit(1, 50*time.Millisecond)(okHandler)

	doRequest(h, "10.0.0.1:1", "")
	if code := doRequest(h, "10.0.0.1:1", ""); code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", code)
	}
	time.Sleep(60 * time.Millisecond)
	if code := doRequest(h, "10.0.0.1:1", ""); code != http.StatusOK {
		t.Fatalf("after window: got %d, want 200", code)
	}
}

func TestRateLimitIgnoresSpoofedForwardedFor(t *testing.T) {
	h := RateLimit(1, time.Minute)(okHandler)

	// A direct (public) client cannot pick a fresh key by changing X-Forwarded-For.
	if code := doRequest(h, "198.51.100.7:1", "1.1.1.1"); code != http.StatusOK {
		t.Fatalf("got %d, want 200", code)
	}
	if code := doRequest(h, "198.51.100.7:2", "2.2.2.2"); code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", code)
	}
}

func TestClientIPTrustedProxies(t *testing.T) {
	trusted := parseTrustedProxies([]string{"198.51.100.0/24", "203.0.113.9"})
	cases := []struct {
		remote, xff, want string
	}{
		{"198.51.100.7:1", "192.0.2.1, 203.0.113.9", "192.0.2.1"}, // skips trusted hops
		{"198.51.100.7:1", "6.6.6.6, 192.0.2.1", "192.0.2.1"},     // rightmost untrusted wins
		{"192.0.2.50:1", "192.0.2.1", "192.0.2.50"},               // untrusted peer: header ignored
		{"10.0.0.1:1", "192.0.2.1", "10.0.0.1"},                   // private not trusted when list given
		{"198.51.100.7:1", "", "198.51.100.7"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = c.remote
		if c.xff != "" {
			req.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := clientIP(req, trusted); got != c.want {
			t.Errorf("clientIP(%s, %q) = %s, want %s", c.remote, c.xff, got, c.want)
		}
	}
}
