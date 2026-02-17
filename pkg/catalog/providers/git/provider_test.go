package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kubeflow/model-registry/pkg/catalog"
)

// testEntity is a simple entity for testing.
type testEntity struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

type testCatalog struct {
	Items []testEntity `yaml:"items"`
}

func parseTestEntities(data []byte) ([]catalog.Record[testEntity, any], error) {
	var cat testCatalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, err
	}
	records := make([]catalog.Record[testEntity, any], 0, len(cat.Items))
	for _, item := range cat.Items {
		records = append(records, catalog.Record[testEntity, any]{
			Entity: item,
		})
	}
	return records, nil
}

// createBareRepo creates a bare Git repo with a YAML file for testing.
func createBareRepo(t *testing.T, items []testEntity) string {
	t.Helper()

	// Create a regular repo first, add content, then use it as the "remote".
	dir := t.TempDir()
	repo, err := gogit.PlainInitWithOptions(dir, &gogit.PlainInitOptions{
		InitOptions: gogit.InitOptions{
			DefaultBranch: "refs/heads/main",
		},
	})
	require.NoError(t, err)

	// Create a YAML file in the repo.
	dataDir := filepath.Join(dir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))

	cat := testCatalog{Items: items}
	data, err := yaml.Marshal(&cat)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "test.yaml"), data, 0o644))

	// Stage and commit.
	w, err := repo.Worktree()
	require.NoError(t, err)

	_, err = w.Add("data/test.yaml")
	require.NoError(t, err)

	_, err = w.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	return dir
}

func TestNewProvider(t *testing.T) {
	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl": "https://github.com/example/repo.git",
			"branch":  "develop",
			"path":    "catalogs/**/*.yaml",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/example/repo.git", provider.repoURL)
	assert.Equal(t, "develop", provider.branch)
	assert.Equal(t, "catalogs/**/*.yaml", provider.pathPattern)
	assert.True(t, provider.shallowClone)
}

func TestNewProvider_MissingURL(t *testing.T) {
	source := &catalog.Source{
		ID:         "test-source",
		Properties: map[string]any{},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	_, err := NewProvider(config, source, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing repoUrl")
}

func TestNewProvider_Defaults(t *testing.T) {
	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl": "https://github.com/example/repo.git",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	assert.Equal(t, "main", provider.branch)
	assert.Equal(t, "**/*.yaml", provider.pathPattern)
	assert.Equal(t, 1*time.Hour, provider.syncInterval)
	assert.True(t, provider.shallowClone)
}

func TestCloneAndRead(t *testing.T) {
	items := []testEntity{
		{Name: "item1", Value: "value1"},
		{Name: "item2", Value: "value2"},
	}
	repoDir := createBareRepo(t, items)

	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl": repoDir,
			"path":    "data/*.yaml",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	// Disable shallow clone for local repo test.
	provider.shallowClone = false

	records, err := provider.cloneAndRead()
	require.NoError(t, err)
	defer provider.cleanup()

	assert.Len(t, records, 2)
	assert.Equal(t, "item1", records[0].Entity.Name)
	assert.Equal(t, "item2", records[1].Entity.Name)
	assert.NotEmpty(t, provider.LastCommit())
}

func TestRecords(t *testing.T) {
	items := []testEntity{
		{Name: "alpha", Value: "a"},
		{Name: "beta", Value: "b"},
		{Name: "gamma", Value: "c"},
	}
	repoDir := createBareRepo(t, items)

	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl":      repoDir,
			"path":         "data/*.yaml",
			"syncInterval": "10s",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	provider.shallowClone = false

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := provider.Records(ctx)
	require.NoError(t, err)

	// Collect initial records (3 data records + 1 zero sentinel).
	var received []catalog.Record[testEntity, any]
	for i := 0; i < 4; i++ {
		select {
		case r := <-ch:
			received = append(received, r)
		case <-ctx.Done():
			t.Fatal("timed out waiting for records")
		}
	}

	// 3 real records + 1 zero sentinel.
	assert.Len(t, received, 4)
	assert.Equal(t, "alpha", received[0].Entity.Name)
	assert.Equal(t, "beta", received[1].Entity.Name)
	assert.Equal(t, "gamma", received[2].Entity.Name)
	assert.Empty(t, received[3].Entity.Name) // Zero sentinel.
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.yaml", "test.yaml", true},
		{"*.yaml", "test.json", false},
		{"data/*.yaml", "data/test.yaml", true},
		{"data/*.yaml", "other/test.yaml", false},
		{"**/*.yaml", "test.yaml", true},
		{"**/*.yaml", "data/test.yaml", true},
		{"**/*.yaml", "a/b/c/test.yaml", true},
		{"**/*.yaml", "test.json", false},
		{"data/**/*.yaml", "data/sub/test.yaml", true},
		{"data/**/*.yaml", "data/a/b/test.yaml", true},
		{"data/**/*.yaml", "other/test.yaml", false},
		{"**/*", "anything/at/all", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.path)
			assert.Equal(t, tt.want, got, "matchGlob(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestFilter(t *testing.T) {
	items := []testEntity{
		{Name: "keep", Value: "yes"},
		{Name: "skip", Value: "no"},
	}
	repoDir := createBareRepo(t, items)

	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl": repoDir,
			"path":    "data/*.yaml",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
		Filter: func(r catalog.Record[testEntity, any]) bool {
			return r.Entity.Name == "keep"
		},
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	provider.shallowClone = false

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := provider.Records(ctx)
	require.NoError(t, err)

	// Should get 1 real record + 1 zero sentinel.
	var received []catalog.Record[testEntity, any]
	for i := 0; i < 2; i++ {
		select {
		case r := <-ch:
			received = append(received, r)
		case <-ctx.Done():
			t.Fatal("timed out waiting for records")
		}
	}

	assert.Len(t, received, 2)
	assert.Equal(t, "keep", received[0].Entity.Name)
	assert.Empty(t, received[1].Entity.Name) // Zero sentinel.
}

func TestShallowCloneDefault(t *testing.T) {
	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl": "https://example.com/repo.git",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	assert.True(t, provider.shallowClone)
}

func TestShallowCloneOverride(t *testing.T) {
	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl":      "https://example.com/repo.git",
			"shallowClone": false,
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	assert.False(t, provider.shallowClone)
}

func TestCustomSyncInterval(t *testing.T) {
	source := &catalog.Source{
		ID: "test-source",
		Properties: map[string]any{
			"repoUrl":      "https://example.com/repo.git",
			"syncInterval": "30m",
		},
	}

	config := Config[testEntity, any]{
		Parse: parseTestEntities,
	}

	provider, err := NewProvider(config, source, "")
	require.NoError(t, err)
	assert.Equal(t, 30*time.Minute, provider.syncInterval)
}
