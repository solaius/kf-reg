package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB creates an in-memory SQLite GORM database with the
// RefreshStatusRecord table auto-migrated.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RefreshStatusRecord{}))
	return db
}

// newTestServer creates a Server backed by an in-memory SQLite DB.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	return NewServer(cfg, nil, db, slog.Default())
}

func TestSaveAndGetRefreshStatus(t *testing.T) {
	srv := newTestServer(t)

	result := &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 6,
		Duration:       150 * time.Millisecond,
	}

	srv.saveRefreshStatus("default", "mcp", "src1", result)

	record := srv.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record)
	assert.Equal(t, "src1", record.SourceID)
	assert.Equal(t, "mcp", record.PluginName)
	assert.Equal(t, "success", record.LastRefreshStatus)
	assert.Equal(t, 6, record.EntitiesLoaded)
	assert.Equal(t, int64(150), record.DurationMs)
	assert.Contains(t, record.LastRefreshSummary, "Loaded 6 entities")
	assert.NotNil(t, record.LastRefreshTime)
	assert.Empty(t, record.LastError)
}

func TestSaveRefreshStatus_Error(t *testing.T) {
	srv := newTestServer(t)

	result := &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 0,
		Duration:       50 * time.Millisecond,
		Error:          "connection refused",
	}

	srv.saveRefreshStatus("default", "mcp", "src1", result)

	record := srv.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record)
	assert.Equal(t, "error", record.LastRefreshStatus)
	assert.Equal(t, "connection refused", record.LastError)
	assert.Equal(t, "Refresh failed", record.LastRefreshSummary)
}

func TestSaveRefreshStatus_Upsert(t *testing.T) {
	srv := newTestServer(t)

	// First save.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 3,
		Duration:       100 * time.Millisecond,
	})

	// Second save should update, not create a duplicate.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 7,
		Duration:       200 * time.Millisecond,
	})

	record := srv.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record)
	assert.Equal(t, 7, record.EntitiesLoaded)
	assert.Equal(t, int64(200), record.DurationMs)

	// Verify only one record exists.
	records := srv.listRefreshStatuses("default", "mcp")
	assert.Len(t, records, 1)
}

func TestListRefreshStatuses(t *testing.T) {
	srv := newTestServer(t)

	// Save statuses for two sources.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 3,
		Duration:       100 * time.Millisecond,
	})
	srv.saveRefreshStatus("default", "mcp", "src2", &RefreshResult{
		SourceID:       "src2",
		EntitiesLoaded: 5,
		Duration:       200 * time.Millisecond,
	})
	// Save for a different plugin - should not appear.
	srv.saveRefreshStatus("default", "other", "src3", &RefreshResult{
		SourceID:       "src3",
		EntitiesLoaded: 1,
		Duration:       50 * time.Millisecond,
	})

	records := srv.listRefreshStatuses("default", "mcp")
	assert.Len(t, records, 2)

	otherRecords := srv.listRefreshStatuses("default", "other")
	assert.Len(t, otherRecords, 1)
}

func TestGetRefreshStatus_NotFound(t *testing.T) {
	srv := newTestServer(t)

	record := srv.getRefreshStatus("default", "mcp", "nonexistent")
	assert.Nil(t, record)
}

func TestRefreshStatus_NilDBIsNoop(t *testing.T) {
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	srv := NewServer(cfg, nil, nil, slog.Default()) // no DB

	// These should not panic.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{EntitiesLoaded: 1})
	assert.Nil(t, srv.getRefreshStatus("default", "mcp", "src1"))
	assert.Nil(t, srv.listRefreshStatuses("default", "mcp"))
}

func TestRefreshStatus_NilResultIsNoop(t *testing.T) {
	srv := newTestServer(t)

	// Saving nil result should not panic or create a record.
	srv.saveRefreshStatus("default", "mcp", "src1", nil)
	assert.Nil(t, srv.getRefreshStatus("default", "mcp", "src1"))
}

func TestDeleteRefreshStatus(t *testing.T) {
	srv := newTestServer(t)

	// Save a record, then delete it.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 5,
		Duration:       100 * time.Millisecond,
	})

	// Verify it exists.
	record := srv.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record)

	// Delete it.
	srv.deleteRefreshStatus("default", "mcp", "src1")

	// Verify it's gone.
	record = srv.getRefreshStatus("default", "mcp", "src1")
	assert.Nil(t, record)
}

