package testkit

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/stretchr/testify/mock"
)

// ─── FuncMocker interface ─────────────────────────────────────────────────────

// FuncMocker wraps a testify/mock.Mock so the runner can activate and verify
// non-HTTP side-effects (email, SMS, push notification, etc.) from scenario files.
//
// Register your own mockers:
//
//	func init() {
//	    testkit.RegisterMocker("myservice", testkit.NewFuncMocker("myservice"))
//	}
type FuncMocker interface {
	// Intercept is called by the runner when a mock step is active.
	// rawBody is the base64-decoded ReturnData.Body from the scenario.
	Intercept(rawBody []byte) error

	// Reset clears call history between test scenarios.
	Reset()

	// WasCalled returns how many times Intercept was called since the last Reset.
	WasCalled() int

	// Mock exposes the embedded testify mock for advanced call expectations.
	// Callers can use this to add custom On/Return chains in their tests.
	Mock() *mock.Mock
}

// ─── GenericFuncMocker — testify-backed implementation ───────────────────────

// GenericFuncMocker is a testify/mock-backed FuncMocker.
// It records every call to Intercept so testify assertions work naturally.
type GenericFuncMocker struct {
	m      mock.Mock
	method string
	mu     sync.Mutex
	calls  int
}

// NewFuncMocker creates a GenericFuncMocker for the named method.
// The mocker is pre-configured to return nil on any call to Intercept.
func NewFuncMocker(method string) *GenericFuncMocker {
	gm := &GenericFuncMocker{method: method}
	// Default: accept any call and return nil error.
	gm.m.On("Intercept", mock.AnythingOfType("[]uint8")).Return(nil)
	return gm
}

// Intercept records the call via testify and returns the configured return value.
func (gm *GenericFuncMocker) Intercept(rawBody []byte) error {
	gm.mu.Lock()
	gm.calls++
	gm.mu.Unlock()

	args := gm.m.Called(rawBody)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

// Reset clears testify call records and resets the call counter.
func (gm *GenericFuncMocker) Reset() {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.calls = 0
	gm.m.Calls = nil // clear testify history
	// Re-register the default expectation after reset.
	gm.m.On("Intercept", mock.AnythingOfType("[]uint8")).Return(nil)
}

// WasCalled returns how many times Intercept was called since the last Reset.
func (gm *GenericFuncMocker) WasCalled() int {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.calls
}

// Mock exposes the underlying testify mock for advanced expectations.
func (gm *GenericFuncMocker) Mock() *mock.Mock { return &gm.m }

// ─── Registry ─────────────────────────────────────────────────────────────────

var (
	mockerMu       sync.RWMutex
	mockerRegistry = map[string]FuncMocker{
		"sendmail":     NewFuncMocker("sendmail"),
		"sms":          NewFuncMocker("sms"),
		"notification": NewFuncMocker("notification"),
	}
)

// RegisterMocker registers a FuncMocker for the given method name.
// Call from your test package's init() to add custom mockers.
func RegisterMocker(method string, m FuncMocker) {
	mockerMu.Lock()
	defer mockerMu.Unlock()
	mockerRegistry[method] = m
}

// GetMocker retrieves a registered FuncMocker by method name (nil if not found).
// Use this in your tests to set custom expectations or inspect calls:
//
//	m := testkit.GetMocker("sendmail")
//	m.Mock().On("Intercept", mock.Anything).Return(nil)
func GetMocker(method string) FuncMocker {
	mockerMu.RLock()
	defer mockerMu.RUnlock()
	return mockerRegistry[method]
}

func getMocker(method string) FuncMocker { return GetMocker(method) }

// resetAllMockers resets every registered mocker between scenarios.
func resetAllMockers() {
	mockerMu.RLock()
	defer mockerMu.RUnlock()
	for _, m := range mockerRegistry {
		m.Reset()
	}
}

// ─── Scenario activation ──────────────────────────────────────────────────────

// ActivateFuncMocks activates all non-HTTP mock steps from the scenario.
func ActivateFuncMocks(s *Scenario) error {
	for i, step := range s.NetUtilMockStep {
		if step.Method == "httprequest" || !step.IsMock {
			continue
		}
		m := getMocker(step.Method)
		if m == nil {
			if s.IsMockRequired {
				return fmt.Errorf("testkit: no mocker registered for %q (step %d)", step.Method, i)
			}
			continue
		}

		// Decode base64 body before calling Intercept.
		var raw []byte
		if step.ReturnData.Body != "" {
			decoded, err := base64.StdEncoding.DecodeString(step.ReturnData.Body)
			if err != nil {
				decoded, err = base64.RawStdEncoding.DecodeString(step.ReturnData.Body)
				if err != nil {
					return fmt.Errorf("testkit: step %d base64 decode: %w", i, err)
				}
			}
			raw = decoded
		}

		if err := m.Intercept(raw); err != nil {
			return fmt.Errorf("testkit: step %d mock intercept failed: %w", i, err)
		}
	}
	return nil
}

// AssertFuncMocksCalled verifies that every isMock=true non-HTTP step was called.
func AssertFuncMocksCalled(s *Scenario) []error {
	var errs []error
	seen := map[string]bool{}
	for _, step := range s.NetUtilMockStep {
		if step.Method == "httprequest" || !step.IsMock || seen[step.Method] {
			continue
		}
		seen[step.Method] = true
		m := getMocker(step.Method)
		if m == nil {
			continue
		}
		if m.WasCalled() == 0 {
			errs = append(errs, fmt.Errorf(
				"mock %q registered but never called during scenario %q",
				step.Method, s.Name,
			))
		}
	}
	return errs
}
