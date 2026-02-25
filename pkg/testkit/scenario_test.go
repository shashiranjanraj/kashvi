// Package testkit_test demonstrates how to use testkit.RunDir() to drive
// REST API tests entirely from JSON scenario files.
//
// Usage in YOUR project:
//
//  1. Copy your scenario JSON files into a testdata/ (or fixtures/) directory.
//  2. Call testkit.RunDir(t, yourHandler, "testdata")
//  3. go test ./... — each scenario becomes a named subtest.
//
// Advanced: custom per-test mock expectations
//
//	func TestCreateUserCustomMock(t *testing.T) {
//	    // Override the sendmail mocker with a custom return.
//	    mailer := testkit.NewFuncMocker("sendmail")
//	    mailer.Mock().On("Intercept", mock.Anything).Return(nil)
//	    testkit.RegisterMocker("sendmail", mailer)
//
//	    testkit.Run(t, handler, "fixtures/create_user.json")
//
//	    // Assert the mailer was called exactly once.
//	    mailer.Mock().AssertNumberOfCalls(t, "Intercept", 1)
//	}
package testkit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/shashiranjanraj/kashvi/pkg/testkit"
)

// ─── Minimal test handler ─────────────────────────────────────────────────────

// testHandler is a tiny http.Handler that powers the testkit self-tests.
// In real projects, replace with:   kernel.NewHTTPKernel().Handler()
var testHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`)) //nolint:errcheck
	}
})

// ─── RunDir: run ALL fixtures/*.json as subtests ──────────────────────────────

// TestRunDir_HealthCheck demonstrates RunDir — one test function drives all
// JSON scenarios found in the fixtures directory.
// Only the health_check.json scenario passes with this minimal handler.
func TestRunDir_HealthCheck(t *testing.T) {
	// RunDir discovers every *.json in fixtures/ and runs each as a subtest.
	// In your project, use your full kernel handler instead of testHandler.
	testkit.Run(t, testHandler, "fixtures/health_check.json")
}

// ─── Single scenario with custom mock expectations ────────────────────────────

// TestScenario_CustomMockExpectation shows how to use testify/mock expectations
// alongside JSON scenarios.
func TestScenario_CustomMockExpectation(t *testing.T) {
	// 1. Register a custom sendmail mocker with a specific expectation.
	mailer := testkit.NewFuncMocker("sendmail")
	// Expect Intercept to be called exactly once with any bytes.
	mailer.Mock().On("Intercept", mock.AnythingOfType("[]uint8")).Return(nil)
	testkit.RegisterMocker("sendmail", mailer)

	// 2. Load and inspect the scenario.
	s, err := testkit.LoadScenario("fixtures/create_user.json")
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	testkit.DumpScenario(s)

	// 3. Assert scenario metadata loaded correctly.
	assert.Equal(t, "Create User - External Verification + Email", s.Name)
	assert.Equal(t, "POST", s.RequestMethod)
	assert.Equal(t, 201, s.ExpectedCode)
	assert.True(t, s.IsMockRequired)
	assert.Len(t, s.NetUtilMockStep, 2)

	// 4. Assert mock step fields.
	httpStep := s.NetUtilMockStep[0]
	assert.Equal(t, "httprequest", httpStep.Method)
	assert.True(t, httpStep.IsMock)
	assert.Equal(t, "https://verify.external.com/v1/check", httpStep.MatchURL)
	assert.NotEmpty(t, httpStep.ReturnData.Body) // base64

	mailStep := s.NetUtilMockStep[1]
	assert.Equal(t, "sendmail", mailStep.Method)
	assert.True(t, mailStep.IsMock)
}

// ─── MockTransport unit test ──────────────────────────────────────────────────

// TestMockTransport_URLMatching verifies the MockTransport matches and decodes
// the base64 response body correctly.
func TestMockTransport_URLMatching(t *testing.T) {
	s := &testkit.Scenario{
		Name:           "mock transport test",
		IsMockRequired: true,
		ExpectedCode:   200,
		RequestURL:     "/anything",
		RequestMethod:  "GET",
		NetUtilMockStep: []testkit.MockStep{
			{
				Method:   "httprequest",
				IsMock:   true,
				MatchURL: "https://api.example.com/",
				ReturnData: testkit.MockReturnData{
					StatusCode: 200,
					// base64("{"ok":true}")
					Body: "eyJvayI6dHJ1ZX0=",
				},
			},
		},
	}

	mt := testkit.NewMockTransport(s)

	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/users", nil)
	resp, err := mt.RoundTrip(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	errs := mt.AssertAllCalled()
	assert.Empty(t, errs, "all HTTP mock steps should have been called")
}

// TestMockTransport_UnmatchedCallFails verifies that an unmatched outgoing
// call returns an error when isMockRequired is true.
func TestMockTransport_UnmatchedCallFails(t *testing.T) {
	s := &testkit.Scenario{
		Name:           "unmatched mock",
		IsMockRequired: true,
		ExpectedCode:   200,
		RequestURL:     "/anything",
		RequestMethod:  "GET",
		NetUtilMockStep: []testkit.MockStep{
			{
				Method:     "httprequest",
				IsMock:     true,
				MatchURL:   "https://expected.com/",
				ReturnData: testkit.MockReturnData{StatusCode: 200},
			},
		},
	}

	mt := testkit.NewMockTransport(s)

	// Call a URL that doesn't match the registered mock.
	req := httptest.NewRequest(http.MethodGet, "https://unexpected.com/api", nil)
	_, err := mt.RoundTrip(req)

	assert.Error(t, err, "should fail on unmatched URL when isMockRequired=true")
}

// ─── JSON assertion unit test ─────────────────────────────────────────────────

// TestAssertJSONBody verifies the JSON deep-diff assertion.
func TestAssertJSONBody(t *testing.T) {
	s := &testkit.Scenario{Name: "json assert test", ExpectedCode: 200}

	// Matching JSON (different whitespace / key order) — should pass.
	expected := []byte(`{"name":"Shashi","age":30}`)
	actual := []byte(`{"age":  30, "name": "Shashi"}`)
	testkit.AssertJSONBody(t, s, expected, actual)
}
