package governance

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestVersionStore(t *testing.T) (*VersionStore, *BindingStore) {
	t.Helper()
	db := newTestDB(t)
	vs := NewVersionStore(db)
	require.NoError(t, vs.AutoMigrate())
	bs := NewBindingStore(db)
	require.NoError(t, bs.AutoMigrate())
	return vs, bs
}

func TestVersionStore_CreateAndGet(t *testing.T) {
	vs, _ := newTestVersionStore(t)

	record := &AssetVersionRecord{
		ID:                 uuid.New().String(),
		AssetUID:           "mcp:mcpserver:filesystem",
		VersionID:          "v1.0:abc12345",
		VersionLabel:       "v1.0",
		CreatedBy:          "alice",
		GovernanceSnapshot: JSONAny{"lifecycleState": "approved"},
		AssetSnapshot:      JSONAny{},
	}

	err := vs.CreateVersion(record)
	require.NoError(t, err)

	// Get the version back.
	got, err := vs.GetVersion("v1.0:abc12345")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v1.0:abc12345", got.VersionID)
	assert.Equal(t, "v1.0", got.VersionLabel)
	assert.Equal(t, "alice", got.CreatedBy)
	assert.Equal(t, "mcp:mcpserver:filesystem", got.AssetUID)

	// Not found.
	got, err = vs.GetVersion("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestVersionStore_ListVersions(t *testing.T) {
	vs, _ := newTestVersionStore(t)

	assetUID := "mcp:mcpserver:filesystem"
	baseTime := time.Now().Add(-time.Hour)

	// Create 5 versions with distinct timestamps.
	for i := 0; i < 5; i++ {
		record := &AssetVersionRecord{
			ID:                 uuid.New().String(),
			AssetUID:           assetUID,
			VersionID:          fmt.Sprintf("v%d:%s", i, uuid.New().String()[:8]),
			VersionLabel:       fmt.Sprintf("v%d", i),
			CreatedAt:          baseTime.Add(time.Duration(i) * time.Minute),
			CreatedBy:          "alice",
			GovernanceSnapshot: JSONAny{"lifecycleState": "draft"},
			AssetSnapshot:      JSONAny{},
		}
		require.NoError(t, vs.CreateVersion(record))
	}

	// Create a version for a different asset.
	require.NoError(t, vs.CreateVersion(&AssetVersionRecord{
		ID:                 uuid.New().String(),
		AssetUID:           "model:model:llama",
		VersionID:          "other-v1:" + uuid.New().String()[:8],
		VersionLabel:       "v1",
		CreatedBy:          "bob",
		GovernanceSnapshot: JSONAny{},
		AssetSnapshot:      JSONAny{},
	}))

	// List all for our asset.
	versions, nextToken, total, err := vs.ListVersions(assetUID, 10, "")
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, versions, 5)
	assert.Empty(t, nextToken)

	// Should be ordered by created_at DESC (newest first).
	for i := 1; i < len(versions); i++ {
		assert.True(t, versions[i-1].CreatedAt.After(versions[i].CreatedAt) || versions[i-1].CreatedAt.Equal(versions[i].CreatedAt),
			"versions should be ordered newest first")
	}

	// Paginate with pageSize 2.
	page1, token1, total1, err := vs.ListVersions(assetUID, 2, "")
	require.NoError(t, err)
	assert.Equal(t, 5, total1)
	assert.Len(t, page1, 2)
	assert.NotEmpty(t, token1)

	page2, token2, _, err := vs.ListVersions(assetUID, 2, token1)
	require.NoError(t, err)
	assert.Len(t, page2, 2)
	assert.NotEmpty(t, token2)

	page3, token3, _, err := vs.ListVersions(assetUID, 2, token2)
	require.NoError(t, err)
	assert.Len(t, page3, 1)
	assert.Empty(t, token3)

	// No versions for unknown asset.
	empty, _, total, err := vs.ListVersions("nonexistent", 10, "")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, empty)
}

func TestBindingStore_UpsertAndGet(t *testing.T) {
	_, bs := newTestVersionStore(t)

	// Create a new binding.
	binding := &EnvBindingRecord{
		ID:          uuid.New().String(),
		Plugin:      "mcp",
		AssetKind:   "mcpserver",
		AssetName:   "filesystem",
		Environment: "dev",
		AssetUID:    "mcp:mcpserver:filesystem",
		VersionID:   "v1.0:abc12345",
		BoundAt:     time.Now(),
		BoundBy:     "alice",
	}
	err := bs.SetBinding(binding)
	require.NoError(t, err)

	// Get the binding.
	got, err := bs.GetBinding("default", "mcp", "mcpserver", "filesystem", "dev")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dev", got.Environment)
	assert.Equal(t, "v1.0:abc12345", got.VersionID)
	assert.Equal(t, "alice", got.BoundBy)

	// Update same binding (upsert).
	updatedBinding := &EnvBindingRecord{
		ID:                uuid.New().String(),
		Plugin:            "mcp",
		AssetKind:         "mcpserver",
		AssetName:         "filesystem",
		Environment:       "dev",
		AssetUID:          "mcp:mcpserver:filesystem",
		VersionID:         "v2.0:def67890",
		BoundAt:           time.Now(),
		BoundBy:           "bob",
		PreviousVersionID: "v1.0:abc12345",
	}
	err = bs.SetBinding(updatedBinding)
	require.NoError(t, err)

	// Verify it was updated.
	got, err = bs.GetBinding("default", "mcp", "mcpserver", "filesystem", "dev")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v2.0:def67890", got.VersionID)
	assert.Equal(t, "bob", got.BoundBy)
	assert.Equal(t, "v1.0:abc12345", got.PreviousVersionID)

	// Not found for different environment.
	got, err = bs.GetBinding("default", "mcp", "mcpserver", "filesystem", "prod")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestBindingStore_ListBindings(t *testing.T) {
	_, bs := newTestVersionStore(t)

	now := time.Now()

	// Create bindings for multiple environments.
	for _, env := range []string{"dev", "prod", "stage"} {
		err := bs.SetBinding(&EnvBindingRecord{
			ID:          uuid.New().String(),
			Plugin:      "mcp",
			AssetKind:   "mcpserver",
			AssetName:   "filesystem",
			Environment: env,
			AssetUID:    "mcp:mcpserver:filesystem",
			VersionID:   "v1.0:abc12345",
			BoundAt:     now,
			BoundBy:     "alice",
		})
		require.NoError(t, err)
	}

	// Create a binding for a different asset.
	err := bs.SetBinding(&EnvBindingRecord{
		ID:          uuid.New().String(),
		Plugin:      "model",
		AssetKind:   "model",
		AssetName:   "llama",
		Environment: "dev",
		AssetUID:    "model:model:llama",
		VersionID:   "v1.0:xyz",
		BoundAt:     now,
		BoundBy:     "bob",
	})
	require.NoError(t, err)

	// List bindings for filesystem.
	bindings, err := bs.ListBindings("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	assert.Len(t, bindings, 3)
	// Should be ordered alphabetically by environment.
	assert.Equal(t, "dev", bindings[0].Environment)
	assert.Equal(t, "prod", bindings[1].Environment)
	assert.Equal(t, "stage", bindings[2].Environment)

	// List bindings for llama.
	bindings, err = bs.ListBindings("default", "model", "model", "llama")
	require.NoError(t, err)
	assert.Len(t, bindings, 1)

	// List bindings for nonexistent asset.
	bindings, err = bs.ListBindings("default", "mcp", "mcpserver", "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, bindings)
}
