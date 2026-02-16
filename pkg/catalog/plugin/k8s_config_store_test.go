package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace     = "kubeflow"
	testConfigMapName = "catalog-sources"
	testDataKey       = "sources.yaml"
)

const testK8sConfigYAML = `apiVersion: catalog/v1alpha1
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

func createTestConfigMap(t *testing.T, data string) *fake.Clientset {
	t.Helper()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{
			testDataKey: data,
		},
	}
	return fake.NewSimpleClientset(cm)
}

func newTestK8sStore(client *fake.Clientset) *K8sSourceConfigStore {
	return NewK8sSourceConfigStore(client, testNamespace, testConfigMapName, testDataKey)
}

func TestK8sConfigStore_Load(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)

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

func TestK8sConfigStore_Load_StableVersion(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
	ctx := context.Background()

	_, v1, err := store.Load(ctx)
	require.NoError(t, err)

	_, v2, err := store.Load(ctx)
	require.NoError(t, err)

	assert.Equal(t, v1, v2, "loading the same ConfigMap twice should produce the same version hash")
}

func TestK8sConfigStore_Load_ConfigMapNotFound(t *testing.T) {
	client := fake.NewSimpleClientset() // no ConfigMap
	store := NewK8sSourceConfigStore(client, testNamespace, testConfigMapName, testDataKey)

	_, _, err := store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get ConfigMap")
}

func TestK8sConfigStore_Load_MissingDataKey(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{
			"other-key.yaml": "something",
		},
	}
	client := fake.NewSimpleClientset(cm)
	store := newTestK8sStore(client)

	_, _, err := store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key \"sources.yaml\" not found")
}

func TestK8sConfigStore_Load_InvalidYAML(t *testing.T) {
	client := createTestConfigMap(t, "not: [valid: yaml: {{")
	store := newTestK8sStore(client)

	_, _, err := store.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestK8sConfigStore_Save(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
	ctx := context.Background()

	// Load to get current version.
	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// Mutate: add a new source.
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

func TestK8sConfigStore_Save_VersionConflict(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
	ctx := context.Background()

	cfg, version, err := store.Load(ctx)
	require.NoError(t, err)

	// Simulate an external edit by directly modifying the ConfigMap.
	cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, testConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	cm.Data[testDataKey] = "apiVersion: catalog/v1alpha1\nkind: CatalogSources\ncatalogs: {}\n"
	_, err = client.CoreV1().ConfigMaps(testNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Save with the stale version should fail.
	_, err = store.Save(ctx, cfg, version)
	require.ErrorIs(t, err, ErrVersionConflict)
}

func TestK8sConfigStore_Save_StaleVersionString(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
	ctx := context.Background()

	cfg, _, err := store.Load(ctx)
	require.NoError(t, err)

	// Try to save with a fabricated version.
	_, err = store.Save(ctx, cfg, "bogus-version-hash")
	require.ErrorIs(t, err, ErrVersionConflict)
}

func TestK8sConfigStore_Save_ConcurrentLoadSave(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
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

func TestK8sConfigStore_Watch_ReturnsNil(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)

	ch, err := store.Watch(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, ch, "Watch should return nil channel for K8sSourceConfigStore")
}

func TestK8sConfigStore_ListRevisions_Empty(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)

	revs, err := store.ListRevisions(context.Background())
	require.NoError(t, err)
	assert.Empty(t, revs)
}

func TestK8sConfigStore_ListRevisions(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
	ctx := context.Background()

	// Do 3 saves, each producing different content to get distinct revisions.
	for i := 0; i < 3; i++ {
		cfg, version, err := store.Load(ctx)
		require.NoError(t, err)

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

func TestK8sConfigStore_RevisionCreatedOnSave(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
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

func TestK8sConfigStore_Rollback(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
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

	// Rollback to the original (first revision = the state before mutation).
	restoredCfg, _, err := store.Rollback(ctx, revs[0].Version)
	require.NoError(t, err)

	// The restored config should have the original 2 sources.
	assert.Len(t, restoredCfg.Catalogs["models"].Sources, 2)
}

func TestK8sConfigStore_Rollback_NotFound(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)

	_, _, err := store.Rollback(context.Background(), "nonexistent-version")
	require.ErrorIs(t, err, ErrRevisionNotFound)
}

func TestK8sConfigStore_RevisionPruning(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
	ctx := context.Background()

	// Do more saves than maxK8sRevisionHistory.
	for i := 0; i < maxK8sRevisionHistory+5; i++ {
		cfg, version, err := store.Load(ctx)
		require.NoError(t, err)

		models := cfg.Catalogs["models"]
		models.Sources = append(models.Sources, SourceConfig{
			ID:   fmt.Sprintf("prune-src-%d", i),
			Name: fmt.Sprintf("Prune Source %d", i),
			Type: "yaml",
		})
		cfg.Catalogs["models"] = models

		_, err = store.Save(ctx, cfg, version)
		require.NoError(t, err)
	}

	revs, err := store.ListRevisions(ctx)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(revs), maxK8sRevisionHistory)
}

func TestK8sConfigStore_RoundTrip_PreservesStructure(t *testing.T) {
	client := createTestConfigMap(t, testK8sConfigYAML)
	store := newTestK8sStore(client)
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
