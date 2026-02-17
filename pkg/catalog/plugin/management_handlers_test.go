package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// mgmtTestPlugin is a mock plugin implementing management interfaces and CatalogPlugin.
type mgmtTestPlugin struct {
	sources    []SourceInfo
	validateFn func(SourceConfigInput) (*ValidationResult, error)
}

// CatalogPlugin interface methods so mgmtTestPlugin can be passed as CatalogPlugin.
func (p *mgmtTestPlugin) Name() string                           { return "test" }
func (p *mgmtTestPlugin) Version() string                        { return "v1" }
func (p *mgmtTestPlugin) Description() string                    { return "test plugin" }
func (p *mgmtTestPlugin) Init(_ context.Context, _ Config) error { return nil }
func (p *mgmtTestPlugin) Start(_ context.Context) error          { return nil }
func (p *mgmtTestPlugin) Stop(_ context.Context) error           { return nil }
func (p *mgmtTestPlugin) Healthy() bool                          { return true }
func (p *mgmtTestPlugin) RegisterRoutes(_ chi.Router) error      { return nil }
func (p *mgmtTestPlugin) Migrations() []Migration                { return nil }

func (p *mgmtTestPlugin) ListSources(_ context.Context) ([]SourceInfo, error) {
	return p.sources, nil
}

func (p *mgmtTestPlugin) ValidateSource(_ context.Context, src SourceConfigInput) (*ValidationResult, error) {
	if p.validateFn != nil {
		return p.validateFn(src)
	}
	return &ValidationResult{Valid: true}, nil
}

func (p *mgmtTestPlugin) ApplySource(_ context.Context, _ SourceConfigInput) error {
	return nil
}

func (p *mgmtTestPlugin) EnableSource(_ context.Context, _ string, _ bool) error {
	return nil
}

func (p *mgmtTestPlugin) DeleteSource(_ context.Context, _ string) error {
	return nil
}

func (p *mgmtTestPlugin) Refresh(_ context.Context, sourceID string) (*RefreshResult, error) {
	return &RefreshResult{
		SourceID:       sourceID,
		EntitiesLoaded: 5,
		Duration:       100 * time.Millisecond,
	}, nil
}

func (p *mgmtTestPlugin) RefreshAll(_ context.Context) (*RefreshResult, error) {
	return &RefreshResult{
		EntitiesLoaded: 10,
		Duration:       200 * time.Millisecond,
	}, nil
}

func (p *mgmtTestPlugin) Diagnostics(_ context.Context) (*PluginDiagnostics, error) {
	return &PluginDiagnostics{
		PluginName: "test",
		Sources: []SourceDiagnostic{
			{ID: "src1", Name: "Source 1", State: "available", EntityCount: 5},
		},
	}, nil
}

func TestSourcesListHandler(t *testing.T) {
	p := &mgmtTestPlugin{
		sources: []SourceInfo{
			{ID: "src1", Name: "Source One", Type: "yaml", Enabled: true, Status: SourceStatus{State: "available"}},
			{ID: "src2", Name: "Source Two", Type: "http", Enabled: false, Status: SourceStatus{State: "disabled"}},
		},
	}

	handler := sourcesListHandler(p, nil, "test")
	req := httptest.NewRequest("GET", "/sources", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result["count"])
	sources, ok := result["sources"].([]any)
	require.True(t, ok)
	assert.Len(t, sources, 2)
}

