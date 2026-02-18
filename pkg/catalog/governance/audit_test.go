package governance

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuditStore(t *testing.T) *AuditStore {
	t.Helper()
	db := newTestDB(t)
	return NewAuditStore(db)
}

func TestAuditStore_Append(t *testing.T) {
	store := newTestAuditStore(t)

	event := &AuditEventRecord{
		ID:        uuid.New().String(),
		EventType: "governance.metadata.changed",
		Actor:     "alice",
		AssetUID:  "mcp:mcpserver:filesystem",
		Action:    "patch",
		Outcome:   "success",
		OldValue:  JSONAny{"riskLevel": "medium"},
		NewValue:  JSONAny{"riskLevel": "high"},
		CreatedAt: time.Now(),
	}

	err := store.Append(event)
	require.NoError(t, err)

	// Verify the event was stored.
	events, _, total, err := store.ListByAsset("mcp:mcpserver:filesystem", 10, "")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, events, 1)
	assert.Equal(t, "governance.metadata.changed", events[0].EventType)
	assert.Equal(t, "alice", events[0].Actor)
	assert.Equal(t, "success", events[0].Outcome)
}

func TestAuditStore_ListByAsset(t *testing.T) {
	store := newTestAuditStore(t)
	assetUID := "mcp:mcpserver:filesystem"

	// Create multiple events with distinct timestamps.
	baseTime := time.Now().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		event := &AuditEventRecord{
			ID:        uuid.New().String(),
			EventType: "governance.metadata.changed",
			Actor:     "alice",
			AssetUID:  assetUID,
			Action:    "patch",
			Outcome:   "success",
			CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}
		require.NoError(t, store.Append(event))
	}

	// Create an event for a different asset.
	require.NoError(t, store.Append(&AuditEventRecord{
		ID:        uuid.New().String(),
		EventType: "governance.metadata.changed",
		Actor:     "bob",
		AssetUID:  "model:model:llama",
		Action:    "patch",
		Outcome:   "success",
		CreatedAt: time.Now(),
	}))

	// List all for the asset.
	events, nextToken, total, err := store.ListByAsset(assetUID, 10, "")
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, events, 5)
	assert.Empty(t, nextToken)

	// Events should be ordered by created_at DESC (newest first).
	for i := 1; i < len(events); i++ {
		assert.True(t, events[i-1].CreatedAt.After(events[i].CreatedAt) || events[i-1].CreatedAt.Equal(events[i].CreatedAt),
			"events should be ordered newest first")
	}

	// Paginate with pageSize 2.
	page1, token1, total1, err := store.ListByAsset(assetUID, 2, "")
	require.NoError(t, err)
	assert.Equal(t, 5, total1)
	assert.Len(t, page1, 2)
	assert.NotEmpty(t, token1)

	page2, token2, _, err := store.ListByAsset(assetUID, 2, token1)
	require.NoError(t, err)
	assert.Len(t, page2, 2)
	assert.NotEmpty(t, token2)

	page3, token3, _, err := store.ListByAsset(assetUID, 2, token2)
	require.NoError(t, err)
	assert.Len(t, page3, 1)
	assert.Empty(t, token3)
}

func TestAuditStore_ListAll(t *testing.T) {
	store := newTestAuditStore(t)

	baseTime := time.Now().Add(-time.Hour)

	// Create events with different types.
	for i := 0; i < 3; i++ {
		require.NoError(t, store.Append(&AuditEventRecord{
			ID:        uuid.New().String(),
			EventType: "governance.metadata.changed",
			Actor:     "alice",
			AssetUID:  "asset-1",
			Outcome:   "success",
			CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}))
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, store.Append(&AuditEventRecord{
			ID:        uuid.New().String(),
			EventType: "governance.lifecycle.transitioned",
			Actor:     "bob",
			AssetUID:  "asset-2",
			Outcome:   "success",
			CreatedAt: baseTime.Add(time.Duration(i+3) * time.Minute),
		}))
	}

	// List all without filter.
	events, _, total, err := store.ListAll(10, "", "")
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, events, 5)

	// Filter by event type.
	events, _, total, err = store.ListAll(10, "", "governance.metadata.changed")
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, events, 3)
	for _, e := range events {
		assert.Equal(t, "governance.metadata.changed", e.EventType)
	}

	events, _, total, err = store.ListAll(10, "", "governance.lifecycle.transitioned")
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, events, 2)

	// Filter with non-matching type.
	events, _, total, err = store.ListAll(10, "", "nonexistent.type")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, events)
}

func TestAuditStore_DeleteOlderThan(t *testing.T) {
	store := newTestAuditStore(t)

	now := time.Now()
	oldTime := now.Add(-100 * 24 * time.Hour)    // 100 days ago
	recentTime := now.Add(-10 * 24 * time.Hour)  // 10 days ago

	// Create old events.
	for i := 0; i < 3; i++ {
		require.NoError(t, store.Append(&AuditEventRecord{
			ID:        uuid.New().String(),
			EventType: "governance.metadata.changed",
			Actor:     "alice",
			AssetUID:  "asset-old",
			Outcome:   "success",
			CreatedAt: oldTime.Add(time.Duration(i) * time.Minute),
		}))
	}

	// Create recent events.
	for i := 0; i < 2; i++ {
		require.NoError(t, store.Append(&AuditEventRecord{
			ID:        uuid.New().String(),
			EventType: "governance.lifecycle.transitioned",
			Actor:     "bob",
			AssetUID:  "asset-recent",
			Outcome:   "success",
			CreatedAt: recentTime.Add(time.Duration(i) * time.Minute),
		}))
	}

	// Delete events older than 90 days.
	cutoff := now.Add(-90 * 24 * time.Hour)
	deleted, err := store.DeleteOlderThan(cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	// Verify only recent events remain.
	events, _, total, err := store.ListAll(10, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, events, 2)
	for _, e := range events {
		assert.Equal(t, "asset-recent", e.AssetUID)
	}
}

func TestAuditStore_DeleteOlderThan_NoMatches(t *testing.T) {
	store := newTestAuditStore(t)

	// Create a recent event.
	require.NoError(t, store.Append(&AuditEventRecord{
		ID:        uuid.New().String(),
		EventType: "governance.metadata.changed",
		Actor:     "alice",
		AssetUID:  "asset-1",
		Outcome:   "success",
		CreatedAt: time.Now(),
	}))

	// Delete with cutoff in the past -- should delete nothing.
	cutoff := time.Now().Add(-365 * 24 * time.Hour)
	deleted, err := store.DeleteOlderThan(cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	// Verify event still exists.
	events, _, total, err := store.ListAll(10, "", "")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, events, 1)
}
