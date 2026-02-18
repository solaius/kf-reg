package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validManifestYAML = `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogServerBuild
spec:
  base:
    module: github.com/kubeflow/model-registry
    version: v0.9.0
  plugins:
    - name: mcp
      module: github.com/kubeflow/model-registry/catalog/plugins/mcp
      version: v0.9.0
    - name: models
      module: github.com/kubeflow/model-registry/catalog/plugins/models
      version: v0.9.0
`

func TestLoadServerManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(path, []byte(validManifestYAML), 0644))

	m, err := loadServerManifest(path)
	require.NoError(t, err)

	assert.Equal(t, "catalog.kubeflow.org/v1alpha1", m.APIVersion)
	assert.Equal(t, "CatalogServerBuild", m.Kind)
	assert.Equal(t, "github.com/kubeflow/model-registry", m.Spec.Base.Module)
	assert.Equal(t, "v0.9.0", m.Spec.Base.Version)
	require.Len(t, m.Spec.Plugins, 2)
	assert.Equal(t, "mcp", m.Spec.Plugins[0].Name)
	assert.Equal(t, "github.com/kubeflow/model-registry/catalog/plugins/mcp", m.Spec.Plugins[0].Module)
	assert.Equal(t, "v0.9.0", m.Spec.Plugins[0].Version)
}

func TestLoadServerManifest_FileNotFound(t *testing.T) {
	_, err := loadServerManifest("/nonexistent/manifest.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read manifest")
}

func TestLoadServerManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid yaml"), 0644))

	_, err := loadServerManifest(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestValidateServerManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest ServerManifest
		wantErrs int
	}{
		{
			name: "valid manifest",
			manifest: ServerManifest{
				APIVersion: "v1",
				Kind:       "CatalogServerBuild",
				Spec: ServerManifestSpec{
					Base:    ServerBase{Module: "github.com/test", Version: "v1.0.0"},
					Plugins: []ServerPluginRef{{Name: "test", Module: "github.com/test/plugin", Version: "v1.0.0"}},
				},
			},
			wantErrs: 0,
		},
		{
			name:     "empty manifest",
			manifest: ServerManifest{},
			wantErrs: 5, // apiVersion, kind, base.module, base.version, plugins
		},
		{
			name: "wrong kind",
			manifest: ServerManifest{
				APIVersion: "v1",
				Kind:       "WrongKind",
				Spec: ServerManifestSpec{
					Base:    ServerBase{Module: "m", Version: "v1"},
					Plugins: []ServerPluginRef{{Name: "n", Module: "m", Version: "v1"}},
				},
			},
			wantErrs: 1,
		},
		{
			name: "plugin missing fields",
			manifest: ServerManifest{
				APIVersion: "v1",
				Kind:       "CatalogServerBuild",
				Spec: ServerManifestSpec{
					Base:    ServerBase{Module: "m", Version: "v1"},
					Plugins: []ServerPluginRef{{Name: "", Module: "", Version: ""}},
				},
			},
			wantErrs: 3, // name, module, version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateServerManifest(&tt.manifest)
			assert.Len(t, errs, tt.wantErrs, "errors: %v", errs)
		})
	}
}

func TestRunBuildServer_GeneratesFiles(t *testing.T) {
	// Create a manifest file
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(validManifestYAML), 0644))

	outputDir := filepath.Join(dir, "output")

	err := runBuildServer(manifestPath, outputDir, "1.22", false)
	require.NoError(t, err)

	// Verify all files were generated
	for _, file := range []string{"main.go", "go.mod", "Dockerfile", "COMPATIBILITY.md"} {
		path := filepath.Join(outputDir, file)
		_, err := os.Stat(path)
		assert.NoError(t, err, "expected %s to exist", file)
	}
}

func TestRunBuildServer_MainGoContent(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(validManifestYAML), 0644))

	outputDir := filepath.Join(dir, "output")
	require.NoError(t, runBuildServer(manifestPath, outputDir, "1.22", false))

	content, err := os.ReadFile(filepath.Join(outputDir, "main.go"))
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "package main")
	assert.Contains(t, s, "github.com/kubeflow/model-registry/catalog/plugins/mcp")
	assert.Contains(t, s, "github.com/kubeflow/model-registry/catalog/plugins/models")
	assert.Contains(t, s, "DO NOT EDIT")
}

func TestRunBuildServer_GoModContent(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(validManifestYAML), 0644))

	outputDir := filepath.Join(dir, "output")
	require.NoError(t, runBuildServer(manifestPath, outputDir, "1.22", false))

	content, err := os.ReadFile(filepath.Join(outputDir, "go.mod"))
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "go 1.22")
	assert.Contains(t, s, "github.com/kubeflow/model-registry v0.9.0")
	assert.Contains(t, s, "github.com/kubeflow/model-registry/catalog/plugins/mcp v0.9.0")
}

func TestRunBuildServer_DockerfileContent(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(validManifestYAML), 0644))

	outputDir := filepath.Join(dir, "output")
	require.NoError(t, runBuildServer(manifestPath, outputDir, "1.22", false))

	content, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "golang:1.22-alpine")
	assert.Contains(t, s, "catalog-server")
	assert.Contains(t, s, "distroless")
}

func TestRunBuildServer_InvalidManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`apiVersion: v1
kind: WrongKind
spec:
  base:
    module: ""
`), 0644))

	outputDir := filepath.Join(dir, "output")
	err := runBuildServer(manifestPath, outputDir, "1.22", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest validation failed")
}

func TestRunBuildServer_MissingManifest(t *testing.T) {
	err := runBuildServer("/nonexistent.yaml", "/tmp/out", "1.22", false)
	require.Error(t, err)
}

func TestRunBuildServer_SinglePlugin(t *testing.T) {
	manifest := `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogServerBuild
spec:
  base:
    module: github.com/kubeflow/model-registry
    version: v1.0.0
  plugins:
    - name: agents
      module: github.com/kubeflow/model-registry/catalog/plugins/agents
      version: v1.0.0
`
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifest), 0644))

	outputDir := filepath.Join(dir, "output")
	err := runBuildServer(manifestPath, outputDir, "1.23", false)
	require.NoError(t, err)

	// Verify main.go references only agents
	content, err := os.ReadFile(filepath.Join(outputDir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "plugins/agents")

	// Verify Dockerfile uses correct Go version
	content, err = os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "golang:1.23-alpine")
}
