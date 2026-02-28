// Package testkit provides a JSON-scenario-driven REST API testing framework.
//
// Each scenario is a JSON file that describes:
//   - The HTTP request to fire (method, URL, body file, headers)
//   - Expected HTTP status code
//   - Expected response body file (optional, for JSON diff assertion)
//   - Mock steps for outgoing HTTP calls and other side-effects (mail, SMS…)
//
// Scenario files live next to your *_test.go files:
//
//	testdata/
//	  create_user.json           ← scenario
//	  create_user_req.json       ← request body
//	  create_user_res.json       ← expected response body
//
// Example _test.go:
//
//	func TestAPI(t *testing.T) {
//	    handler := kernel.NewHTTPKernel().Handler()
//	    testkit.RunDir(t, handler, "testdata")
//	}
package testkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ─── Schema ───────────────────────────────────────────────────────────────────

// Scenario describes a single REST API test case loaded from a JSON file.
type Scenario struct {
	// Meta
	Name        string `json:"name"`
	Description string `json:"description"`

	// Request
	RequestMethod   string            `json:"requestMethod"`   // GET, POST, PUT, PATCH, DELETE
	RequestURL      string            `json:"requestUrl"`      // e.g. /api/v1/users
	RequestFileName string            `json:"requestFileName"` // path to JSON request body file (relative to scenario dir)
	Headers         map[string]string `json:"headers"`         // extra request headers

	// Response assertions
	ResponseFileName   string `json:"responseFileName"`   // path to expected response JSON file
	ExpectedCode       int    `json:"expectedCode"`       // expected HTTP status code
	ExpectedStatusCode int    `json:"expectedStatusCode"` // alias for expected HTTP status code

	// Behaviour flags
	IsDbMocked             bool `json:"isDbMocked"`
	IsMockRequired         bool `json:"isMockRequired"`         // fail if an outgoing call has no matching mock
	IsConfigChangeRequired bool `json:"isConfigChangeRequired"` // reserved for future env overrides

	// Mock steps — executed/intercepted in definition order.
	NetUtilMockStep []MockStep `json:"netUtilMockStep"`

	// resolved at load time — not in JSON
	dir string // directory of the scenario file
}

// MockStep describes one intercepted outgoing call.
//
// Built-in methods:
//
//	"httprequest" — intercepts pkg/http outgoing HTTP calls
//	"sendmail"    — intercepts pkg/mail sends
//	"sms"         — intercepts SMS/notification sends
//	Any other string is dispatched to a registered FuncMocker.
type MockStep struct {
	// Method identifies what is being mocked.
	// "httprequest" | "sendmail" | "sms" | <custom>
	Method string `json:"method"`

	// IsMock — when true the step is intercepted and returnData is returned.
	// When false the real implementation is called (useful to document real deps).
	IsMock bool `json:"isMock"`

	// MatchURL is used by "httprequest" to match the outgoing request URL.
	// Supports prefix matching (e.g. "https://api.example.com/").
	// Leave empty to match ANY outgoing HTTP request.
	MatchURL string `json:"matchUrl"`

	// ReturnData is the synthetic response returned by the mock.
	ReturnData MockReturnData `json:"returnData"`
}

// MockReturnData is the synthetic response for a mock step.
type MockReturnData struct {
	// StatusCode is used by "httprequest" mocks. Defaults to 200.
	StatusCode int `json:"statusCode"`

	// Body is the response/return value.
	// For "httprequest": the HTTP response body.
	// For function mocks: passed as raw bytes to the mocker.
	//
	// Value must be base64-encoded. The runner decodes it before use.
	// Use "" for empty responses.
	Body string `json:"body"` // base64-encoded
}

// ─── Loading ──────────────────────────────────────────────────────────────────

// LoadScenario reads and validates a scenario from a JSON file.
func LoadScenario(path string) (*Scenario, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("testkit: resolve path %q: %w", path, err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("testkit: read %q: %w", abs, err)
	}

	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("testkit: parse %q: %w", abs, err)
	}

	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("testkit: invalid scenario %q: %w", abs, err)
	}

	s.dir = filepath.Dir(abs)
	return &s, nil
}