func TestDeleteRefreshStatus_NonExistent(t *testing.T) {
	srv := newTestServer(t)

	// Deleting a non-existent record should not error.
	srv.deleteRefreshStatus("default", "mcp", "nonexistent")

	// Verify no records exist.
	records := srv.listRefreshStatuses("default", "mcp")
	assert.Empty(t, records)
}

func TestDeleteRefreshStatus_NilDB(t *testing.T) {
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	srv := NewServer(cfg, nil, nil, slog.Default()) // no DB

	// Should not panic.
	srv.deleteRefreshStatus("default", "mcp", "src1")
	srv.deleteAllRefreshStatuses("mcp")
}

func TestDeleteAllRefreshStatuses(t *testing.T) {
	srv := newTestServer(t)

	// Save records for two different plugins.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 3,
		Duration:       100 * time.Millisecond,
	})
	srv.saveRefreshStatus("default", "mcp", "src2", &RefreshResult{
		SourceID:       "src2",
		EntitiesLoaded: 5,
		Duration:       200 * time.Millisecond,
	})
	srv.saveRefreshStatus("default", "other", "src3", &RefreshResult{
		SourceID:       "src3",
		EntitiesLoaded: 1,
		Duration:       50 * time.Millisecond,
	})

	// Delete all records for the "mcp" plugin.
	srv.deleteAllRefreshStatuses("mcp")

	// Verify "mcp" records are gone.
	records := srv.listRefreshStatuses("default", "mcp")
	assert.Empty(t, records)

	// Verify "other" plugin records are untouched.
	otherRecords := srv.listRefreshStatuses("default", "other")
	assert.Len(t, otherRecords, 1)
	assert.Equal(t, "src3", otherRecords[0].SourceID)
}

func TestDeleteSourceHandler_CleansUpRefreshStatus(t *testing.T) {
	srv := newTestServer(t)

	// Pre-populate a refresh status record for the source.
	srv.saveRefreshStatus("default", "test", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 10,
		Duration:       300 * time.Millisecond,
	})

	// Verify the record exists.
	record := srv.getRefreshStatus("default", "test", "src1")
	require.NotNil(t, record)

	p := &mgmtTestPlugin{}

	r := chi.NewRouter()
	r.Delete("/sources/{sourceId}", deleteSourceHandler(p, srv, "test", "test"))

	req := httptest.NewRequest("DELETE", "/sources/src1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the refresh status record was cleaned up.
	record = srv.getRefreshStatus("default", "test", "src1")
	assert.Nil(t, record, "refresh status should be deleted when source is deleted")
}

