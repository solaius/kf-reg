package plugin

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestOverlayStore creates an OverlayStore backed by an in-memory SQLite DB.
func newTestOverlayStore(t *testing.T) *OverlayStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	store := NewOverlayStore(db)
	require.NoError(t, store.AutoMigrate())
	return store
}

func TestOverlayStore_UpsertAndGet(t *testing.T) {
	store := newTestOverlayStore(t)

	record := &OverlayRecord{
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Tags:       StringSlice{"production", "stable"},
		Annotations: JSONMap{
			"team":  "ml-platform",
			"owner": "alice",
		},
		Labels: JSONMap{
			"env": "prod",
		},
		Lifecycle: "active",
	}

	err := store.Upsert(record)
	require.NoError(t, err)

	// Get the record back.
	got, err := store.Get("default", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "mcp", got.PluginName)
	assert.Equal(t, "mcpserver", got.EntityKind)
	assert.Equal(t, "server-1", got.EntityUID)
	assert.Equal(t, StringSlice{"production", "stable"}, got.Tags)
	assert.Equal(t, JSONMap{"team": "ml-platform", "owner": "alice"}, got.Annotations)
	assert.Equal(t, JSONMap{"env": "prod"}, got.Labels)
	assert.Equal(t, "active", got.Lifecycle)
}

