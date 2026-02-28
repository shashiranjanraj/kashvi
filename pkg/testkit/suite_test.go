package testkit

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestSuiteRunner(t *testing.T) {
	// 1. Setup sample master config
	masterConfig := []ConfigEntry{
		{
			ServiceName:       "TestEchoEndpoint",
			FilePath:          "sample_api",
			ScenariosFileName: "echo_scenario.json",
			ServiceURL:        "/api/echo",
			HTTPMethodType:    "POST",
			WorkflowService:   "HandleEcho",
		},
	}

	scenarios := []Scenario{
		{
			Name:             "EchoSuccess",
			Description:      "Echoes the request body",
			RequestMethod:    "POST",
			RequestURL:       "/api/echo",
			ExpectedCode:     200,
			RequestFileName:  "req.json",
			ResponseFileName: "res.json",
		},
	}

	// 2. Write temp files
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "test_scenarios.json")

	masterData, _ := json.Marshal(masterConfig)
	_ = os.WriteFile(masterPath, masterData, 0644)

	apiDir := filepath.Join(dir, "sample_api")
	_ = os.MkdirAll(apiDir, 0755)

	scenarioData, _ := json.Marshal(scenarios)
	_ = os.WriteFile(filepath.Join(apiDir, "echo_scenario.json"), scenarioData, 0644)

	reqData := []byte(`{"message": "hello"}`)
	resData := []byte(`{"message": "hello"}`)
	_ = os.WriteFile(filepath.Join(apiDir, "req.json"), reqData, 0644)
	_ = os.WriteFile(filepath.Join(apiDir, "res.json"), resData, 0644)

	// 3. Create mock handler
	handlers := map[string]http.HandlerFunc{
		"HandleEcho": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message": "hello"}`))
		},
	}

	// 4. Run suite
	// Note: We'd normally use t directly, but since we are testing the testkit itself,
	// errors inside RunSuite trigger t.Fatal. A clean run without panics/fatals is a success.
	RunSuite(t, masterPath, handlers)
}
