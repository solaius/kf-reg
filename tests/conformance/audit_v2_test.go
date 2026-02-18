package conformance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kubeflow/model-registry/pkg/audit"
	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/catalog/governance"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// setupAuditTestDB creates an in-memory SQLite DB with audit tables.
func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&governance.AuditEventRecord{}); err != nil {
		t.Fatalf("failed to migrate audit table: %v", err)
	}
	return db
}

// TestPhase8AuditEventCapture verifies that the audit middleware captures events
// for management actions with correct fields.
func TestPhase8AuditEventCapture(t *testing.T) {
	db := setupAuditTestDB(t)
	store := governance.NewAuditStore(db)
	cfg := audit.DefaultAuditConfig()

	// Build a handler chain: identity -> tenancy -> audit -> management endpoint.
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"applied"}`))
	})

	r := chi.NewRouter()
	r.Use(authz.IdentityMiddleware())
	r.Use(tenancy.NewMiddleware(tenancy.ModeSingle))
	r.Use(audit.AuditMiddleware(store, cfg, nil))
	r.Post("/api/mcp_catalog/v1alpha1/management/apply-source", inner.ServeHTTP)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Make a management action request.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/mcp_catalog/v1alpha1/management/apply-source", nil)
	req.Header.Set("X-Remote-User", "alice@example.com")
	req.Header.Set("X-Remote-Group", "admins")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify exactly one audit event was created.
	var events []governance.AuditEventRecord
	if err := db.Find(&events).Error; err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}

	event := events[0]
	if event.Namespace != "default" {
		t.Errorf("namespace = %q, want %q", event.Namespace, "default")
	}
	if event.Actor != "alice@example.com" {
		t.Errorf("actor = %q, want %q", event.Actor, "alice@example.com")
	}
	if event.Plugin != "mcp" {
		t.Errorf("plugin = %q, want %q", event.Plugin, "mcp")
	}
	if event.Outcome != "success" {
		t.Errorf("outcome = %q, want %q", event.Outcome, "success")
	}
	if event.StatusCode != http.StatusOK {
		t.Errorf("statusCode = %d, want %d", event.StatusCode, http.StatusOK)
	}
	if event.ActionVerb != "apply-source" {
		t.Errorf("actionVerb = %q, want %q", event.ActionVerb, "apply-source")
	}
	if event.ID == "" {
		t.Error("audit event ID is empty")
	}
}

// TestPhase8AuditSkipsGETRequests verifies that GET requests are not audited.
func TestPhase8AuditSkipsGETRequests(t *testing.T) {
	db := setupAuditTestDB(t)
	store := governance.NewAuditStore(db)
	cfg := audit.DefaultAuditConfig()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := chi.NewRouter()
	r.Use(audit.AuditMiddleware(store, cfg, nil))
	r.Get("/api/mcp_catalog/v1alpha1/management/sources", inner.ServeHTTP)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/mcp_catalog/v1alpha1/management/sources")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()

	var count int64
	db.Model(&governance.AuditEventRecord{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 audit events for GET, got %d", count)
	}
}

// TestPhase8AuditListPagination verifies audit event listing with pagination.
func TestPhase8AuditListPagination(t *testing.T) {
	db := setupAuditTestDB(t)
	store := governance.NewAuditStore(db)

	// Insert 5 audit events with staggered timestamps.
	for i := 0; i < 5; i++ {
		event := &governance.AuditEventRecord{
			ID:         uuid.New().String(),
			Namespace:  "default",
			EventType:  "management",
			Actor:      "alice",
			Action:     "apply-source",
			ActionVerb: "apply-source",
			Outcome:    "success",
			StatusCode: 200,
			Plugin:     "mcp",
			CreatedAt:  time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := store.Append(event); err != nil {
			t.Fatalf("failed to append event: %v", err)
		}
	}

	r := chi.NewRouter()
	r.Get("/events", audit.ListEventsHandler(store))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("first page", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/events?pageSize=2")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Events        []map[string]any `json:"events"`
			NextPageToken string           `json:"nextPageToken"`
			TotalSize     int              `json:"totalSize"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if len(result.Events) != 2 {
			t.Errorf("expected 2 events on first page, got %d", len(result.Events))
		}
		if result.TotalSize != 5 {
			t.Errorf("totalSize = %d, want 5", result.TotalSize)
		}
		if result.NextPageToken == "" {
			t.Error("expected non-empty nextPageToken")
		}
	})
}