func TestOverlayStore_Upsert_UpdateExisting(t *testing.T) {
	store := newTestOverlayStore(t)

	// Create initial record.
	record := &OverlayRecord{
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Tags:       StringSlice{"v1"},
		Lifecycle:  "active",
	}
	require.NoError(t, store.Upsert(record))

	// Update with new tags.
	record.Tags = StringSlice{"v2", "production"}
	record.Lifecycle = "deprecated"
	require.NoError(t, store.Upsert(record))

	// Verify the update.
	got, err := store.Get("default", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, StringSlice{"v2", "production"}, got.Tags)
	assert.Equal(t, "deprecated", got.Lifecycle)

	// Verify only one record exists.
	records, err := store.ListByPlugin("default", "mcp")
	require.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestOverlayStore_Get_NotFound(t *testing.T) {
	store := newTestOverlayStore(t)

	got, err := store.Get("default", "mcp", "mcpserver", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestOverlayStore_Delete(t *testing.T) {
	store := newTestOverlayStore(t)

	// Create a record.
	record := &OverlayRecord{
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Tags:       StringSlice{"v1"},
	}
	require.NoError(t, store.Upsert(record))

	// Verify it exists.
	got, err := store.Get("default", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	// Delete it.
	err = store.Delete("default", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)

	// Verify it's gone.
	got, err = store.Get("default", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestOverlayStore_Delete_NonExistent(t *testing.T) {
	store := newTestOverlayStore(t)

	// Deleting a non-existent record should not error.
	err := store.Delete("default", "mcp", "mcpserver", "nonexistent")
	require.NoError(t, err)
}

func TestOverlayStore_ListByPlugin(t *testing.T) {
	store := newTestOverlayStore(t)

	// Create records for two plugins.
	require.NoError(t, store.Upsert(&OverlayRecord{
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Tags:       StringSlice{"v1"},
	}))
	require.NoError(t, store.Upsert(&OverlayRecord{
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-2",
		Tags:       StringSlice{"v2"},
	}))
	require.NoError(t, store.Upsert(&OverlayRecord{
		PluginName: "model",
		EntityKind: "model",
		EntityUID:  "model-1",
		Tags:       StringSlice{"latest"},
	}))

	// List for "mcp" plugin.
	mcpRecords, err := store.ListByPlugin("default", "mcp")
	require.NoError(t, err)
	assert.Len(t, mcpRecords, 2)

	// List for "model" plugin.
	modelRecords, err := store.ListByPlugin("default", "model")
	require.NoError(t, err)
	assert.Len(t, modelRecords, 1)

	// List for non-existent plugin.
	emptyRecords, err := store.ListByPlugin("default", "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, emptyRecords)
}

func TestOverlayStore_TenantIsolation(t *testing.T) {
	store := newTestOverlayStore(t)

	// Create records in two namespaces with the same identity.
	recA := &OverlayRecord{
		Namespace:  "ns-a",
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Tags:       StringSlice{"alpha"},
		Lifecycle:  "active",
	}
	require.NoError(t, store.Upsert(recA))

	recB := &OverlayRecord{
		Namespace:  "ns-b",
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Tags:       StringSlice{"beta"},
		Lifecycle:  "deprecated",
	}
	require.NoError(t, store.Upsert(recB))

	// Get should return the correct namespace's record.
	gotA, err := store.Get("ns-a", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.Equal(t, StringSlice{"alpha"}, gotA.Tags)
	assert.Equal(t, "active", gotA.Lifecycle)

	gotB, err := store.Get("ns-b", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.Equal(t, StringSlice{"beta"}, gotB.Tags)
	assert.Equal(t, "deprecated", gotB.Lifecycle)

	// ListByPlugin should be namespace-scoped.
	listA, err := store.ListByPlugin("ns-a", "mcp")
	require.NoError(t, err)
	assert.Len(t, listA, 1)

	listB, err := store.ListByPlugin("ns-b", "mcp")
	require.NoError(t, err)
	assert.Len(t, listB, 1)

	// Delete in ns-a should not affect ns-b.
	err = store.Delete("ns-a", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	gotA, err = store.Get("ns-a", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	assert.Nil(t, gotA)
	gotB, err = store.Get("ns-b", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	assert.NotNil(t, gotB)
}

func TestOverlayStore_NilFields(t *testing.T) {
	store := newTestOverlayStore(t)

	// Create a record with nil tags, annotations, labels.
	record := &OverlayRecord{
		PluginName: "mcp",
		EntityKind: "mcpserver",
		EntityUID:  "server-1",
		Lifecycle:  "active",
	}
	require.NoError(t, store.Upsert(record))

	got, err := store.Get("default", "mcp", "mcpserver", "server-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.Tags)
	assert.Nil(t, got.Annotations)
	assert.Nil(t, got.Labels)
	assert.Equal(t, "active", got.Lifecycle)
}

func TestBuiltinActionHandler_Tag(t *testing.T) {
	store := newTestOverlayStore(t)
	handler := NewBuiltinActionHandler(store, "mcp", "mcpserver")
	ctx := context.Background()

	t.Run("set tags", func(t *testing.T) {
		req := ActionRequest{
			Action: "tag",
			Params: map[string]any{
				"tags": []any{"production", "v2"},
			},
		}
		result, err := handler.HandleTag(ctx, "server-1", req)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)

		// Verify overlay.
		overlay, err := store.Get("default", "mcp", "mcpserver", "server-1")
		require.NoError(t, err)
		require.NotNil(t, overlay)
		assert.Equal(t, StringSlice{"production", "v2"}, overlay.Tags)
	})

	t.Run("dry-run tag", func(t *testing.T) {
		req := ActionRequest{
			Action: "tag",
			DryRun: true,
			Params: map[string]any{
				"tags": []any{"test"},
			},
		}
		result, err := handler.HandleTag(ctx, "server-2", req)
		require.NoError(t, err)
		assert.Equal(t, "dry-run", result.Status)

		// Should NOT create an overlay.
		overlay, err := store.Get("default", "mcp", "mcpserver", "server-2")
		require.NoError(t, err)
		assert.Nil(t, overlay)
	})

	t.Run("missing tags param", func(t *testing.T) {
		req := ActionRequest{
			Action: "tag",
			Params: map[string]any{},
		}
		_, err := handler.HandleTag(ctx, "server-1", req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})
}

func TestBuiltinActionHandler_Annotate(t *testing.T) {
	store := newTestOverlayStore(t)
	handler := NewBuiltinActionHandler(store, "mcp", "mcpserver")
	ctx := context.Background()

	t.Run("set annotations", func(t *testing.T) {
		req := ActionRequest{
			Action: "annotate",
			Params: map[string]any{
				"annotations": map[string]any{
					"team": "ml-platform",
				},
			},
		}
		result, err := handler.HandleAnnotate(ctx, "server-1", req)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)

		overlay, err := store.Get("default", "mcp", "mcpserver", "server-1")
		require.NoError(t, err)
		require.NotNil(t, overlay)
		assert.Equal(t, "ml-platform", overlay.Annotations["team"])
	})

	t.Run("merge annotations", func(t *testing.T) {
		req := ActionRequest{
			Action: "annotate",
			Params: map[string]any{
				"annotations": map[string]any{
					"owner": "bob",
				},
			},
		}
		result, err := handler.HandleAnnotate(ctx, "server-1", req)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)

		overlay, err := store.Get("default", "mcp", "mcpserver", "server-1")
		require.NoError(t, err)
		require.NotNil(t, overlay)
		// Both annotations should exist.
		assert.Equal(t, "ml-platform", overlay.Annotations["team"])
		assert.Equal(t, "bob", overlay.Annotations["owner"])
	})

	t.Run("dry-run annotate", func(t *testing.T) {
		req := ActionRequest{
			Action: "annotate",
			DryRun: true,
			Params: map[string]any{
				"annotations": map[string]any{"foo": "bar"},
			},
		}
		result, err := handler.HandleAnnotate(ctx, "new-server", req)
		require.NoError(t, err)
		assert.Equal(t, "dry-run", result.Status)

		overlay, err := store.Get("default", "mcp", "mcpserver", "new-server")
		require.NoError(t, err)
		assert.Nil(t, overlay)
	})
}

func TestBuiltinActionHandler_Deprecate(t *testing.T) {
	store := newTestOverlayStore(t)
	handler := NewBuiltinActionHandler(store, "mcp", "mcpserver")
	ctx := context.Background()

	t.Run("deprecate entity", func(t *testing.T) {
		req := ActionRequest{
			Action: "deprecate",
			Params: map[string]any{},
		}
		result, err := handler.HandleDeprecate(ctx, "server-1", req)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, "deprecated", result.Data["lifecycle"])

		overlay, err := store.Get("default", "mcp", "mcpserver", "server-1")
		require.NoError(t, err)
		require.NotNil(t, overlay)
		assert.Equal(t, "deprecated", overlay.Lifecycle)
	})

	t.Run("deprecate with custom phase", func(t *testing.T) {
		req := ActionRequest{
			Action: "deprecate",
			Params: map[string]any{
				"phase": "archived",
			},
		}
		result, err := handler.HandleDeprecate(ctx, "server-2", req)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, "archived", result.Data["lifecycle"])
	})

	t.Run("dry-run deprecate", func(t *testing.T) {
		req := ActionRequest{
			Action: "deprecate",
			DryRun: true,
			Params: map[string]any{},
		}
		result, err := handler.HandleDeprecate(ctx, "server-3", req)
		require.NoError(t, err)
		assert.Equal(t, "dry-run", result.Status)

		overlay, err := store.Get("default", "mcp", "mcpserver", "server-3")
		require.NoError(t, err)
		assert.Nil(t, overlay)
	})
}

func TestBuiltinActionDefinitions(t *testing.T) {
	defs := BuiltinActionDefinitions()
	assert.Len(t, defs, 3)

	ids := make([]string, len(defs))
	for i, d := range defs {
		ids[i] = d.ID
	}
	assert.Contains(t, ids, "tag")
	assert.Contains(t, ids, "annotate")
	assert.Contains(t, ids, "deprecate")

	for _, d := range defs {
		assert.True(t, d.SupportsDryRun, "all builtin actions should support dry-run")
		assert.True(t, d.Idempotent, "all builtin actions should be idempotent")
	}
}
