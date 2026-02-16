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
)

// mgmtTestPlugin is a mock plugin implementing management interfaces.
type mgmtTestPlugin struct {
	sources    []SourceInfo
	validateFn func(SourceConfigInput) (*ValidationResult, error)
}

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

	handler := sourcesListHandler(p)
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
	handler := refreshAllHandler(p)

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
	r.Post("/refresh/{sourceId}", refreshSourceHandler(p))

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
	r.Post("/sources/{sourceId}/enable", enableHandler(p))

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
	r.Delete("/sources/{sourceId}", deleteSourceHandler(p))

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
	mgmt := managementRouter(p, DefaultRoleExtractor)
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
