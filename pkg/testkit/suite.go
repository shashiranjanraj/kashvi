// Package testkit — suite.go
//
// Suite orchestration for data-driven REST API testing.

package testkit

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shashiranjanraj/kashvi/pkg/router"
)

// ConfigEntry represents a single REST API group in the master test_scenarios.json.
type ConfigEntry struct {
	ServiceName          string `json:"serviceName"`
	FilePath             string `json:"filePath"`
	ScenariosFileName    string `json:"scenariosFileName"`
	ServiceURL           string `json:"serviceUrl"`
	HTTPMethodType       string `json:"httpMethodType"`  // e.g. "GET", "POST"
	WorkflowService      string `json:"workflowService"` // The map key to look up the http.HandlerFunc
	IsGetService         bool   `json:"isGetService,omitempty"`
	IsNetUtilsUsed       bool   `json:"isNetUtilsUsed,omitempty"`
	IsFirestoreUtilsUsed bool   `json:"isFireStoreUtilsUsed,omitempty"`
}

// SuiteRun executes a suite of scenarios driven by a master JSON config file.
// masterConfigPath: Path to test_scenarios.json
// handlers: A map where keys correspond to ConfigEntry.WorkflowService and values are the handlers being tested.
func RunSuite(t *testing.T, masterConfigPath string, handlers map[string]http.HandlerFunc) {
	t.Helper()

	absMasterPath, err := filepath.Abs(masterConfigPath)
	if err != nil {
		t.Fatalf("testkit: resolve master config path %q: %v", masterConfigPath, err)
	}

	data, err := os.ReadFile(absMasterPath)
	if err != nil {
		t.Fatalf("testkit: read master config %q: %v", absMasterPath, err)
	}

	var entries []ConfigEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("testkit: parse master config %q: %v", absMasterPath, err)
	}

	baseDir := filepath.Dir(absMasterPath)

	for _, entry := range entries {
		t.Run(entry.ServiceName, func(t *testing.T) {
			handlerFunc, ok := handlers[entry.WorkflowService]
			if !ok {
				t.Fatalf("testkit: handler %q not found in provided map", entry.WorkflowService)
			}

			// Mount the handler onto a fake Kashvi router just for this entry.
			// This exercises right routing and middleware if desired, or just invokes the raw handler.
			// For suite isolation, we create a fresh router per service entry.
			r := router.New()
			// Ensure path starts with slash
			url := entry.ServiceURL
			if url != "" && url[0] != '/' {
				url = "/" + url
			}
			switch strings.ToUpper(entry.HTTPMethodType) {
			case "GET":
				r.Get(url, entry.WorkflowService, handlerFunc)
			case "POST":
				r.Post(url, entry.WorkflowService, handlerFunc)
			case "PUT":
				r.Put(url, entry.WorkflowService, handlerFunc)
			case "PATCH":
				r.Patch(url, entry.WorkflowService, handlerFunc)
			case "DELETE":
				r.Delete(url, entry.WorkflowService, handlerFunc)
			default:
				r.Get(url, entry.WorkflowService, handlerFunc) // fallback
			}

			// Resolve where the scenario array is located
			scenarioPath := filepath.Join(baseDir, entry.FilePath, entry.ScenariosFileName)

			// Some users might have made `FilePath` relative to the suite runner execution directory
			// rather than relative to the master config. Let's try baseDir + FilePath first.
			if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
				// Fallback: try raw filePath directly if it was meant to be purely relative to the test runner context (cwd)
				scenarioPath = filepath.Join(entry.FilePath, entry.ScenariosFileName)
			}

			// Load the Array of scenarios. Our existing `LoadScenario` expects one object per file.
			// To support the user's workflow, we need a bulk-load mechanism for array-based scenarios.
			scenarios, err := LoadScenarioArray(scenarioPath)
			if err != nil {
				t.Fatalf("testkit: load scenario array %q: %v", scenarioPath, err)
			}

			for _, s := range scenarios {
				// Inject the entry-level routing data into the scenario so `runScenario` fires the right request implicitly
				if s.RequestURL == "" {
					s.RequestURL = url
				}
				if s.RequestMethod == "" {
					s.RequestMethod = entry.HTTPMethodType
				}

				t.Run(s.Name, func(t *testing.T) {
					// Orchestrate
					runScenario(t, r.Handler(), s)
				})
			}
		})
	}
}
