package testkit

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertStatusCode checks the response code with testify.
func AssertStatusCode(t *testing.T, scenario *Scenario, got int) {
	t.Helper()
	assert.Equal(t, scenario.ExpectedCode, got,
		"[%s] HTTP status code mismatch", scenario.Name)
}

// AssertJSONBody deep-compares actual response bytes against the expected file
// contents using testify's assert.Equal after normalising both through JSON
// unmarshal (so key order and whitespace never matter).
// Reports field-level diffs on failure.
func AssertJSONBody(t *testing.T, scenario *Scenario, expected, actual []byte) {
	t.Helper()
	if len(expected) == 0 {
		return
	}

	var expVal, actVal interface{}

	require.NoError(t,
		json.Unmarshal(expected, &expVal),
		"[%s] expected response file is not valid JSON", scenario.Name,
	)

	if !assert.NoError(t,
		json.Unmarshal(actual, &actVal),
		"[%s] actual response is not valid JSON\nbody: %s", scenario.Name, string(actual),
	) {
		return
	}

	// Use testify's deep-equal diff — best-in-class output.
	assert.Equal(t, expVal, actVal,
		"[%s] response body mismatch", scenario.Name)
}

// AssertMocksAllCalled fails the test if any isMock=true step was never triggered.
func AssertMocksAllCalled(t *testing.T, scenario *Scenario, mt *MockTransport) {
	t.Helper()

	for _, err := range mt.AssertAllCalled() {
		assert.NoError(t, err, "[%s]", scenario.Name)
	}
	for _, err := range AssertFuncMocksCalled(scenario) {
		assert.NoError(t, err, "[%s]", scenario.Name)
	}
}

// ─── JSON diff helper (human-readable fallback) ───────────────────────────────

// DiffJSON returns a list of human-readable difference strings between two
// JSON-decoded values.  Used internally; testify's assert.Equal will already
// print a good diff, but this is kept for DumpScenario / manual use.
func DiffJSON(path string, expected, actual interface{}) []string {
	var diffs []string
	switch exp := expected.(type) {
	case map[string]interface{}:
		act, ok := actual.(map[string]interface{})
		if !ok {
			return append(diffs, fmt.Sprintf("  %s: expected object, got %T", keyPath(path), actual))
		}
		for k, ev := range exp {
			p := keyPath(path) + "." + k
			av, exists := act[k]
			if !exists {
				diffs = append(diffs, fmt.Sprintf("  %s: missing in actual", p))
				continue
			}
			diffs = append(diffs, DiffJSON(p, ev, av)...)
		}
	case []interface{}:
		act, ok := actual.([]interface{})
		if !ok {
			return append(diffs, fmt.Sprintf("  %s: expected array, got %T", keyPath(path), actual))
		}
		if len(exp) != len(act) {
			diffs = append(diffs, fmt.Sprintf("  %s: array length expected=%d actual=%d", keyPath(path), len(exp), len(act)))
		}
		for i := 0; i < len(exp) && i < len(act); i++ {
			diffs = append(diffs, DiffJSON(fmt.Sprintf("%s[%d]", keyPath(path), i), exp[i], act[i])...)
		}
	default:
		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			diffs = append(diffs, fmt.Sprintf("  %s:\n    - %v\n    + %v", keyPath(path), expected, actual))
		}
	}
	return diffs
}

func keyPath(path string) string {
	if path == "" {
		return "root"
	}
	return strings.TrimPrefix(path, ".")
}
