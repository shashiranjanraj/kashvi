// Package testkit — runner.go
//
// Run() executes a single scenario against an http.Handler.
// RunDir() discovers all *.json files in a directory and runs them as subtests.
package testkit

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kashvihttp "github.com/shashiranjanraj/kashvi/pkg/http"
)

// ─── Public API ───────────────────────────────────────────────────────────────

// Run executes a single scenario from a JSON file against the provided handler.
//
// Lifecycle per scenario:
//  1. Load the scenario JSON file.
//  2. Read request body from requestFileName (if set).
//  3. Install HTTP mock transport on Kashvi's HTTP client.
//  4. Activate function mocks (sendmail, sms, …).
//  5. Fire the request against handler using httptest.
//  6. Assert status code.
//  7. Assert response body (JSON diff) against responseFileName (if set).
//  8. Verify all isMock=true steps were called.
//  9. Reset all mocks.
func Run(t *testing.T, handler http.Handler, scenarioPath string) {
	t.Helper()

	s, err := LoadScenario(scenarioPath)
	if err != nil {
		t.Fatalf("testkit: load scenario %q: %v", scenarioPath, err)
	}

	t.Run(s.Name, func(t *testing.T) {
		runScenario(t, handler, s)
	})
}

// RunDir discovers every *.json file in dir and runs each as a t.Run subtest.
// Scenario files that fail to parse are reported as test failures (not fatal).
func RunDir(t *testing.T, handler http.Handler, dir string) {
	t.Helper()

	pattern := filepath.Join(dir, "*.json")
	entries, err := filepath.Glob(pattern)
	if err != nil || len(entries) == 0 {
		t.Fatalf("testkit: no scenario files found in %q", dir)
	}

	for _, path := range entries {
		path := path // loop variable capture
		s, err := LoadScenario(path)
		if err != nil {
			t.Errorf("testkit: load %q: %v", path, err)
			continue
		}

		t.Run(s.Name, func(t *testing.T) {
			runScenario(t, handler, s)
		})
	}
}

// ─── Internal execution ───────────────────────────────────────────────────────

func runScenario(t *testing.T, handler http.Handler, s *Scenario) {
	t.Helper()

	// ── 1. Build request body ─────────────────────────────────────────────

	var reqBody io.Reader
	if p := s.RequestBodyPath(); p != "" {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("[%s] read request file %q: %v", s.Name, p, err)
		}
		reqBody = bytes.NewReader(data)
	}

	// ── 2+3. Install HTTP mock transport ──────────────────────────────────

	mt := NewMockTransport(s)
	originalTransport := kashvihttp.DefaultClient.Transport
	kashvihttp.DefaultClient.Transport = mt
	defer func() {
		kashvihttp.DefaultClient.Transport = originalTransport
	}()

	// ── 4. Activate function mocks ────────────────────────────────────────

	resetAllMockers()
	if err := ActivateFuncMocks(s); err != nil {
		t.Fatalf("[%s] activate func mocks: %v", s.Name, err)
	}

	// ── 5. Fire the request ───────────────────────────────────────────────

	method := strings.ToUpper(s.RequestMethod)
	if method == "" {
		method = http.MethodGet
	}

	req := httptest.NewRequest(method, s.RequestURL, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range s.Headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// ── 6. Assert status code ─────────────────────────────────────────────

	AssertStatusCode(t, s, rec.Code)

	// ── 7. Assert response body ───────────────────────────────────────────

	if p := s.ResponseBodyPath(); p != "" {
		expected, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("[%s] read response file %q: %v", s.Name, p, err)
		} else {
			AssertJSONBody(t, s, expected, rec.Body.Bytes())
		}
	}

	// ── 8. Verify mocks were called ───────────────────────────────────────

	AssertMocksAllCalled(t, s, mt)

	// ── 9. Cleanup ────────────────────────────────────────────────────────

	resetAllMockers()
}

// ─── Debug helpers ────────────────────────────────────────────────────────────

// DumpScenario prints a human-readable summary of the scenario to stdout.
// Useful during test development to inspect what was loaded.
func DumpScenario(s *Scenario) {
	fmt.Printf("Scenario: %s\n", s.Name)
	fmt.Printf("  %s %s → %d\n", s.RequestMethod, s.RequestURL, s.ExpectedCode)
	fmt.Printf("  requestFile:  %s\n", s.RequestFileName)
	fmt.Printf("  responseFile: %s\n", s.ResponseFileName)
	fmt.Printf("  isMockRequired: %v  isDbMocked: %v\n", s.IsMockRequired, s.IsDbMocked)
	for i, step := range s.NetUtilMockStep {
		fmt.Printf("  mockStep[%d]: method=%s  isMock=%v  matchUrl=%q\n",
			i, step.Method, step.IsMock, step.MatchURL)
	}
}