// TestPhase8AuditListFiltering verifies audit listing filters by namespace, actor, plugin.
func TestPhase8AuditListFiltering(t *testing.T) {
	db := setupAuditTestDB(t)
	store := governance.NewAuditStore(db)

	// Insert events with different attributes.
	events := []governance.AuditEventRecord{
		{ID: uuid.New().String(), Namespace: "team-a", Actor: "alice", Plugin: "mcp", EventType: "management", Action: "apply", ActionVerb: "apply", Outcome: "success", StatusCode: 200, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Namespace: "team-a", Actor: "bob", Plugin: "knowledge", EventType: "management", Action: "refresh", ActionVerb: "refresh", Outcome: "success", StatusCode: 200, CreatedAt: time.Now().Add(time.Second)},
		{ID: uuid.New().String(), Namespace: "team-b", Actor: "alice", Plugin: "mcp", EventType: "management", Action: "delete", ActionVerb: "delete", Outcome: "success", StatusCode: 200, CreatedAt: time.Now().Add(2 * time.Second)},
	}

	for i := range events {
		if err := store.Append(&events[i]); err != nil {
			t.Fatalf("failed to append event: %v", err)
		}
	}

	r := chi.NewRouter()
	r.Get("/events", audit.ListEventsHandler(store))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("filter by namespace", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/events?namespace=team-a")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Events    []map[string]any `json:"events"`
			TotalSize int              `json:"totalSize"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.TotalSize != 2 {
			t.Errorf("filter by namespace=team-a: totalSize = %d, want 2", result.TotalSize)
		}
	})

	t.Run("filter by actor", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/events?actor=alice")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			TotalSize int `json:"totalSize"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.TotalSize != 2 {
			t.Errorf("filter by actor=alice: totalSize = %d, want 2", result.TotalSize)
		}
	})

	t.Run("filter by plugin", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/events?plugin=knowledge")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			TotalSize int `json:"totalSize"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.TotalSize != 1 {
			t.Errorf("filter by plugin=knowledge: totalSize = %d, want 1", result.TotalSize)
		}
	})
}

// TestPhase8AuditDeniedAction verifies that denied (403) actions are captured
// when LogDenied is true.
func TestPhase8AuditDeniedAction(t *testing.T) {
	db := setupAuditTestDB(t)
	store := governance.NewAuditStore(db)
	cfg := audit.DefaultAuditConfig()
	cfg.LogDenied = true

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	})

	r := chi.NewRouter()
	r.Use(authz.IdentityMiddleware())
	r.Use(tenancy.NewMiddleware(tenancy.ModeSingle))
	r.Use(audit.AuditMiddleware(store, cfg, nil))
	r.Post("/api/mcp_catalog/v1alpha1/management/apply-source", inner.ServeHTTP)

	ts := httptest.NewServer(r)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/mcp_catalog/v1alpha1/management/apply-source", nil)
	req.Header.Set("X-Remote-User", "unauthorized-user")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()

	var events []governance.AuditEventRecord
	if err := db.Find(&events).Error; err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 audit event for denied action, got %d", len(events))
	}

	if events[0].Outcome != "denied" {
		t.Errorf("outcome = %q, want %q", events[0].Outcome, "denied")
	}
	if events[0].StatusCode != http.StatusForbidden {
		t.Errorf("statusCode = %d, want %d", events[0].StatusCode, http.StatusForbidden)
	}
}

// TestPhase8AuditDisabled verifies that no events are captured when audit is disabled.
func TestPhase8AuditDisabled(t *testing.T) {
	db := setupAuditTestDB(t)
	store := governance.NewAuditStore(db)
	cfg := audit.DefaultAuditConfig()
	cfg.Enabled = false

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := chi.NewRouter()
	r.Use(audit.AuditMiddleware(store, cfg, nil))
	r.Post("/api/mcp_catalog/v1alpha1/management/apply-source", inner.ServeHTTP)

	ts := httptest.NewServer(r)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/mcp_catalog/v1alpha1/management/apply-source", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()

	var count int64
	db.Model(&governance.AuditEventRecord{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 events when audit disabled, got %d", count)
	}
}