func TestSourcesListHandler_EnrichedWithPersistedStatus(t *testing.T) {
	srv := newTestServer(t)

	// Pre-populate refresh status in the DB.
	refreshTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	srv.db.Create(&RefreshStatusRecord{
		Namespace:          "default",
		SourceID:           "src1",
		PluginName:         "test",
		LastRefreshTime:    &refreshTime,
		LastRefreshStatus:  "success",
		LastRefreshSummary: "Loaded 10 entities",
		EntitiesLoaded:     10,
		DurationMs:         500,
	})

	p := &mgmtTestPlugin{
		sources: []SourceInfo{
			{
				ID:      "src1",
				Name:    "Source One",
				Type:    "yaml",
				Enabled: true,
				Status:  SourceStatus{State: "available"},
			},
			{
				ID:      "src2",
				Name:    "Source Two",
				Type:    "http",
				Enabled: true,
				Status:  SourceStatus{State: "available"},
			},
		},
	}

	handler := sourcesListHandler(p, srv, "test")
	req := httptest.NewRequest("GET", "/sources", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result struct {
		Sources []SourceInfo `json:"sources"`
		Count   int          `json:"count"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Count)

	// src1 should have persisted refresh status.
	src1 := result.Sources[0]
	assert.Equal(t, "src1", src1.ID)
	assert.Equal(t, "success", src1.Status.LastRefreshStatus)
	assert.Equal(t, "Loaded 10 entities", src1.Status.LastRefreshSummary)
	require.NotNil(t, src1.Status.LastRefreshTime)

	// src2 should have no refresh status since none was saved.
	src2 := result.Sources[1]
	assert.Equal(t, "src2", src2.ID)
	assert.Empty(t, src2.Status.LastRefreshStatus)
	assert.Nil(t, src2.Status.LastRefreshTime)
}

func TestRefreshSourceHandler_PersistsStatus(t *testing.T) {
	srv := newTestServer(t)

	p := &mgmtTestPlugin{}

	r := chi.NewRouter()
	r.Post("/refresh/{sourceId}", refreshSourceHandler(p, nil, "test", srv))

	req := httptest.NewRequest("POST", "/refresh/src1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the refresh status was persisted.
	record := srv.getRefreshStatus("default", "test", "src1")
	require.NotNil(t, record, "refresh status should be persisted after refresh")
	assert.Equal(t, "success", record.LastRefreshStatus)
	assert.Equal(t, 5, record.EntitiesLoaded)
	assert.NotNil(t, record.LastRefreshTime)
}

func TestApplyHandler_RefreshAfterApply_PersistsStatus(t *testing.T) {
	srv := newTestServer(t)

	p := &mgmtTestPlugin{}

	handler := applyHandler(p, srv, "test", p)

	refreshAfterApply := true
	body := SourceConfigInput{
		ID:                "src1",
		Name:              "Test Source",
		Type:              "yaml",
		RefreshAfterApply: &refreshAfterApply,
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result ApplyResult
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "applied", result.Status)
	require.NotNil(t, result.RefreshResult)

	// Verify the refresh status was persisted.
	record := srv.getRefreshStatus("default", "test", "src1")
	require.NotNil(t, record, "refresh status should be persisted after apply with refreshAfterApply")
	assert.Equal(t, "success", record.LastRefreshStatus)
	assert.Equal(t, 5, record.EntitiesLoaded)
}

func TestRefreshAllHandler_PersistsStatus(t *testing.T) {
	srv := newTestServer(t)

	p := &mgmtTestPlugin{}
	handler := refreshAllHandler(p, nil, "test", srv)

	req := httptest.NewRequest("POST", "/refresh", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the refresh status was persisted under the "_all" key.
	record := srv.getRefreshStatus("default", "test", "_all")
	require.NotNil(t, record, "refresh-all status should be persisted")
	assert.Equal(t, "success", record.LastRefreshStatus)
	assert.Equal(t, 10, record.EntitiesLoaded)
}

func TestFormatRefreshSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   *RefreshResult
		expected string
	}{
		{
			name:     "success with loaded only",
			result:   &RefreshResult{EntitiesLoaded: 6},
			expected: "Loaded 6 entities",
		},
		{
			name:     "success with loaded and removed",
			result:   &RefreshResult{EntitiesLoaded: 6, EntitiesRemoved: 2},
			expected: "Loaded 6 entities, removed 2",
		},
		{
			name:     "error",
			result:   &RefreshResult{Error: "something went wrong"},
			expected: "Refresh failed",
		},
		{
			name:     "zero entities",
			result:   &RefreshResult{EntitiesLoaded: 0},
			expected: "Loaded 0 entities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatRefreshSummary(tt.result))
		})
	}
}

func TestServerInit_AutoMigratesRefreshStatus(t *testing.T) {
	Reset()

	db := newTestDB(t)

	p := &testPlugin{
		name:    "test",
		version: "v1",
		healthy: true,
	}
	Register(p)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"test": {},
		},
	}

	server := NewServer(cfg, nil, db, slog.Default())
	err := server.Init(context.Background())
	require.NoError(t, err)

	// The table should exist and we should be able to insert.
	server.saveRefreshStatus("default", "test", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 3,
		Duration:       100 * time.Millisecond,
	})

	record := server.getRefreshStatus("default", "test", "src1")
	require.NotNil(t, record)
	assert.Equal(t, 3, record.EntitiesLoaded)

	Reset()
}

func TestCleanupPluginData(t *testing.T) {
	srv := newTestServer(t)

	// Save records for two different plugins.
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 3,
		Duration:       100 * time.Millisecond,
	})
	srv.saveRefreshStatus("default", "mcp", "src2", &RefreshResult{
		SourceID:       "src2",
		EntitiesLoaded: 5,
		Duration:       200 * time.Millisecond,
	})
	srv.saveRefreshStatus("default", "other", "src3", &RefreshResult{
		SourceID:       "src3",
		EntitiesLoaded: 1,
		Duration:       50 * time.Millisecond,
	})

	// CleanupPluginData should remove all records for the "mcp" plugin.
	srv.CleanupPluginData("mcp")

	// Verify "mcp" records are gone.
	records := srv.listRefreshStatuses("default", "mcp")
	assert.Empty(t, records)

	// Verify "other" plugin records are untouched.
	otherRecords := srv.listRefreshStatuses("default", "other")
	assert.Len(t, otherRecords, 1)
	assert.Equal(t, "src3", otherRecords[0].SourceID)
}

func TestCleanupPluginData_NilDB(t *testing.T) {
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	srv := NewServer(cfg, nil, nil, slog.Default()) // no DB

	// Should not panic.
	srv.CleanupPluginData("mcp")
}

func TestRefreshStatusSurvivesServerRestart(t *testing.T) {
	// This test simulates a server restart by creating two Server instances
	// sharing the same database.
	db := newTestDB(t)
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}

	// First server instance saves a refresh status.
	srv1 := NewServer(cfg, nil, db, slog.Default())
	srv1.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 10,
		Duration:       300 * time.Millisecond,
	})

	// Second server instance (simulating restart) should see the same data.
	srv2 := NewServer(cfg, nil, db, slog.Default())
	record := srv2.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record, "refresh status should survive server restart")
	assert.Equal(t, 10, record.EntitiesLoaded)
	assert.Equal(t, "success", record.LastRefreshStatus)
	assert.Contains(t, record.LastRefreshSummary, "Loaded 10 entities")
}

func TestRefreshStatus_TenantIsolation(t *testing.T) {
	srv := newTestServer(t)

	// Save records in two different namespaces for the same plugin and source.
	srv.saveRefreshStatus("ns-a", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 5,
		Duration:       100 * time.Millisecond,
	})
	srv.saveRefreshStatus("ns-b", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 10,
		Duration:       200 * time.Millisecond,
	})

	// Each namespace should see only its own record.
	recordA := srv.getRefreshStatus("ns-a", "mcp", "src1")
	require.NotNil(t, recordA)
	assert.Equal(t, 5, recordA.EntitiesLoaded)
	assert.Equal(t, "ns-a", recordA.Namespace)

	recordB := srv.getRefreshStatus("ns-b", "mcp", "src1")
	require.NotNil(t, recordB)
	assert.Equal(t, 10, recordB.EntitiesLoaded)
	assert.Equal(t, "ns-b", recordB.Namespace)

	// List should be scoped per namespace.
	recordsA := srv.listRefreshStatuses("ns-a", "mcp")
	assert.Len(t, recordsA, 1)

	recordsB := srv.listRefreshStatuses("ns-b", "mcp")
	assert.Len(t, recordsB, 1)

	// Delete in ns-a should not affect ns-b.
	srv.deleteRefreshStatus("ns-a", "mcp", "src1")
	assert.Nil(t, srv.getRefreshStatus("ns-a", "mcp", "src1"))
	assert.NotNil(t, srv.getRefreshStatus("ns-b", "mcp", "src1"))
}

func TestRefreshStatus_MigrationIdempotent(t *testing.T) {
	db := newTestDB(t) // already migrates RefreshStatusRecord

	// Running AutoMigrate a second time should not error.
	err := db.AutoMigrate(&RefreshStatusRecord{})
	require.NoError(t, err, "AutoMigrate should be idempotent")

	// Verify the table still works after double migration.
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	srv := NewServer(cfg, nil, db, slog.Default())
	srv.saveRefreshStatus("default", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 3,
		Duration:       100 * time.Millisecond,
	})

	record := srv.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record)
	assert.Equal(t, 3, record.EntitiesLoaded)
}

func TestRefreshStatus_EmptyNamespaceDefaultsToDefault(t *testing.T) {
	srv := newTestServer(t)

	// Save with empty namespace should default to "default".
	srv.saveRefreshStatus("", "mcp", "src1", &RefreshResult{
		SourceID:       "src1",
		EntitiesLoaded: 7,
		Duration:       100 * time.Millisecond,
	})

	// Should be retrievable with explicit "default" namespace.
	record := srv.getRefreshStatus("default", "mcp", "src1")
	require.NotNil(t, record)
	assert.Equal(t, "default", record.Namespace)
	assert.Equal(t, 7, record.EntitiesLoaded)

	// Should also be retrievable with empty namespace (defaults to "default").
	record2 := srv.getRefreshStatus("", "mcp", "src1")
	require.NotNil(t, record2)
	assert.Equal(t, 7, record2.EntitiesLoaded)
}
