package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestGolden_DeterministicGeneration verifies that running generate twice
// on the same catalog.yaml produces identical non-editable output files.
func TestGolden_DeterministicGeneration(t *testing.T) {
	// Create two output directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Write identical catalog.yaml in both
	for _, dir := range []string{dir1, dir2} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte(validCatalogYAML), 0644))
	}

	// Run generate in dir1
	runGenerate(t, dir1)

	// Run generate in dir2
	runGenerate(t, dir2)

	// Compare non-editable (auto-regenerated) files
	// These are the files that should be deterministic
	nonEditableFiles := []string{
		filepath.Join("internal", "db", "models", "testentity.go"),
		filepath.Join("internal", "db", "service", "spec.go"),
		filepath.Join("internal", "db", "service", "filter_mappings.go"),
		filepath.Join("api", "openapi", "src", "generated", "components.yaml"),
		filepath.Join("internal", "catalog", "loader.go"),
		"plugin.go",
		"register.go",
	}

	for _, relPath := range nonEditableFiles {
		path1 := filepath.Join(dir1, relPath)
		path2 := filepath.Join(dir2, relPath)

		content1, err1 := os.ReadFile(path1)
		content2, err2 := os.ReadFile(path2)

		if err1 != nil || err2 != nil {
			// If a file doesn't exist in either, skip comparison
			// (some files depend on OpenAPI generator which we don't run here)
			continue
		}

		assert.Equal(t, string(content1), string(content2),
			"non-editable file %s should be identical across two generate runs", relPath)
	}
}

// TestGolden_ConformanceScaffold verifies the conformance scaffold is generated
// correctly and is not overwritten on subsequent runs.
func TestGolden_ConformanceScaffold(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte(validCatalogYAML), 0644))

	// Generate once
	runGenerate(t, dir)

	// Check conformance test exists
	conformanceFile := filepath.Join(dir, "tests", "conformance", "conformance_test.go")
	_, err := os.Stat(conformanceFile)
	require.NoError(t, err, "conformance test file should exist after generate")

	content1, err := os.ReadFile(conformanceFile)
	require.NoError(t, err)
	assert.Contains(t, string(content1), "conformance.RunConformance")
	assert.Contains(t, string(content1), "test-plugin")

	// Modify the file
	modified := "// modified by user\n" + string(content1)
	require.NoError(t, os.WriteFile(conformanceFile, []byte(modified), 0644))

	// Generate again -- should NOT overwrite
	runGenerate(t, dir)

	content2, err := os.ReadFile(conformanceFile)
	require.NoError(t, err)
	assert.Equal(t, modified, string(content2),
		"conformance test should not be overwritten on re-generate")
}

// TestGolden_DocsKit verifies the docs kit is generated correctly and
// existing docs are not overwritten.
func TestGolden_DocsKit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte(validCatalogYAML), 0644))

	// Generate once
	runGenerate(t, dir)

	// Check all docs exist
	expectedDocs := []string{
		"README.md",
		"provider-guide.md",
		"schema-guide.md",
		"testing.md",
		"publishing.md",
	}

	docsDir := filepath.Join(dir, "docs")
	for _, doc := range expectedDocs {
		path := filepath.Join(docsDir, doc)
		_, err := os.Stat(path)
		assert.NoError(t, err, "docs/%s should exist after generate", doc)
	}

	// Check content has required sections
	readme, err := os.ReadFile(filepath.Join(docsDir, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(readme), "Overview")
	assert.Contains(t, string(readme), "Quick Start")

	providerGuide, err := os.ReadFile(filepath.Join(docsDir, "provider-guide.md"))
	require.NoError(t, err)
	assert.Contains(t, string(providerGuide), "Supported Providers")

	testing_, err := os.ReadFile(filepath.Join(docsDir, "testing.md"))
	require.NoError(t, err)
	assert.Contains(t, string(testing_), "Conformance Suite")

	publishing, err := os.ReadFile(filepath.Join(docsDir, "publishing.md"))
	require.NoError(t, err)
	assert.Contains(t, string(publishing), "Versioning")
	assert.Contains(t, string(publishing), "Release Checklist")

	// Modify a doc
	modified := "# Custom README\nUser-modified content."
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "README.md"), []byte(modified), 0644))

	// Generate again -- should NOT overwrite
	runGenerate(t, dir)

	readmeAfter, err := os.ReadFile(filepath.Join(docsDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, modified, string(readmeAfter),
		"docs/README.md should not be overwritten on re-generate")
}

