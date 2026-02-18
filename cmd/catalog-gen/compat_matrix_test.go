package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPluginCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		serverVersion string
		entry         CompatMatrixEntry
		want          string
	}{
		{
			name:          "within range",
			serverVersion: "0.9.5",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.0.0"},
			want:          "Yes",
		},
		{
			name:          "at min boundary",
			serverVersion: "0.9.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.0.0"},
			want:          "Yes",
		},
		{
			name:          "at max boundary",
			serverVersion: "1.0.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.0.0"},
			want:          "Yes",
		},
		{
			name:          "below min",
			serverVersion: "0.8.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.0.0"},
			want:          "No",
		},
		{
			name:          "above max",
			serverVersion: "2.0.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.0.0"},
			want:          "No",
		},
		{
			name:          "wildcard max within major",
			serverVersion: "1.5.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.x"},
			want:          "Yes",
		},
		{
			name:          "wildcard max exceeds major",
			serverVersion: "2.0.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.x"},
			want:          "No",
		},
		{
			name:          "no constraints",
			serverVersion: "1.0.0",
			entry:         CompatMatrixEntry{MinServer: "", MaxServer: ""},
			want:          "Unknown",
		},
		{
			name:          "invalid server version",
			serverVersion: "invalid",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: "1.0.0"},
			want:          "Unknown",
		},
		{
			name:          "min only",
			serverVersion: "1.0.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: ""},
			want:          "Yes",
		},
		{
			name:          "min only below",
			serverVersion: "0.8.0",
			entry:         CompatMatrixEntry{MinServer: "0.9.0", MaxServer: ""},
			want:          "No",
		},
		{
			name:          "max only",
			serverVersion: "0.5.0",
			entry:         CompatMatrixEntry{MinServer: "", MaxServer: "1.0.0"},
			want:          "Yes",
		},
		{
			name:          "max only above",
			serverVersion: "2.0.0",
			entry:         CompatMatrixEntry{MinServer: "", MaxServer: "1.0.0"},
			want:          "No",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPluginCompatibility(tt.serverVersion, tt.entry)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestBuildCompatMatrix_WithoutPluginYAML(t *testing.T) {
	manifest := &ServerManifest{
		APIVersion: "v1",
		Kind:       "CatalogServerBuild",
		Spec: ServerManifestSpec{
			Base: ServerBase{Module: "github.com/test", Version: "v1.0.0"},
			Plugins: []ServerPluginRef{
				{Name: "nonexistent-plugin", Module: "github.com/test/plugin", Version: "v1.0.0"},
			},
		},
	}

	entries := buildCompatMatrix(manifest)
	require.Len(t, entries, 1)
	assert.Equal(t, "nonexistent-plugin", entries[0].Name)
	assert.Equal(t, "Unknown", entries[0].Compatible)
	assert.Equal(t, "-", entries[0].MinServer)
	assert.Equal(t, "-", entries[0].MaxServer)
	assert.Equal(t, "-", entries[0].FrameworkAPI)
}

func TestBuildCompatMatrix_WithPluginYAML(t *testing.T) {
	// Create a temporary directory structure with plugin.yaml
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "catalog", "plugins", "test-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: test-plugin
spec:
  displayName: Test Plugin
  description: "test"
  version: "1.0.0"
  owners:
    - team: test
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.x"
    frameworkApi: v1alpha1
  providers:
    - yaml
`), 0644))

	// Save and restore working directory since buildCompatMatrix uses relative paths
	originalDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(originalDir) }()

	manifest := &ServerManifest{
		APIVersion: "v1",
		Kind:       "CatalogServerBuild",
		Spec: ServerManifestSpec{
			Base: ServerBase{Module: "github.com/test", Version: "v1.0.0"},
			Plugins: []ServerPluginRef{
				{Name: "test-plugin", Module: "github.com/test/plugin", Version: "v1.0.0"},
			},
		},
	}

	entries := buildCompatMatrix(manifest)
	require.Len(t, entries, 1)
	assert.Equal(t, "test-plugin", entries[0].Name)
	assert.Equal(t, "0.9.0", entries[0].MinServer)
	assert.Equal(t, "1.x", entries[0].MaxServer)
	assert.Equal(t, "v1alpha1", entries[0].FrameworkAPI)
	assert.Equal(t, "Yes", entries[0].Compatible)
}

func TestGenerateCompatMatrix_OutputFile(t *testing.T) {
	dir := t.TempDir()

	manifest := &ServerManifest{
		APIVersion: "v1",
		Kind:       "CatalogServerBuild",
		Spec: ServerManifestSpec{
			Base: ServerBase{Module: "github.com/test", Version: "v1.0.0"},
			Plugins: []ServerPluginRef{
				{Name: "plugin-a", Module: "github.com/test/a", Version: "v1.0.0"},
			},
		},
	}

	err := generateCompatMatrix(manifest, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "COMPATIBILITY.md"))
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "Compatibility Matrix")
	assert.Contains(t, s, "github.com/test")
	assert.Contains(t, s, "plugin-a")
	assert.Contains(t, s, "v1.0.0")
}

func TestBuildCompatMatrix_MultiplePlugins(t *testing.T) {
	manifest := &ServerManifest{
		Spec: ServerManifestSpec{
			Base: ServerBase{Module: "m", Version: "v1.0.0"},
			Plugins: []ServerPluginRef{
				{Name: "a", Module: "m/a", Version: "v1.0.0"},
				{Name: "b", Module: "m/b", Version: "v2.0.0"},
				{Name: "c", Module: "m/c", Version: "v0.1.0"},
			},
		},
	}

	entries := buildCompatMatrix(manifest)
	assert.Len(t, entries, 3)
	assert.Equal(t, "a", entries[0].Name)
	assert.Equal(t, "b", entries[1].Name)
	assert.Equal(t, "c", entries[2].Name)
}

func TestRunCompatMatrixCmd_ToFile(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(validManifestYAML), 0644))

	outputPath := filepath.Join(dir, "matrix.md")
	err := runCompatMatrixCmd(manifestPath, outputPath)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "Compatibility Matrix")
	assert.Contains(t, s, "mcp")
	assert.Contains(t, s, "models")
}

func TestRunCompatMatrixCmd_InvalidManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`apiVersion: v1
kind: WrongKind
spec:
  base:
    module: ""
`), 0644))

	err := runCompatMatrixCmd(manifestPath, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest validation failed")
}

func TestRunCompatMatrixCmd_MissingManifest(t *testing.T) {
	err := runCompatMatrixCmd("/nonexistent.yaml", "")
	require.Error(t, err)
}
