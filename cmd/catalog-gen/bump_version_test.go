package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBumpVersion_Patch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plugin.yaml", validPluginYAML)

	err := runBumpVersion(dir, "patch")
	require.NoError(t, err)

	spec, err := plugin.LoadPluginMetadata(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "0.1.1", spec.Spec.Version)
}

func TestRunBumpVersion_Minor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plugin.yaml", validPluginYAML)

	err := runBumpVersion(dir, "minor")
	require.NoError(t, err)

	spec, err := plugin.LoadPluginMetadata(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "0.2.0", spec.Spec.Version)
}

func TestRunBumpVersion_Major(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plugin.yaml", validPluginYAML)

	err := runBumpVersion(dir, "major")
	require.NoError(t, err)

	spec, err := plugin.LoadPluginMetadata(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", spec.Spec.Version)
}

func TestRunBumpVersion_MissingPluginYAML(t *testing.T) {
	dir := t.TempDir()

	err := runBumpVersion(dir, "patch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load plugin.yaml")
}

func TestRunBumpVersion_InvalidPart(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plugin.yaml", validPluginYAML)

	err := runBumpVersion(dir, "invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bump part")
}

func TestRunBumpVersion_PreservesOtherFields(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plugin.yaml", validPluginYAML)

	err := runBumpVersion(dir, "patch")
	require.NoError(t, err)

	spec, err := plugin.LoadPluginMetadata(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)

	// Version should be bumped
	assert.Equal(t, "0.1.1", spec.Spec.Version)
	// Other fields should be preserved
	assert.Equal(t, "catalog.kubeflow.org/v1alpha1", spec.APIVersion)
	assert.Equal(t, "CatalogPlugin", spec.Kind)
	assert.Equal(t, "test-plugin", spec.Metadata.Name)
	assert.Equal(t, "Test Plugin", spec.Spec.DisplayName)
	assert.Equal(t, "A test plugin", spec.Spec.Description)
	require.Len(t, spec.Spec.Owners, 1)
	assert.Equal(t, "ai-platform", spec.Spec.Owners[0].Team)
}

func TestRunBumpVersion_SuccessiveBumps(t *testing.T) {
	dir := t.TempDir()

	// Start with explicit version
	content := `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test
spec:
  displayName: Test
  description: "desc"
  version: "1.0.0"
  owners:
    - team: t
  compatibility:
    frameworkApi: v1alpha1
  providers:
    - yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(content), 0644))

	// Bump patch twice
	require.NoError(t, runBumpVersion(dir, "patch"))
	require.NoError(t, runBumpVersion(dir, "patch"))

	spec, err := plugin.LoadPluginMetadata(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "1.0.2", spec.Spec.Version)
}