// TestGolden_TwoRunsProduceIdenticalNonEditable runs generate twice in the
// same directory and verifies non-editable files are byte-identical.
func TestGolden_TwoRunsProduceIdenticalNonEditable(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte(validCatalogYAML), 0644))

	// Run 1
	runGenerate(t, dir)

	// Capture non-editable files
	snap1 := snapshotNonEditableFiles(t, dir)
	require.NotEmpty(t, snap1, "should have captured some non-editable files")

	// Run 2
	runGenerate(t, dir)

	// Compare
	snap2 := snapshotNonEditableFiles(t, dir)
	for path, content1 := range snap1 {
		content2, ok := snap2[path]
		require.True(t, ok, "file %s should exist in second snapshot", path)
		assert.Equal(t, content1, content2,
			"non-editable file %s should be identical across two runs", path)
	}
}

// runGenerate runs the generate process in a directory using the same
// logic as initCatalogPlugin but calling generatePluginFiles, conformance,
// and docs directly.
func runGenerate(t *testing.T, dir string) {
	t.Helper()

	// Read catalog config from the directory
	configData, err := os.ReadFile(filepath.Join(dir, "catalog.yaml"))
	require.NoError(t, err)

	var config CatalogConfig
	err = parseYAML(configData, &config)
	require.NoError(t, err)

	// Save and restore working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(originalDir) }()

	// Create required directories
	dirs := []string{
		filepath.Join("internal", "db", "models"),
		filepath.Join("internal", "db", "service"),
		filepath.Join("internal", "server", "openapi"),
		filepath.Join("internal", "catalog", "providers"),
		filepath.Join("api", "openapi", "src", "generated"),
		filepath.Join("api", "generated"),
		filepath.Join("pkg", "openapi"),
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(d, 0755))
	}

	// Generate non-editable files
	require.NoError(t, generatePluginFiles(config))
	require.NoError(t, generateEntityModel(config))
	require.NoError(t, generateDatastoreSpec(config))
	require.NoError(t, generateFilterMappings(config))
	require.NoError(t, generateOpenAPIComponents(config))
	require.NoError(t, generateLoader(config))

	// Generate conformance scaffold (only if not exists)
	require.NoError(t, generateConformanceScaffold(config))

	// Generate docs kit (only if not exists)
	require.NoError(t, generateDocsKit(config))
}

// parseYAML is a test helper to parse YAML.
func parseYAML(data []byte, out any) error {
	return yaml.Unmarshal(data, out)
}

// snapshotNonEditableFiles reads non-editable files and returns a map of
// relative path -> content.
func snapshotNonEditableFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}

	nonEditable := []string{
		"plugin.go",
		"register.go",
		filepath.Join("internal", "db", "models", "testentity.go"),
		filepath.Join("internal", "db", "service", "spec.go"),
		filepath.Join("internal", "db", "service", "filter_mappings.go"),
		filepath.Join("api", "openapi", "src", "generated", "components.yaml"),
		filepath.Join("internal", "catalog", "loader.go"),
	}

	for _, relPath := range nonEditable {
		path := filepath.Join(dir, relPath)
		content, err := os.ReadFile(path)
		if err != nil {
			continue // skip files that don't exist
		}
		files[relPath] = string(content)
	}

	return files
}
