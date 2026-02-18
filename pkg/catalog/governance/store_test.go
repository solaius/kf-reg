package governance

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB creates an in-memory SQLite DB with governance tables migrated.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	store := NewGovernanceStore(db)
	require.NoError(t, store.AutoMigrate())
	return db
}

func newTestGovernanceStore(t *testing.T) *GovernanceStore {
	t.Helper()
	db := newTestDB(t)
	return NewGovernanceStore(db)
}

func TestGovernanceStore_CRUD(t *testing.T) {
	store := newTestGovernanceStore(t)

	// Create a record.
	record := &AssetGovernanceRecord{
		ID:                 "test-id-1",
		Plugin:             "mcp",
		AssetKind:          "mcpserver",
		AssetName:          "filesystem",
		AssetUID:           "mcp:mcpserver:filesystem",
		RiskLevel:          "high",
		LifecycleState:     "draft",
		LifecycleChangedBy: "alice",
		OwnerPrincipal:     "alice@example.com",
		OwnerDisplayName:   "Alice",
		TeamName:           "ml-platform",
		ComplianceTags:     JSONStringSlice{"sox", "gdpr"},
	}
	err := store.Upsert(record)
	require.NoError(t, err)

	// Get the record.
	got, err := store.Get("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "test-id-1", got.ID)
	assert.Equal(t, "mcp", got.Plugin)
	assert.Equal(t, "mcpserver", got.AssetKind)
	assert.Equal(t, "filesystem", got.AssetName)
	assert.Equal(t, "high", got.RiskLevel)
	assert.Equal(t, "draft", got.LifecycleState)
	assert.Equal(t, "alice@example.com", got.OwnerPrincipal)
	assert.Equal(t, "Alice", got.OwnerDisplayName)
	assert.Equal(t, "ml-platform", got.TeamName)
	assert.Equal(t, JSONStringSlice{"sox", "gdpr"}, got.ComplianceTags)

	// Update via upsert.
	record.RiskLevel = "critical"
	record.OwnerDisplayName = "Alice B"
	err = store.Upsert(record)
	require.NoError(t, err)

	got, err = store.Get("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "critical", got.RiskLevel)
	assert.Equal(t, "Alice B", got.OwnerDisplayName)

	// Delete the record.
	err = store.Delete("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)

	got, err = store.Get("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGovernanceStore_GetByUID(t *testing.T) {
	store := newTestGovernanceStore(t)

	record := &AssetGovernanceRecord{
		ID:                 "test-id-2",
		Plugin:             "model",
		AssetKind:          "model",
		AssetName:          "llama-3",
		AssetUID:           "model:model:llama-3",
		RiskLevel:          "medium",
		LifecycleState:     "approved",
		LifecycleChangedBy: "system",
	}
	require.NoError(t, store.Upsert(record))

	got, err := store.GetByUID("model:model:llama-3")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "llama-3", got.AssetName)

	// Not found.
	got, err = store.GetByUID("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGovernanceStore_Get_NotFound(t *testing.T) {
	store := newTestGovernanceStore(t)

	got, err := store.Get("default", "mcp", "mcpserver", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGovernanceStore_Delete_NonExistent(t *testing.T) {
	store := newTestGovernanceStore(t)

	err := store.Delete("default", "mcp", "mcpserver", "nonexistent")
	require.NoError(t, err)
}

func TestGovernanceStore_EnsureExists(t *testing.T) {
	store := newTestGovernanceStore(t)

	// First call creates a new record with defaults.
	record, err := store.EnsureExists("default", "mcp", "mcpserver", "filesystem", "", "alice")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, "mcp", record.Plugin)
	assert.Equal(t, "mcpserver", record.AssetKind)
	assert.Equal(t, "filesystem", record.AssetName)
	assert.Equal(t, "mcp:mcpserver:filesystem", record.AssetUID)
	assert.Equal(t, "medium", record.RiskLevel)
	assert.Equal(t, "draft", record.LifecycleState)
	assert.Equal(t, "alice", record.LifecycleChangedBy)
	assert.NotEmpty(t, record.ID)

	// Second call returns the existing record.
	record2, err := store.EnsureExists("default", "mcp", "mcpserver", "filesystem", "", "bob")
	require.NoError(t, err)
	require.NotNil(t, record2)
	assert.Equal(t, record.ID, record2.ID)
	assert.Equal(t, "alice", record2.LifecycleChangedBy) // unchanged

	// With explicit UID.
	record3, err := store.EnsureExists("default", "model", "model", "llama", "custom-uid-123", "charlie")
	require.NoError(t, err)
	require.NotNil(t, record3)
	assert.Equal(t, "custom-uid-123", record3.AssetUID)
}

func TestGovernanceStore_List(t *testing.T) {
	store := newTestGovernanceStore(t)

	// Create multiple records for the same plugin.
	for i := 0; i < 5; i++ {
		record := &AssetGovernanceRecord{
			ID:                 fmt.Sprintf("id-%d", i),
			Plugin:             "mcp",
			AssetKind:          "mcpserver",
			AssetName:          fmt.Sprintf("server-%d", i),
			AssetUID:           fmt.Sprintf("mcp:mcpserver:server-%d", i),
			RiskLevel:          "medium",
			LifecycleState:     "draft",
			LifecycleChangedBy: "system",
		}
		require.NoError(t, store.Upsert(record))
	}

	// Create a record for a different plugin.
	require.NoError(t, store.Upsert(&AssetGovernanceRecord{
		ID:                 "other-id",
		Plugin:             "model",
		AssetKind:          "model",
		AssetName:          "llama",
		AssetUID:           "model:model:llama",
		RiskLevel:          "low",
		LifecycleState:     "approved",
		LifecycleChangedBy: "system",
	}))

	// List all for "mcp" plugin.
	records, nextToken, err := store.List("default", "mcp", 10, "")
	require.NoError(t, err)
	assert.Len(t, records, 5)
	assert.Empty(t, nextToken)

	// Paginate with pageSize 2.
	page1, token1, err := store.List("default", "mcp", 2, "")
	require.NoError(t, err)
	assert.Len(t, page1, 2)
	assert.NotEmpty(t, token1)

	page2, token2, err := store.List("default", "mcp", 2, token1)
	require.NoError(t, err)
	assert.Len(t, page2, 2)
	assert.NotEmpty(t, token2)

	page3, token3, err := store.List("default", "mcp", 2, token2)
	require.NoError(t, err)
	assert.Len(t, page3, 1)
	assert.Empty(t, token3)

	// List for "model" plugin.
	modelRecords, _, err := store.List("default", "model", 10, "")
	require.NoError(t, err)
	assert.Len(t, modelRecords, 1)

	// List for non-existent plugin.
	emptyRecords, _, err := store.List("default", "nonexistent", 10, "")
	require.NoError(t, err)
	assert.Empty(t, emptyRecords)
}

func TestGovernanceStore_TenantIsolation(t *testing.T) {
	store := newTestGovernanceStore(t)

	// Create records in two namespaces for the same plugin/kind/name.
	recA, err := store.EnsureExists("ns-a", "mcp", "mcpserver", "filesystem", "", "alice")
	require.NoError(t, err)
	require.NotNil(t, recA)
	assert.Equal(t, "ns-a", recA.Namespace)

	recB, err := store.EnsureExists("ns-b", "mcp", "mcpserver", "filesystem", "", "bob")
	require.NoError(t, err)
	require.NotNil(t, recB)
	assert.Equal(t, "ns-b", recB.Namespace)

	// Records should have different IDs.
	assert.NotEqual(t, recA.ID, recB.ID)

	// Get should be namespace-scoped.
	gotA, err := store.Get("ns-a", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.Equal(t, recA.ID, gotA.ID)

	gotB, err := store.Get("ns-b", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.Equal(t, recB.ID, gotB.ID)

	// ns-a record should be invisible from ns-b and vice versa.
	assert.NotEqual(t, gotA.ID, gotB.ID)

	// List should be namespace-scoped.
	listA, _, err := store.List("ns-a", "mcp", 10, "")
	require.NoError(t, err)
	assert.Len(t, listA, 1)
	assert.Equal(t, recA.ID, listA[0].ID)

	listB, _, err := store.List("ns-b", "mcp", 10, "")
	require.NoError(t, err)
	assert.Len(t, listB, 1)
	assert.Equal(t, recB.ID, listB[0].ID)

	// Delete in ns-a should not affect ns-b.
	err = store.Delete("ns-a", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)

	gotA, err = store.Get("ns-a", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	assert.Nil(t, gotA, "ns-a record should be deleted")

	gotB, err = store.Get("ns-b", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	assert.NotNil(t, gotB, "ns-b record should be unaffected")
}

func TestGovernanceStore_MigrationIdempotent(t *testing.T) {
	db := newTestDB(t) // already migrated

	// Running AutoMigrate again should not error.
	store := NewGovernanceStore(db)
	err := store.AutoMigrate()
	require.NoError(t, err, "AutoMigrate should be idempotent")

	// Verify the store still works.
	record, err := store.EnsureExists("default", "mcp", "mcpserver", "filesystem", "", "alice")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, "default", record.Namespace)
}
