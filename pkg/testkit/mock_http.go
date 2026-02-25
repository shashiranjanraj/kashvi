package testkit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// ─── MockTransport ────────────────────────────────────────────────────────────

// MockTransport implements http.RoundTripper.
// It matches outgoing HTTP requests against a list of MockSteps from a Scenario
// and returns synthetic responses instead of making real network calls.
//
// Install it on the shared HTTP client before the test:
//
//	mt := testkit.NewMockTransport(scenario)
//	http.DefaultClient.Transport = mt
//	defer http.ResetTransport()
//	// ... run test ...
//	mt.AssertAllCalled(t)
type MockTransport struct {
	mu      sync.Mutex
	steps   []httpMockEntry // only the "httprequest" steps
	require bool            // fail on unmocked call if isMockRequired
}

type httpMockEntry struct {
	step      MockStep
	callCount int
}

// NewMockTransport builds a MockTransport from the "httprequest" steps in s.
// Other mock types (sendmail, etc.) are handled separately by FuncMocker.
func NewMockTransport(s *Scenario) *MockTransport {
	mt := &MockTransport{require: s.IsMockRequired}
	for _, step := range s.NetUtilMockStep {
		if step.Method != "httprequest" {
			continue
		}
		mt.steps = append(mt.steps, httpMockEntry{step: step})
	}
	return mt
}

// RoundTrip intercepts the outgoing request and returns a synthetic response.
func (mt *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	for i := range mt.steps {
		entry := &mt.steps[i]
		if !entry.step.IsMock {
			// pass-through — let the real transport handle it
			break
		}

		if !urlMatches(req.URL.String(), entry.step.MatchURL) {
			continue
		}

		entry.callCount++
		return buildHTTPResponse(req, entry.step.ReturnData)
	}

	if mt.require {
		return nil, fmt.Errorf("testkit: unexpected outgoing HTTP call to %s — no matching mock step", req.URL)
	}

	// No mock found and not required — return a generic 404.
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{"error":"no mock configured"}`)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// AssertAllCalled verifies that every isMock=true step was triggered at least once.
// Call this at the end of each test scenario.
func (mt *MockTransport) AssertAllCalled() []error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	var errs []error
	for _, e := range mt.steps {
		if e.step.IsMock && e.callCount == 0 {
			errs = append(errs, fmt.Errorf(
				"testkit: mock step %q (matchUrl=%q) was never called",
				e.step.Method, e.step.MatchURL,
			))
		}
	}
	return errs
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// urlMatches returns true when candidate matches pattern.
// Empty pattern matches any URL. Otherwise a prefix match is performed.
func urlMatches(candidate, pattern string) bool {
	if pattern == "" {
		return true
	}
	return strings.HasPrefix(candidate, pattern)
}

// buildHTTPResponse creates a synthetic *http.Response from MockReturnData.
// The body field is decoded from base64.
func buildHTTPResponse(req *http.Request, rd MockReturnData) (*http.Response, error) {
	code := rd.StatusCode
	if code == 0 {
		code = http.StatusOK
	}

	var bodyBytes []byte
	if rd.Body != "" {
		decoded, err := base64.StdEncoding.DecodeString(rd.Body)
		if err != nil {
			// Try RawStdEncoding (no padding) as fallback.
			decoded, err = base64.RawStdEncoding.DecodeString(rd.Body)
			if err != nil {
				return nil, fmt.Errorf("testkit: base64 decode mock body: %w", err)
			}
		}
		bodyBytes = decoded
	}

	header := make(http.Header)
	header.Set("Content-Type", "application/json")

	return &http.Response{
		StatusCode: code,
		Status:     fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		Request:    req,
	}, nil
}
