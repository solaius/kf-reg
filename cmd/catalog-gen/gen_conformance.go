package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateConformanceScaffold generates conformance test scaffold for a plugin.
// The generated test file imports pkg/catalog/conformance and configures the
// harness for this plugin. It is created once and can be edited.
func generateConformanceScaffold(config CatalogConfig) error {
	conformanceDir := filepath.Join("tests", "conformance")
	if err := os.MkdirAll(conformanceDir, 0755); err != nil {
		return fmt.Errorf("failed to create conformance directory: %w", err)
	}

	testFile := filepath.Join(conformanceDir, "conformance_test.go")

	// Only create if it doesn't exist (editable file)
	if _, err := os.Stat(testFile); err == nil {
		fmt.Printf("  Skipped: %s (already exists)\n", testFile)
		return nil
	}

	packageName := filepath.Base(config.Metadata.Name)

	data := map[string]any{
		"Name":        packageName,
		"PackageName": packageName,
		"EntityName":  config.Spec.Entity.Name,
		"Package":     config.Spec.Package,
		"BasePath":    config.Spec.API.BasePath,
	}

	if err := executeTemplate(TmplConformanceTest, testFile, data); err != nil {
		return fmt.Errorf("failed to generate conformance test: %w", err)
	}
	fmt.Printf("  Created: %s\n", testFile)

	return nil
}

// conformanceTestExists checks if a conformance test file exists in the given directory.
func conformanceTestExists(dir string) bool {
	conformanceDir := filepath.Join(dir, "tests", "conformance")
	entries, err := os.ReadDir(conformanceDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}
