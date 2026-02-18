package conformance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kubeflow/model-registry/pkg/catalog/governance"
	"github.com/kubeflow/model-registry/pkg/jobs"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// TestPhase8TenancySingleMode verifies single-tenant mode via live server.
// Single-tenant mode is the default and should set namespace to "default"
// for all requests without requiring any namespace parameter.
func TestPhase8TenancySingleMode(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	t.Run("plugins endpoint works without namespace", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/api/plugins")
		if err != nil {
			t.Fatalf("GET /api/plugins failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("health endpoints work without namespace", func(t *testing.T) {
		for _, path := range []string{"/livez", "/readyz"} {
			resp, err := http.Get(serverURL + path)
			if err != nil {
				t.Fatalf("GET %s failed: %v", path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
			}
		}
	})
}

// TestPhase8TenancyMiddlewareSingleMode tests the SingleTenantResolver middleware.
// In single-tenant mode, all requests should get namespace="default" without
// requiring any namespace parameter or header.
func TestPhase8TenancyMiddlewareSingleMode(t *testing.T) {
	handler := tenancy.NewMiddleware(tenancy.ModeSingle)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, ok := tenancy.TenantFromContext(r.Context())
		if !ok {
			t.Error("tenant context not set")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"namespace": tc.Namespace,
		})
	}))

	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("no namespace param sets default", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/test")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["namespace"] != "default" {
			t.Errorf("expected namespace=default, got %q", result["namespace"])
		}
	})
}

// TestPhase8TenancyMiddlewareNamespaceMode tests the NamespaceTenantResolver middleware.
// In namespace mode, requests must provide a namespace via query param or header.
func TestPhase8TenancyMiddlewareNamespaceMode(t *testing.T) {
	handler := tenancy.NewMiddleware(tenancy.ModeNamespace)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, ok := tenancy.TenantFromContext(r.Context())
		if !ok {
			t.Error("tenant context not set")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"namespace": tc.Namespace,
		})
	}))

	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("missing namespace returns 400", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/test")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing namespace, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["error"] != "bad_request" {
			t.Errorf("expected error=bad_request, got %q", result["error"])
		}
	})

	t.Run("namespace via query param", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/test?namespace=team-a")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["namespace"] != "team-a" {
			t.Errorf("expected namespace=team-a, got %q", result["namespace"])
		}
	})

	t.Run("namespace via header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
		req.Header.Set("X-Namespace", "team-b")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["namespace"] != "team-b" {
			t.Errorf("expected namespace=team-b, got %q", result["namespace"])
		}
	})

	t.Run("query param takes precedence over header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test?namespace=from-query", nil)
		req.Header.Set("X-Namespace", "from-header")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["namespace"] != "from-query" {
			t.Errorf("expected namespace=from-query, got %q", result["namespace"])
		}
	})

	t.Run("invalid namespace returns 400", func(t *testing.T) {
		tests := []struct {
			name string
			ns   string
		}{
			{"uppercase", "Team-A"},
			{"special chars", "team@a"},
			{"starts with hyphen", "-team"},
			{"ends with hyphen", "team-"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resp, err := http.Get(ts.URL + "/test?namespace=" + tt.ns)
				if err != nil {
					t.Fatalf("GET failed: %v", err)
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("expected 400 for namespace %q, got %d", tt.ns, resp.StatusCode)
				}
			})
		}
	})
}