// validate performs basic sanity checks on the loaded scenario.
func (s *Scenario) validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if s.RequestURL == "" {
		return fmt.Errorf("requestUrl is required")
	}
	if s.ExpectedCode == 0 {
		return fmt.Errorf("expectedCode is required")
	}
	if s.RequestMethod == "" {
		s.RequestMethod = "GET" // sensible default
	}
	for i, step := range s.NetUtilMockStep {
		if step.Method == "" {
			return fmt.Errorf("netUtilMockStep[%d].method is required", i)
		}
	}
	return nil
}

// RequestBodyPath returns the absolute path to the request body file,
// resolved relative to the scenario file's directory.
// Returns "" when RequestFileName is not set.
func (s *Scenario) RequestBodyPath() string {
	if s.RequestFileName == "" {
		return ""
	}
	if filepath.IsAbs(s.RequestFileName) {
		return s.RequestFileName
	}
	return filepath.Join(s.dir, s.RequestFileName)
}

// ResponseBodyPath returns the absolute path to the expected response file.
// Returns "" when ResponseFileName is not set.
func (s *Scenario) ResponseBodyPath() string {
	if s.ResponseFileName == "" {
		return ""
	}
	if filepath.IsAbs(s.ResponseFileName) {
		return s.ResponseFileName
	}
	return filepath.Join(s.dir, s.ResponseFileName)
}

// LoadAllFromDir loads every *.json file in dir as a Scenario.
// Files that fail to parse are collected as errors, not panicked.
func LoadAllFromDir(dir string) ([]*Scenario, []error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(entries) == 0 {
		return nil, []error{fmt.Errorf("testkit: no scenario files found in %q", dir)}
	}

	var (
		scenarios []*Scenario
		errs      []error
	)
	for _, path := range entries {
		s, err := LoadScenario(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		scenarios = append(scenarios, s)
	}
	return scenarios, errs
}

// LoadScenarioArray reads and validates an array of scenarios from a JSON file.
// This is used by the suite runner which expects multiple scenarios per file.
func LoadScenarioArray(path string) ([]*Scenario, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("testkit: resolve scenario array path %q: %w", path, err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("testkit: read scenario array %q: %w", abs, err)
	}

	var scenarios []*Scenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		return nil, fmt.Errorf("testkit: parse scenario array %q: %w", abs, err)
	}

	dir := filepath.Dir(abs)
	for _, s := range scenarios {
		// Set directory so RequestBodyPath and ResponseBodyPath can resolve correctly
		s.dir = dir

		// ExpectedCode might be set to 0 by default, which is caught by validate()
		// If ExpectedCode is missing or 0, we can safely default it to 200 to help users
		// But in this JSON it's unmarshaled directly to ExpectedCode by standard Go json.Unmarshal.
		// NOTE: In the user's schema, they used expectedStatusCode. Since our struct uses expectedCode,
		// we must support both. We'll add a helper unmarshal step or just let the caller ensure expectedCode
		// is populated in the JSON so no override happens.
		if s.ExpectedCode == 0 {
			if s.ExpectedStatusCode != 0 {
				s.ExpectedCode = s.ExpectedStatusCode
			} else {
				// fallback check if they didn't provide expectedCode, default to 200
				s.ExpectedCode = 200
			}
		}

		// Note: We bypass s.validate() here because URL and Method are often injected by the Suite
		// Runner loop instead of being rigidly defined in every JSON array element.
		// If they remain completely empty during RunSuite, the HTTP request will fail naturally.
		// However, we still validate names and steps.
		if s.Name == "" {
			return nil, fmt.Errorf("testkit: invalid scenario array item: name is required")
		}
		for i, step := range s.NetUtilMockStep {
			if step.Method == "" {
				return nil, fmt.Errorf("testkit: invalid scenario array item %q: netUtilMockStep[%d].method is required", s.Name, i)
			}
		}
	}

	return scenarios, nil
}