func TestValidateHandler(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		p := &mgmtTestPlugin{}
		handler := validateHandler(p)

		body := SourceConfigInput{ID: "test", Name: "Test", Type: "yaml"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/validate-source", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result ValidationResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("invalid body", func(t *testing.T) {
		p := &mgmtTestPlugin{}
		handler := validateHandler(p)

		req := httptest.NewRequest("POST", "/validate-source", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestRefreshAllHandler(t *testing.T) {
	p := &mgmtTestPlugin{}
	handler := refreshAllHandler(p, nil, "test", nil)

	req := httptest.NewRequest("POST", "/refresh", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result RefreshResult
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, 10, result.EntitiesLoaded)
}

func TestRefreshSourceHandler(t *testing.T) {
	p := &mgmtTestPlugin{}

	r := chi.NewRouter()
	r.Post("/refresh/{sourceId}", refreshSourceHandler(p, nil, "test", nil))

	req := httptest.NewRequest("POST", "/refresh/src1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result RefreshResult
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "src1", result.SourceID)
	assert.Equal(t, 5, result.EntitiesLoaded)
}

func TestDiagnosticsHandler(t *testing.T) {
	p := &mgmtTestPlugin{}
	handler := diagnosticsHandler(p)

	req := httptest.NewRequest("GET", "/diagnostics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result PluginDiagnostics
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result.PluginName)
	assert.Len(t, result.Sources, 1)
}

func TestEnableHandler(t *testing.T) {
	p := &mgmtTestPlugin{}

	r := chi.NewRouter()
	r.Post("/sources/{sourceId}/enable", enableHandler(p, nil, "test"))

	body := `{"enabled": false}`
	req := httptest.NewRequest("POST", "/sources/src1/enable", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "updated", result["status"])
	assert.Equal(t, false, result["enabled"])
}

func TestDeleteSourceHandler(t *testing.T) {
	p := &mgmtTestPlugin{}

	r := chi.NewRouter()
	r.Delete("/sources/{sourceId}", deleteSourceHandler(p, nil, "test", "test"))

	req := httptest.NewRequest("DELETE", "/sources/src1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "deleted", result["status"])
}

func TestManagementRouterRBAC(t *testing.T) {
	p := &testMgmtPlugin{}

	r := chi.NewRouter()
	mgmt := managementRouter(p, DefaultRoleExtractor, nil)
	r.Mount("/", mgmt)

	t.Run("viewer can list sources", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sources", nil)
		req.Header.Set(RoleHeader, "viewer")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("viewer cannot apply source", func(t *testing.T) {
		body := `{"id":"test","name":"Test","type":"yaml"}`
		req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader([]byte(body)))
		req.Header.Set(RoleHeader, "viewer")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("operator can apply source", func(t *testing.T) {
		body := `{"id":"test","name":"Test","type":"yaml"}`
		req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader([]byte(body)))
		req.Header.Set(RoleHeader, "operator")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("viewer can get diagnostics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/diagnostics", nil)
		req.Header.Set(RoleHeader, "viewer")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("viewer cannot refresh", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/refresh", nil)
		req.Header.Set(RoleHeader, "viewer")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

// testMgmtPlugin implements all management interfaces.
type testMgmtPlugin struct{}

func (p *testMgmtPlugin) ListSources(_ context.Context) ([]SourceInfo, error) {
	return []SourceInfo{}, nil
}
func (p *testMgmtPlugin) ValidateSource(_ context.Context, _ SourceConfigInput) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}
func (p *testMgmtPlugin) ApplySource(_ context.Context, _ SourceConfigInput) error { return nil }
func (p *testMgmtPlugin) EnableSource(_ context.Context, _ string, _ bool) error   { return nil }
func (p *testMgmtPlugin) DeleteSource(_ context.Context, _ string) error           { return nil }
func (p *testMgmtPlugin) Refresh(_ context.Context, _ string) (*RefreshResult, error) {
	return &RefreshResult{}, nil
}
func (p *testMgmtPlugin) RefreshAll(_ context.Context) (*RefreshResult, error) {
	return &RefreshResult{}, nil
}
func (p *testMgmtPlugin) Diagnostics(_ context.Context) (*PluginDiagnostics, error) {
	return &PluginDiagnostics{PluginName: "test"}, nil
}

// CatalogPlugin interface methods (required for managementRouter type cast)
func (p *testMgmtPlugin) Name() string                              { return "test" }
func (p *testMgmtPlugin) Version() string                           { return "v1" }
func (p *testMgmtPlugin) Description() string                       { return "test plugin" }
func (p *testMgmtPlugin) Init(_ context.Context, _ Config) error    { return nil }
func (p *testMgmtPlugin) Start(_ context.Context) error             { return nil }
func (p *testMgmtPlugin) Stop(_ context.Context) error              { return nil }
func (p *testMgmtPlugin) Healthy() bool                             { return true }
func (p *testMgmtPlugin) RegisterRoutes(_ chi.Router) error         { return nil }
func (p *testMgmtPlugin) Migrations() []Migration                   { return nil }
func (p *testMgmtPlugin) BasePath() string                          { return "/api/test_catalog/v1" }

// ensure testMgmtPlugin also satisfies CatalogPlugin
var _ CatalogPlugin = (*testMgmtPlugin)(nil)
var _ SourceManager = (*testMgmtPlugin)(nil)
var _ RefreshProvider = (*testMgmtPlugin)(nil)
var _ DiagnosticsProvider = (*testMgmtPlugin)(nil)

// suppress unused import warning
var _ = fmt.Sprintf

// --- Tests for Phase 4 management endpoints ---

func TestDetailedValidateHandler(t *testing.T) {
	t.Run("valid input returns detailed result", func(t *testing.T) {
		p := &mgmtTestPlugin{}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:validate", detailedValidateHandler(p))

		body := SourceConfigInput{ID: "test", Name: "Test", Type: "yaml"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/sources/test:validate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result DetailedValidationResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.NotEmpty(t, result.LayerResults, "should have per-layer results")
	})

	t.Run("invalid YAML returns errors in result", func(t *testing.T) {
		p := &mgmtTestPlugin{}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:validate", detailedValidateHandler(p))

		body := SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
			Properties: map[string]any{
				"content": "not: [valid: yaml: {{",
			},
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/sources/test:validate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result DetailedValidationResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("bad JSON body returns 400", func(t *testing.T) {
		p := &mgmtTestPlugin{}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:validate", detailedValidateHandler(p))

		req := httptest.NewRequest("POST", "/sources/test:validate", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestRevisionsHandler(t *testing.T) {
	t.Run("no config store returns empty list", func(t *testing.T) {
		handler := revisionsHandler(nil)
		req := httptest.NewRequest("GET", "/sources/test/revisions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(0), result["count"])
		revisions, ok := result["revisions"].([]any)
		require.True(t, ok)
		assert.Empty(t, revisions)
	})

	t.Run("with config store returns revisions", func(t *testing.T) {
		store := &mockConfigStore{
			revisions: []ConfigRevision{
				{Version: "abc123", Size: 100},
				{Version: "def456", Size: 200},
			},
		}
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil, WithConfigStore(store))

		handler := revisionsHandler(srv)
		req := httptest.NewRequest("GET", "/sources/test/revisions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(2), result["count"])
	})

	t.Run("server with nil config store returns empty", func(t *testing.T) {
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil) // no WithConfigStore

		handler := revisionsHandler(srv)
		req := httptest.NewRequest("GET", "/sources/test/revisions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(0), result["count"])
	})
}

func TestRollbackHandler(t *testing.T) {
	t.Run("no config store returns 400", func(t *testing.T) {
		p := &testMgmtPlugin{}
		handler := rollbackHandler(nil, "test", p)

		body := `{"version": "abc123"}`
		req := httptest.NewRequest("POST", "/sources/test:rollback", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("missing version returns 400", func(t *testing.T) {
		store := &mockConfigStore{}
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil, WithConfigStore(store))

		p := &testMgmtPlugin{}
		handler := rollbackHandler(srv, "test", p)

		body := `{}`
		req := httptest.NewRequest("POST", "/sources/test:rollback", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("unknown version returns 404", func(t *testing.T) {
		store := &mockConfigStore{
			rollbackErr: ErrRevisionNotFound,
		}
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil, WithConfigStore(store))

		p := &testMgmtPlugin{}
		handler := rollbackHandler(srv, "test", p)

		body := `{"version": "nonexistent"}`
		req := httptest.NewRequest("POST", "/sources/test:rollback", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("version conflict returns 409", func(t *testing.T) {
		store := &mockConfigStore{
			rollbackErr: ErrVersionConflict,
		}
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil, WithConfigStore(store))

		p := &testMgmtPlugin{}
		handler := rollbackHandler(srv, "test", p)

		body := `{"version": "abc123"}`
		req := httptest.NewRequest("POST", "/sources/test:rollback", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("successful rollback returns 200", func(t *testing.T) {
		restoredCfg := &CatalogSourcesConfig{
			Catalogs: map[string]CatalogSection{
				"test": {Sources: []SourceConfig{{ID: "restored"}}},
			},
		}
		store := &mockConfigStore{
			rollbackCfg:     restoredCfg,
			rollbackVersion: "newver",
		}
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil, WithConfigStore(store))

		p := &testMgmtPlugin{}
		handler := rollbackHandler(srv, "test", p)

		body := `{"version": "abc123"}`
		req := httptest.NewRequest("POST", "/sources/test:rollback", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "rolled_back", result["status"])
		assert.Equal(t, "newver", result["version"])
	})

	t.Run("bad JSON body returns 400", func(t *testing.T) {
		store := &mockConfigStore{}
		cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
		srv := NewServer(cfg, nil, nil, nil, WithConfigStore(store))

		p := &testMgmtPlugin{}
		handler := rollbackHandler(srv, "test", p)

		req := httptest.NewRequest("POST", "/sources/test:rollback", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestApplyHandler_ValidationRejects(t *testing.T) {
	t.Run("invalid input returns 422 with validation result", func(t *testing.T) {
		p := &testMgmtPlugin{}
		handler := applyHandler(p, nil, "test", p)

		// Send input with no ID, name, type to trigger semantic validation failure.
		body := SourceConfigInput{
			Properties: map[string]any{
				"content": "not: [valid: yaml: {{",
			},
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)

		var result DetailedValidationResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("valid input is applied successfully", func(t *testing.T) {
		p := &testMgmtPlugin{}
		handler := applyHandler(p, nil, "test", p)

		body := SourceConfigInput{ID: "test-src", Name: "Test Source", Type: "yaml"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "applied", result["status"])
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		p := &testMgmtPlugin{}
		handler := applyHandler(p, nil, "test", p)

		req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestRefreshRateLimitedHandler(t *testing.T) {
	p := &mgmtTestPlugin{}
	rl := NewRefreshRateLimiter(1 * time.Hour) // very long window

	handler := refreshAllHandler(p, rl, "test", nil)

	// First call should succeed.
	req1 := httptest.NewRequest("POST", "/refresh", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// Second call should be rate limited.
	req2 := httptest.NewRequest("POST", "/refresh", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rr2.Code)
	assert.NotEmpty(t, rr2.Header().Get("Retry-After"))
}

// --- Mock ConfigStore for handler tests ---

type mockConfigStore struct {
	revisions       []ConfigRevision
	rollbackCfg     *CatalogSourcesConfig
	rollbackVersion string
	rollbackErr     error
}

func (m *mockConfigStore) Load(_ context.Context) (*CatalogSourcesConfig, string, error) {
	return &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}, "v1", nil
}

func (m *mockConfigStore) Save(_ context.Context, _ *CatalogSourcesConfig, _ string) (string, error) {
	return "v2", nil
}

func (m *mockConfigStore) Watch(_ context.Context) (<-chan ConfigChangeEvent, error) {
	return nil, nil
}

func (m *mockConfigStore) ListRevisions(_ context.Context) ([]ConfigRevision, error) {
	if m.revisions == nil {
		return []ConfigRevision{}, nil
	}
	return m.revisions, nil
}

func (m *mockConfigStore) Rollback(_ context.Context, _ string) (*CatalogSourcesConfig, string, error) {
	if m.rollbackErr != nil {
		return nil, "", m.rollbackErr
	}
	return m.rollbackCfg, m.rollbackVersion, nil
}

var _ ConfigStore = (*mockConfigStore)(nil)

// --- SecretRef resolution handler-level integration test ---

// secretRefCapturingPlugin is a mock plugin that captures the SourceConfigInput
// it receives in ApplySource, allowing tests to inspect the resolved properties.
type secretRefCapturingPlugin struct {
	testMgmtPlugin
	appliedInput *SourceConfigInput
}

func (p *secretRefCapturingPlugin) ApplySource(_ context.Context, input SourceConfigInput) error {
	p.appliedInput = &input
	return nil
}

// TestApplyHandler_ResolvesSecretRefs verifies the full handler-level flow:
// 1. A Server is created with a K8sSecretResolver backed by a fake K8s Secret
// 2. An apply-source request is sent with SecretRef properties
// 3. The mock plugin receives resolved (plain string) values
// 4. The persisted config retains the original SecretRef objects
func TestApplyHandler_ResolvesSecretRefs(t *testing.T) {
	// Create a fake K8s secret.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hf-credentials",
			Namespace: "kubeflow",
		},
		Data: map[string][]byte{
			"hf-token": []byte("hf_live_token_abc123"),
		},
	}
	k8sClient := fake.NewSimpleClientset(secret)
	resolver := NewK8sSecretResolver(k8sClient, "kubeflow")

	// Create a mock config store that captures the saved config.
	store := &secretRefMockConfigStore{}

	// Create a Server with the SecretResolver and ConfigStore.
	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	srv := NewServer(cfg, nil, nil, nil, WithSecretResolver(resolver), WithConfigStore(store))

	// Create a capturing plugin.
	p := &secretRefCapturingPlugin{}
	configKey := pluginConfigKey(&p.testMgmtPlugin)

	handler := applyHandler(p, srv, configKey, p)

	// Build the request body with a SecretRef and a plain property.
	input := SourceConfigInput{
		ID:   "hf-source",
		Name: "HuggingFace Source",
		Type: "huggingface",
		Properties: map[string]any{
			"url": "https://huggingface.co",
			"token": map[string]any{
				"name": "hf-credentials",
				"key":  "hf-token",
			},
		},
	}
	bodyBytes, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "apply should succeed; body: %s", rr.Body.String())

	var result map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "applied", result["status"])

	// Verify the plugin received RESOLVED values (plain strings, not SecretRef maps).
	require.NotNil(t, p.appliedInput, "plugin ApplySource should have been called")
	assert.Equal(t, "hf_live_token_abc123", p.appliedInput.Properties["token"],
		"plugin should receive the resolved secret value")
	assert.Equal(t, "https://huggingface.co", p.appliedInput.Properties["url"],
		"plain properties should pass through unchanged")

	// Verify the config store received the ORIGINAL input (SecretRef intact).
	require.NotNil(t, store.lastSavedConfig, "config should have been persisted")
	section, ok := store.lastSavedConfig.Catalogs[configKey]
	require.True(t, ok, "persisted config should contain the plugin section")
	require.Len(t, section.Sources, 1, "should have exactly one source")

	persistedProps := section.Sources[0].Properties
	tokenProp, isMap := persistedProps["token"].(map[string]any)
	require.True(t, isMap, "persisted token should still be a SecretRef map, got %T", persistedProps["token"])
	assert.Equal(t, "hf-credentials", tokenProp["name"])
	assert.Equal(t, "hf-token", tokenProp["key"])
}

// TestApplyHandler_SecretRefResolutionFailure verifies that the handler returns
// an error when a SecretRef cannot be resolved (e.g., missing secret).
func TestApplyHandler_SecretRefResolutionFailure(t *testing.T) {
	// Create resolver with NO secrets -> resolution will fail.
	k8sClient := fake.NewSimpleClientset()
	resolver := NewK8sSecretResolver(k8sClient, "default")

	cfg := &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}
	srv := NewServer(cfg, nil, nil, nil, WithSecretResolver(resolver))

	p := &secretRefCapturingPlugin{}
	handler := applyHandler(p, srv, "test", p)

	input := SourceConfigInput{
		ID:   "test-src",
		Name: "Test",
		Type: "yaml",
		Properties: map[string]any{
			"token": map[string]any{
				"name": "nonexistent-secret",
				"key":  "token",
			},
		},
	}
	bodyBytes, _ := json.Marshal(input)

	req := httptest.NewRequest("POST", "/apply-source", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to resolve secret")
	assert.Nil(t, p.appliedInput, "plugin ApplySource should NOT have been called on resolution failure")
}

// secretRefMockConfigStore captures the last saved config for assertions.
type secretRefMockConfigStore struct {
	lastSavedConfig *CatalogSourcesConfig
}

func (m *secretRefMockConfigStore) Load(_ context.Context) (*CatalogSourcesConfig, string, error) {
	return &CatalogSourcesConfig{Catalogs: map[string]CatalogSection{}}, "v1", nil
}

func (m *secretRefMockConfigStore) Save(_ context.Context, cfg *CatalogSourcesConfig, _ string) (string, error) {
	m.lastSavedConfig = cfg
	return "v2", nil
}

func (m *secretRefMockConfigStore) Watch(_ context.Context) (<-chan ConfigChangeEvent, error) {
	return nil, nil
}

func (m *secretRefMockConfigStore) ListRevisions(_ context.Context) ([]ConfigRevision, error) {
	return []ConfigRevision{}, nil
}

func (m *secretRefMockConfigStore) Rollback(_ context.Context, _ string) (*CatalogSourcesConfig, string, error) {
	return nil, "", ErrRevisionNotFound
}

var _ ConfigStore = (*secretRefMockConfigStore)(nil)

// --- Tests for action endpoints in management router ---

// actionMgmtPlugin embeds testMgmtPlugin and adds ActionProvider support.
type actionMgmtPlugin struct {
	testMgmtPlugin
	actions      map[ActionScope][]ActionDefinition
	handleResult *ActionResult
	handleErr    error
}

func (p *actionMgmtPlugin) ListActions(scope ActionScope) []ActionDefinition {
	if p.actions == nil {
		return nil
	}
	return p.actions[scope]
}

func (p *actionMgmtPlugin) HandleAction(_ context.Context, scope ActionScope, targetID string, req ActionRequest) (*ActionResult, error) {
	if p.handleErr != nil {
		return nil, p.handleErr
	}
	if p.handleResult != nil {
		return p.handleResult, nil
	}
	return &ActionResult{
		Action:  req.Action,
		Status:  "completed",
		Message: fmt.Sprintf("executed %s on %s %s", req.Action, scope, targetID),
	}, nil
}

var _ ActionProvider = (*actionMgmtPlugin)(nil)

func TestManagementRouter_ActionRoutes(t *testing.T) {
	noopExtractor := func(_ *http.Request) Role { return RoleOperator }

	t.Run("source action dispatches through management router", func(t *testing.T) {
		p := &actionMgmtPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {
					{ID: "refresh", DisplayName: "Refresh", SupportsDryRun: true},
				},
			},
			handleResult: &ActionResult{
				Action:  "refresh",
				Status:  "completed",
				Message: "refreshed",
			},
		}

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, noopExtractor, nil))

		body := `{"action": "refresh"}`
		req := httptest.NewRequest("POST", "/sources/my-src:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result ActionResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, "refresh", result.Action)
	})

	t.Run("asset action dispatches through management router", func(t *testing.T) {
		p := &actionMgmtPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeAsset: {
					{ID: "tag", DisplayName: "Tag", SupportsDryRun: true, Idempotent: true},
				},
			},
			handleResult: &ActionResult{
				Action:  "tag",
				Status:  "completed",
				Message: "tagged",
			},
		}

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, noopExtractor, nil))

		body := `{"action": "tag", "params": {"tags": ["prod"]}}`
		req := httptest.NewRequest("POST", "/entities/my-model:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result ActionResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, "tag", result.Action)
	})

	t.Run("source actions list endpoint returns actions", func(t *testing.T) {
		p := &actionMgmtPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {
					{ID: "refresh", DisplayName: "Refresh"},
					{ID: "purge", DisplayName: "Purge", Destructive: true},
				},
			},
		}

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, noopExtractor, nil))

		req := httptest.NewRequest("GET", "/actions/source", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(2), result["count"])
	})

	t.Run("asset actions list endpoint returns actions", func(t *testing.T) {
		p := &actionMgmtPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeAsset: {
					{ID: "tag", DisplayName: "Tag"},
					{ID: "annotate", DisplayName: "Annotate"},
					{ID: "deprecate", DisplayName: "Deprecate"},
				},
			},
		}

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, noopExtractor, nil))

		req := httptest.NewRequest("GET", "/actions/asset", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(3), result["count"])
	})

	t.Run("plugin without ActionProvider has no action routes", func(t *testing.T) {
		p := &testMgmtPlugin{} // does not implement ActionProvider

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, noopExtractor, nil))

		// Verify that action discovery returns 405 (method not allowed) since the
		// route is not registered at all.
		req := httptest.NewRequest("GET", "/actions/source", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("RBAC blocks viewer from executing action", func(t *testing.T) {
		viewerExtractor := func(_ *http.Request) Role { return RoleViewer }

		p := &actionMgmtPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {
					{ID: "refresh", DisplayName: "Refresh"},
				},
			},
		}

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, viewerExtractor, nil))

		body := `{"action": "refresh"}`
		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("RBAC allows viewer to list actions", func(t *testing.T) {
		viewerExtractor := func(_ *http.Request) Role { return RoleViewer }

		p := &actionMgmtPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {
					{ID: "refresh", DisplayName: "Refresh"},
				},
			},
		}

		r := chi.NewRouter()
		r.Mount("/", managementRouter(p, viewerExtractor, nil))

		req := httptest.NewRequest("GET", "/actions/source", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// Actions list is read-only, should be accessible to viewers.
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
