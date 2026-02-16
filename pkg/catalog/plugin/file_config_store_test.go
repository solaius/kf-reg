package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfigYAML = `apiVersion: catalog/v1alpha1
kind: CatalogSources
catalogs:
  models:
    sources:
      - id: src-1
        name: Source One
        type: yaml
        enabled: true
        properties:
          yamlCatalogPath: ./data/models.yaml
      - id: src-2
        name: Source Two
        type: hf
        properties:
          allowedOrganization: redhat
`

func writeTestConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "sources.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestFileConfigStore_Load(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	cfg, version, err := store.Load(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, version)

	assert.Equal(t, "catalog/v1alpha1", cfg.APIVersion)
	assert.Equal(t, "CatalogSources", cfg.Kind)
	assert.Contains(t, cfg.Catalogs, "models")

	models := cfg.Catalogs["models"]
	assert.Len(t, models.Sources, 2)
	assert.Equal(t, "src-1", models.Sources[0].ID)
	assert.Equal(t, "Source One", models.Sources[0].Name)
}

func TestFileConfigStore_Load_FileNotFound(t *testing.T) {
	store := NewFileConfigStore(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	_, _, err := store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestFileConfigStore_Load_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, "not: [valid: yaml: {{")

	store := NewFileConfigStore(path)
	_, _, err := store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestFileConfigStore_Load_StableVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)

	_, v1, err := store.Load(context.Background())
	require.NoError(t, err)

	_, v2, err := store.Load(context.Background())
	require.NoError(t, err)

	assert.Equal(t, v1, v2, "loading the same file twice should produce the same version hash")
}

func TestFileConfigStore_Save(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	ctx := context.Background()

	// Load to get current version.
	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// Mutate the config: add a new source.
	models := cfg.Catalogs["models"]
	models.Sources = append(models.Sources, SourceConfig{
		ID:   "src-3",
		Name: "Source Three",
		Type: "yaml",
	})
	cfg.Catalogs["models"] = models

	// Save with correct version.
	newVersion, err := store.Save(ctx, cfg, version)
	require.NoError(t, err)
	assert.NotEqual(t, version, newVersion, "version should change after save")

	// Reload and verify the change persisted.
	cfg2, v2, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, newVersion, v2)
	assert.Len(t, cfg2.Catalogs["models"].Sources, 3)

	// Find the new source.
	var found bool
	for _, s := range cfg2.Catalogs["models"].Sources {
		if s.ID == "src-3" {
			found = true
			assert.Equal(t, "Source Three", s.Name)
		}
	}
	assert.True(t, found, "newly added source should be in the reloaded config")
}

func TestFileConfigStore_Save_VersionConflict(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	ctx := context.Background()

	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// Simulate an external edit by writing different content directly.
	err = os.WriteFile(path, []byte("apiVersion: catalog/v1alpha1\nkind: CatalogSources\ncatalogs: {}\n"), 0644)
	require.NoError(t, err)

	// Save with the stale version should fail.
	_, err = store.Save(ctx, cfg, version)
	require.ErrorIs(t, err, ErrVersionConflict)
}

func TestFileConfigStore_Save_StaleVersionString(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	ctx := context.Background()

	cfg, _, err := store.Load(ctx)
	require.NoError(t, err)

	// Try to save with a fabricated version.
	_, err = store.Save(ctx, cfg, "bogus-version-hash")
	require.ErrorIs(t, err, ErrVersionConflict)
}

func TestFileConfigStore_Save_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	ctx := context.Background()

	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// Save and verify no temp files are left behind.
	_, err = store.Save(ctx, cfg, version)
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp", "temp file should be cleaned up after save")
	}
}

func TestFileConfigStore_Watch_ReturnsNil(t *testing.T) {
	store := NewFileConfigStore("/dummy/path")
	ch, err := store.Watch(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, ch, "Watch should return nil channel for FileConfigStore")
}

func TestFileConfigStore_Path(t *testing.T) {
	store := NewFileConfigStore("/some/path/sources.yaml")
	assert.Equal(t, "/some/path/sources.yaml", store.Path())
}

func TestFileConfigStore_RoundTrip_PreservesStructure(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	ctx := context.Background()

	// Load.
	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// Save unchanged.
	newVersion, err := store.Save(ctx, cfg, version)
	require.NoError(t, err)

	// Reload.
	cfg2, v2, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, newVersion, v2)

	// Structural equivalence.
	assert.Equal(t, cfg.APIVersion, cfg2.APIVersion)
	assert.Equal(t, cfg.Kind, cfg2.Kind)
	assert.Equal(t, len(cfg.Catalogs), len(cfg2.Catalogs))

	for key, section := range cfg.Catalogs {
		section2, ok := cfg2.Catalogs[key]
		require.True(t, ok, "catalog %q should exist after round-trip", key)
		assert.Equal(t, len(section.Sources), len(section2.Sources))
	}
}

func TestFileConfigStore_ConcurrentLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store := NewFileConfigStore(path)
	ctx := context.Background()

	// Load initial version.
	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// First save succeeds.
	v2, err := store.Save(ctx, cfg, version)
	require.NoError(t, err)

	// Second save with the old version fails.
	_, err = store.Save(ctx, cfg, version)
	require.ErrorIs(t, err, ErrVersionConflict)

	// Second save with the new version succeeds.
	_, err = store.Save(ctx, cfg, v2)
	require.NoError(t, err)
}
