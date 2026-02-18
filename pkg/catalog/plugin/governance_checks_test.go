package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const governanceTestPluginYAML = `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test-plugin
spec:
  displayName: Test Plugin
  description: "A test plugin for governance checks"
  version: "0.1.0"
  license: Apache-2.0
  owners:
    - team: ai-platform
      contact: ai-platform@example.com
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.x"
    frameworkApi: v1alpha1
  providers:
    - yaml
`

// setupGovernanceDir creates a temporary directory with all files needed to
// pass governance checks.
func setupGovernanceDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// plugin.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(governanceTestPluginYAML), 0644))

	// catalog.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte("kind: CatalogConfig\n"), 0644))

	// conformance tests
	confDir := filepath.Join(dir, "tests", "conformance")
	require.NoError(t, os.MkdirAll(confDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(confDir, "conformance_test.go"), []byte("package test"), 0644))

	// README.md
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Plugin\n"), 0644))

	return dir
}

func TestRunGovernanceChecks_AllPass(t *testing.T) {
	dir := setupGovernanceDir(t)

	report := RunGovernanceChecks(dir)

	assert.True(t, report.Passed, "expected all checks to pass")
	assert.Equal(t, dir, report.PluginDir)
	for _, c := range report.Checks {
		assert.True(t, c.Passed, "check %q should pass, message: %s", c.Name, c.Message)
	}
}

func TestRunGovernanceChecks_MissingPluginYAML(t *testing.T) {
	dir := setupGovernanceDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "plugin.yaml")))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)

	// plugin.yaml valid, compatibility, ownership, and license checks should fail
	failedNames := failedCheckNames(report)
	assert.Contains(t, failedNames, "plugin.yaml valid")
}

func TestRunGovernanceChecks_MissingCatalogYAML(t *testing.T) {
	dir := setupGovernanceDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "catalog.yaml")))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	failedNames := failedCheckNames(report)
	assert.Contains(t, failedNames, "catalog.yaml exists")
}

func TestRunGovernanceChecks_MissingOwners(t *testing.T) {
	dir := setupGovernanceDir(t)
	noOwners := `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test-plugin
spec:
  displayName: Test Plugin
  description: "A test plugin"
  version: "0.1.0"
  license: Apache-2.0
  owners: []
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
    frameworkApi: v1alpha1
  providers:
    - yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(noOwners), 0644))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	failedNames := failedCheckNames(report)
	// ValidatePluginMetadata catches missing owners, so "plugin.yaml valid" fails
	assert.Contains(t, failedNames, "plugin.yaml valid")
	assert.Contains(t, failedNames, "ownership declared")
}

func TestRunGovernanceChecks_MissingLicense(t *testing.T) {
	dir := setupGovernanceDir(t)
	noLicense := `apiVersion: catalog.kubeflow.org/v1alpha1
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
    frameworkApi: v1alpha1
  providers:
    - yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(noLicense), 0644))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	failedNames := failedCheckNames(report)
	assert.Contains(t, failedNames, "license present")
}

func TestRunGovernanceChecks_LicenseFile(t *testing.T) {
	dir := setupGovernanceDir(t)

	// Remove license from plugin.yaml but add a LICENSE file
	noLicense := `apiVersion: catalog.kubeflow.org/v1alpha1
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
    frameworkApi: v1alpha1
  providers:
    - yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(noLicense), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("Apache License 2.0"), 0644))

	report := RunGovernanceChecks(dir)

	// License check should pass via the LICENSE file fallback
	for _, c := range report.Checks {
		if c.Name == "license present" {
			assert.True(t, c.Passed, "license check should pass with LICENSE file")
		}
	}
}

func TestRunGovernanceChecks_MissingConformance(t *testing.T) {
	dir := setupGovernanceDir(t)
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "tests")))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	failedNames := failedCheckNames(report)
	assert.Contains(t, failedNames, "conformance tests exist")
}

func TestRunGovernanceChecks_MissingDocs(t *testing.T) {
	dir := setupGovernanceDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "README.md")))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	failedNames := failedCheckNames(report)
	assert.Contains(t, failedNames, "documentation present")
}

func TestRunGovernanceChecks_MissingCompatibility(t *testing.T) {
	dir := setupGovernanceDir(t)
	noCompat := `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test-plugin
spec:
  displayName: Test Plugin
  description: "A test plugin"
  version: "0.1.0"
  license: Apache-2.0
  owners:
    - team: ai-platform
  compatibility:
    catalogServer: {}
    frameworkApi: ""
  providers:
    - yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(noCompat), 0644))

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	failedNames := failedCheckNames(report)
	assert.Contains(t, failedNames, "compatibility fields present")
}

func TestRunGovernanceChecks_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	report := RunGovernanceChecks(dir)

	assert.False(t, report.Passed)
	// Multiple checks should fail
	failCount := 0
	for _, c := range report.Checks {
		if !c.Passed {
			failCount++
		}
	}
	assert.True(t, failCount >= 4, "expected at least 4 failures in empty dir, got %d", failCount)
}

// failedCheckNames returns the names of checks that did not pass.
func failedCheckNames(report GovernanceReport) []string {
	var names []string
	for _, c := range report.Checks {
		if !c.Passed {
			names = append(names, c.Name)
		}
	}
	return names
}
