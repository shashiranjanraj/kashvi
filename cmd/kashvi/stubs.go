package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed stubs/*.stub
var defaultStubs embed.FS

// StubData holds variables passed to the .stub templates
type StubData struct {
	Name       string
	Lower      string
	StructName string // e.g. M_202301010000_create_users_table
	Authorize  bool   // Add Auth middleware/behavior
	Cache      bool   // Add Cache middleware/behavior
}

// renderStub locates the stub (user override first, embedded fallback)
// and returns the string output from text/template.
func renderStub(stubName string, data StubData) (string, error) {
	var stubContent []byte
	var err error

	// 1. Try to load user override from .kashvi/stubs/
	userPath := filepath.Join(".kashvi", "stubs", stubName+".stub")
	if _, errStat := os.Stat(userPath); errStat == nil {
		stubContent, err = os.ReadFile(userPath)
		if err != nil {
			return "", fmt.Errorf("failed to read user stub %s: %v", userPath, err)
		}
	} else {
		// 2. Fallback to embedded stub
		stubContent, err = defaultStubs.ReadFile("stubs/" + stubName + ".stub")
		if err != nil {
			return "", fmt.Errorf("embedded stub not found: %s", stubName)
		}
	}

	// 3. Compile as Go template
	t, err := template.New(stubName).Parse(string(stubContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %v", stubName, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %v", stubName, err)
	}

	return buf.String(), nil
}