// TestPhase8TenancyIsolation verifies that team-a cannot see team-b data.
// This tests the namespace-scoped data isolation pattern by using separate
// DB-backed stores for audit events and jobs, each scoped by namespace.
func TestPhase8TenancyIsolation(t *testing.T) {
	// Set up in-memory DB with audit and jobs tables.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&governance.AuditEventRecord{}, &jobs.RefreshJob{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	auditStore := governance.NewAuditStore(db)
	jobStore := jobs.NewJobStore(db)

	// Insert audit events in team-a and team-b.
	for _, ns := range []string{"team-a", "team-b"} {
		for i := 0; i < 3; i++ {
			event := &governance.AuditEventRecord{
				ID:         uuid.New().String(),
				Namespace:  ns,
				EventType:  "management",
				Actor:      ns + "-user",
				Action:     "apply-source",
				ActionVerb: "apply-source",
				Outcome:    "success",
				StatusCode: 200,
				Plugin:     "mcp",
				CreatedAt:  time.Now().Add(time.Duration(i) * time.Second),
			}
			if err := auditStore.Append(event); err != nil {
				t.Fatalf("failed to append audit event: %v", err)
			}
		}
	}

	// Insert jobs in team-a and team-b.
	for _, ns := range []string{"team-a", "team-b"} {
		for i := 0; i < 2; i++ {
			job := &jobs.RefreshJob{
				ID:             uuid.New().String(),
				Namespace:      ns,
				Plugin:         "mcp",
				SourceID:       "src",
				RequestedBy:    ns + "-user",
				RequestedAt:    time.Now().Add(time.Duration(i) * time.Second),
				State:          jobs.JobStateQueued,
				IdempotencyKey: uuid.New().String(),
			}
			if _, err := jobStore.Enqueue(job); err != nil {
				t.Fatalf("failed to enqueue job: %v", err)
			}
		}
	}

	t.Run("audit events scoped by namespace", func(t *testing.T) {
		// team-a should only see team-a events.
		teamAEvents, _, teamATotal, err := auditStore.ListFiltered(governance.AuditListFilter{
			Namespace: "team-a",
		}, 20, "")
		if err != nil {
			t.Fatalf("ListFiltered error: %v", err)
		}
		if teamATotal != 3 {
			t.Errorf("team-a total = %d, want 3", teamATotal)
		}
		for _, evt := range teamAEvents {
			if evt.Namespace != "team-a" {
				t.Errorf("team-a query returned event from namespace %q", evt.Namespace)
			}
		}

		// team-b should only see team-b events.
		_, _, teamBTotal, err := auditStore.ListFiltered(governance.AuditListFilter{
			Namespace: "team-b",
		}, 20, "")
		if err != nil {
			t.Fatalf("ListFiltered error: %v", err)
		}
		if teamBTotal != 3 {
			t.Errorf("team-b total = %d, want 3", teamBTotal)
		}

		// team-c should see nothing.
		_, _, teamCTotal, err := auditStore.ListFiltered(governance.AuditListFilter{
			Namespace: "team-c",
		}, 20, "")
		if err != nil {
			t.Fatalf("ListFiltered error: %v", err)
		}
		if teamCTotal != 0 {
			t.Errorf("team-c total = %d, want 0", teamCTotal)
		}
	})

	t.Run("jobs scoped by namespace", func(t *testing.T) {
		// team-a should only see team-a jobs.
		teamAJobs, _, teamATotal, err := jobStore.List(jobs.JobListFilter{
			Namespace: "team-a",
		}, 20, "")
		if err != nil {
			t.Fatalf("List error: %v", err)
		}
		if teamATotal != 2 {
			t.Errorf("team-a total = %d, want 2", teamATotal)
		}
		for _, j := range teamAJobs {
			if j.Namespace != "team-a" {
				t.Errorf("team-a query returned job from namespace %q", j.Namespace)
			}
		}

		// team-b should only see team-b jobs.
		_, _, teamBTotal, err := jobStore.List(jobs.JobListFilter{
			Namespace: "team-b",
		}, 20, "")
		if err != nil {
			t.Fatalf("List error: %v", err)
		}
		if teamBTotal != 2 {
			t.Errorf("team-b total = %d, want 2", teamBTotal)
		}
	})

	t.Run("namespace middleware enforces isolation at HTTP level", func(t *testing.T) {
		// Build a namespace-aware handler that returns namespace-scoped data.
		var mu sync.Mutex
		nsData := map[string][]string{
			"team-a": {"model-a-1", "model-a-2"},
			"team-b": {"model-b-1"},
		}

		r := chi.NewRouter()
		r.Use(tenancy.Middleware(tenancy.NamespaceTenantResolver{}))
		r.Get("/api/models", func(w http.ResponseWriter, r *http.Request) {
			ns := tenancy.NamespaceFromContext(r.Context())
			mu.Lock()
			items := nsData[ns]
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": items,
				"size":  len(items),
			})
		})

		ts := httptest.NewServer(r)
		defer ts.Close()

		// team-a sees 2 models.
		resp, err := http.Get(ts.URL + "/api/models?namespace=team-a")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()
		var teamAResult struct {
			Items []string `json:"items"`
			Size  int      `json:"size"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&teamAResult); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if teamAResult.Size != 2 {
			t.Errorf("team-a size = %d, want 2", teamAResult.Size)
		}

		// team-b sees 1 model.
		resp2, err := http.Get(ts.URL + "/api/models?namespace=team-b")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp2.Body.Close()
		var teamBResult struct {
			Items []string `json:"items"`
			Size  int      `json:"size"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&teamBResult); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if teamBResult.Size != 1 {
			t.Errorf("team-b size = %d, want 1", teamBResult.Size)
		}

		// team-a items should not appear in team-b response.
		for _, item := range teamBResult.Items {
			if item == "model-a-1" || item == "model-a-2" {
				t.Errorf("team-b response contains team-a item %q", item)
			}
		}
	})
}

// TestPhase8TenancyContextPropagation verifies TenantContext round-trips through context.
func TestPhase8TenancyContextPropagation(t *testing.T) {
	tc := tenancy.TenantContext{
		Namespace: "team-alpha",
		User:      "alice",
		Groups:    []string{"admins", "ml-team"},
	}

	ctx := tenancy.WithTenant(context.Background(), tc)

	got, ok := tenancy.TenantFromContext(ctx)
	if !ok {
		t.Fatal("expected tenant context to be present")
	}
	if got.Namespace != tc.Namespace {
		t.Errorf("namespace = %q, want %q", got.Namespace, tc.Namespace)
	}
	if got.User != tc.User {
		t.Errorf("user = %q, want %q", got.User, tc.User)
	}
	if len(got.Groups) != len(tc.Groups) {
		t.Errorf("groups length = %d, want %d", len(got.Groups), len(tc.Groups))
	}

	// Verify NamespaceFromContext convenience function.
	ns := tenancy.NamespaceFromContext(ctx)
	if ns != "team-alpha" {
		t.Errorf("NamespaceFromContext = %q, want %q", ns, "team-alpha")
	}

	// Verify empty context returns empty namespace.
	ns = tenancy.NamespaceFromContext(context.Background())
	if ns != "" {
		t.Errorf("NamespaceFromContext on empty ctx = %q, want empty", ns)
	}
}
