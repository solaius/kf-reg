package plugin

import (
	"context"
	"fmt"
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

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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
	store, err := NewFileConfigStore(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.NoError(t, err)
	_, _, err = store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestFileConfigStore_Load_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, "not: [valid: yaml: {{")

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	_, _, err = store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestFileConfigStore_Load_StableVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)

	_, v1, err := store.Load(context.Background())
	require.NoError(t, err)

	_, v2, err := store.Load(context.Background())
	require.NoError(t, err)

	assert.Equal(t, v1, v2, "loading the same file twice should produce the same version hash")
}

func TestFileConfigStore_Save(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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
	store, err := NewFileConfigStore(filepath.Join(t.TempDir(), "dummy.yaml"))
	require.NoError(t, err)
	ch, err := store.Watch(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, ch, "Watch should return nil channel for FileConfigStore")
}

func TestFileConfigStore_Path(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sources.yaml")
	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	assert.Equal(t, path, store.Path())
}

func TestFileConfigStore_RoundTrip_PreservesStructure(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
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

func TestFileConfigStore_PathTraversal(t *testing.T) {
	_, err := NewFileConfigStore("../../etc/passwd")
	require.ErrorIs(t, err, ErrPathTraversal)
}

func TestFileConfigStore_OversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.yaml")

	// Create a file larger than 1 MiB.
	data := make([]byte, maxConfigFileSize+1)
	for i := range data {
		data[i] = 'a'
	}
	err := os.WriteFile(path, data, 0644)
	require.NoError(t, err)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	_, _, err = store.Load(context.Background())
	require.ErrorIs(t, err, ErrFileTooLarge)
}

func TestFileConfigStore_RevisionCreatedOnSave(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	ctx := context.Background()

	// Initially no revisions.
	revs, err := store.ListRevisions(ctx)
	require.NoError(t, err)
	assert.Empty(t, revs)

	// Load and save to create a revision.
	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	_, err = store.Save(ctx, cfg, version)
	require.NoError(t, err)

	// Now there should be one revision.
	revs, err = store.ListRevisions(ctx)
	require.NoError(t, err)
	assert.Len(t, revs, 1)
	assert.NotEmpty(t, revs[0].Version)
	assert.True(t, revs[0].Size > 0)
}

func TestFileConfigStore_ListRevisions(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	ctx := context.Background()

	// Do 3 saves, each producing different content to get distinct versions.
	for i := 0; i < 3; i++ {
		cfg, version, err := store.Load(ctx)
		require.NoError(t, err)

		// Mutate to produce a different hash each time.
		models := cfg.Catalogs["models"]
		models.Sources = append(models.Sources, SourceConfig{
			ID:   fmt.Sprintf("rev-src-%d", i),
			Name: fmt.Sprintf("Rev Source %d", i),
			Type: "yaml",
		})
		cfg.Catalogs["models"] = models

		_, err = store.Save(ctx, cfg, version)
		require.NoError(t, err)
	}

	revs, err := store.ListRevisions(ctx)
	require.NoError(t, err)
	assert.Len(t, revs, 3)

	// Sorted newest first.
	for i := 0; i < len(revs)-1; i++ {
		assert.True(t, !revs[i].Timestamp.Before(revs[i+1].Timestamp),
			"revisions should be sorted newest first")
	}
}

func TestFileConfigStore_RollbackRestoresPrevious(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	ctx := context.Background()

	// Load original.
	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, cfg.Catalogs["models"].Sources, 2)

	// Mutate and save.
	models := cfg.Catalogs["models"]
	models.Sources = append(models.Sources, SourceConfig{
		ID:   "src-3",
		Name: "Source Three",
		Type: "yaml",
	})
	cfg.Catalogs["models"] = models

	_, err = store.Save(ctx, cfg, version)
	require.NoError(t, err)

	// Verify current state has 3 sources.
	cfg2, _, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, cfg2.Catalogs["models"].Sources, 3)

	// Get revisions and rollback to the first one.
	revs, err := store.ListRevisions(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, revs)

	// Rollback to the original (first revision saved = the state before mutation).
	restoredCfg, _, err := store.Rollback(ctx, revs[0].Version)
	require.NoError(t, err)

	// The restored config should have the original 2 sources.
	assert.Len(t, restoredCfg.Catalogs["models"].Sources, 2)
}

func TestFileConfigStore_HistoryPruning(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir, testConfigYAML)

	store, err := NewFileConfigStore(path)
	require.NoError(t, err)
	ctx := context.Background()

	// Do 25 saves (should prune to 20).
	for i := 0; i < 25; i++ {
		cfg, version, err := store.Load(ctx)
		require.NoError(t, err)
		_, err = store.Save(ctx, cfg, version)
		require.NoError(t, err)
	}

	revs, err := store.ListRevisions(ctx)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(revs), maxRevisionHistory)
}
