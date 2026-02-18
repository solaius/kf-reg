package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
}

const validPluginYAML = `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test-plugin
spec:
  displayName: Test Plugin
  description: "A test plugin"
  version: "0.1.0"
  owners:
    - team: ai-platform
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.x"
    frameworkApi: v1alpha1
  providers:
    - yaml
`

const validCatalogYAML = `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogConfig
metadata:
  name: test-plugin
spec:
  package: github.com/test/test
  entity:
    name: TestEntity
  api:
    basePath: /api/test_catalog/v1alpha1
    port: 8081
`

// setupFullValidDir creates a directory with plugin.yaml, catalog.yaml,
// conformance tests, and all docs files.
func setupFullValidDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "plugin.yaml", validPluginYAML)
	writeFile(t, dir, "catalog.yaml", validCatalogYAML)

	// Add conformance test
	conformanceDir := filepath.Join(dir, "tests", "conformance")
	require.NoError(t, os.MkdirAll(conformanceDir, 0755))
	writeFile(t, dir, filepath.Join("tests", "conformance", "conformance_test.go"), "package test")

	// Add docs
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0755))
	for _, doc := range []string{"README.md", "provider-guide.md", "schema-guide.md", "testing.md", "publishing.md"} {
		writeFile(t, dir, filepath.Join("docs", doc), "# "+doc)
	}

	return dir
}

func TestRunValidate_AllValid(t *testing.T) {
	dir := setupFullValidDir(t)

	err := runValidate(dir)
	assert.NoError(t, err)
}

func TestRunValidate_MissingPluginYAML(t *testing.T) {
	dir := setupFullValidDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "plugin.yaml")))

	err := runValidate(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestRunValidate_MissingCatalogYAML(t *testing.T) {
	dir := setupFullValidDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "catalog.yaml")))

	err := runValidate(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestRunValidate_InvalidPluginYAML(t *testing.T) {
	dir := setupFullValidDir(t)
	writeFile(t, dir, "plugin.yaml", "{{not valid yaml")

	err := runValidate(dir)
	require.Error(t, err)
}

func TestRunValidate_PluginMissingFields(t *testing.T) {
	dir := setupFullValidDir(t)
	writeFile(t, dir, "plugin.yaml", `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test
spec:
  version: "not-semver"
  providers:
    - yaml
`)

	err := runValidate(dir)
	require.Error(t, err)
}

func TestRunValidate_CatalogMissingFields(t *testing.T) {
	dir := setupFullValidDir(t)
	writeFile(t, dir, "catalog.yaml", `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogConfig
metadata:
  name: test
spec:
  package: ""
  entity:
    name: ""
`)

	err := runValidate(dir)
	require.Error(t, err)
}

func TestRunValidate_BothMissing(t *testing.T) {
	dir := t.TempDir()

	err := runValidate(dir)
	require.Error(t, err)
}

func TestRunValidate_MissingConformance(t *testing.T) {
	dir := setupFullValidDir(t)
	// Remove conformance tests
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "tests")))

	err := runValidate(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestRunValidate_MissingDocs(t *testing.T) {
	dir := setupFullValidDir(t)
	// Remove docs
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "docs")))

	err := runValidate(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestRunValidate_PartialDocs(t *testing.T) {
	dir := setupFullValidDir(t)
	// Remove just one doc
	require.NoError(t, os.Remove(filepath.Join(dir, "docs", "publishing.md")))

	err := runValidate(dir)
	require.Error(t, err)
}

func TestValidateCatalogYAML_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, os.WriteFile(path, []byte(validCatalogYAML), 0644))

	errs := validateCatalogYAML(path)
	assert.Empty(t, errs)
}

func TestValidateCatalogYAML_NotFound(t *testing.T) {
	errs := validateCatalogYAML("/nonexistent/catalog.yaml")
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "catalog.yaml not found")
}

func TestValidateCatalogYAML_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid"), 0644))

	errs := validateCatalogYAML(path)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "failed to parse catalog.yaml")
}

func TestConformanceTestExists(t *testing.T) {
	// No directory
	assert.False(t, conformanceTestExists(t.TempDir()))

	// Empty directory
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "tests", "conformance"), 0755))
	assert.False(t, conformanceTestExists(dir))

	// With test file
	writeFile(t, dir, filepath.Join("tests", "conformance", "conformance_test.go"), "package test")
	assert.True(t, conformanceTestExists(dir))
}

func TestDocsKitComplete(t *testing.T) {
	// No docs
	dir := t.TempDir()
	errs := docsKitComplete(dir)
	assert.Len(t, errs, 5)

	// Partial docs
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0755))
	writeFile(t, dir, filepath.Join("docs", "README.md"), "# readme")
	errs = docsKitComplete(dir)
	assert.Len(t, errs, 4)

	// All docs
	for _, doc := range []string{"provider-guide.md", "schema-guide.md", "testing.md", "publishing.md"} {
		writeFile(t, dir, filepath.Join("docs", doc), "# "+doc)
	}
	errs = docsKitComplete(dir)
	assert.Empty(t, errs)
}
