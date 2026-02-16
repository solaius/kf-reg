package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPlugin is a CatalogPlugin implementation for testing.
type testPlugin struct {
	name        string
	version     string
	description string
	healthy     bool
	initCalled  bool
	startCalled bool
	stopCalled  bool
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Version() string     { return p.version }
func (p *testPlugin) Description() string { return p.description }

func (p *testPlugin) Init(ctx context.Context, cfg Config) error {
	p.initCalled = true
	return nil
}

func (p *testPlugin) Start(ctx context.Context) error {
	p.startCalled = true
	return nil
}

func (p *testPlugin) Stop(ctx context.Context) error {
	p.stopCalled = true
	return nil
}

func (p *testPlugin) Healthy() bool { return p.healthy }

func (p *testPlugin) RegisterRoutes(router chi.Router) error {
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return nil
}

func (p *testPlugin) Migrations() []Migration { return nil }

func TestServerInit(t *testing.T) {
	Reset()

	plugin := &testPlugin{
		name:    "test",
		version: "v1",
		healthy: true,
	}
	Register(plugin)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"test": {
				Sources: []SourceConfig{},
			},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	assert.True(t, plugin.initCalled)
	assert.Equal(t, 1, len(server.Plugins()))

	Reset()
}

func TestServerInitializesUnconfiguredPlugins(t *testing.T) {
	Reset()

	plugin := &testPlugin{
		name:    "test",
		version: "v1",
		healthy: true,
	}
	Register(plugin)

	// Empty config - no catalogs configured
	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	// Plugin should still be initialized even without config
	assert.True(t, plugin.initCalled)
	assert.Equal(t, 1, len(server.Plugins()))

	Reset()
}

func TestServerHealthEndpoint(t *testing.T) {
	Reset()

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	// Test /healthz
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "alive")

	Reset()
}

func TestServerHealthzLivezAlias(t *testing.T) {
	Reset()

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	// Test /livez
	req := httptest.NewRequest("GET", "/livez", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "alive")

	Reset()
}

func TestServerReadyEndpoint(t *testing.T) {
	Reset()

	healthyPlugin := &testPlugin{
		name:    "healthy",
		version: "v1",
		healthy: true,
	}
	Register(healthyPlugin)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"healthy": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	// Test /readyz when all plugins are healthy
	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")

	Reset()
}

func TestServerPluginsEndpoint(t *testing.T) {
	Reset()

	plugin := &testPlugin{
		name:        "test",
		version:     "v1",
		description: "Test plugin",
		healthy:     true,
	}
	Register(plugin)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"test": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	// Test /api/plugins
	req := httptest.NewRequest("GET", "/api/plugins", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "test")
	assert.Contains(t, body, "v1")

	Reset()
}

func TestServerStartStop(t *testing.T) {
	Reset()

	plugin := &testPlugin{
		name:    "test",
		version: "v1",
		healthy: true,
	}
	Register(plugin)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"test": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	// Start
	err = server.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, plugin.startCalled)

	// Stop
	err = server.Stop(context.Background())
	require.NoError(t, err)
	assert.True(t, plugin.stopCalled)

	Reset()
}

// failingPlugin is a testPlugin whose Init always returns an error.
type failingPlugin struct {
	testPlugin
	initErr error
}

func (p *failingPlugin) Init(ctx context.Context, cfg Config) error {
	p.initCalled = true
	return p.initErr
}

// capablePlugin is a testPlugin that implements CapabilitiesProvider.
type capablePlugin struct {
	testPlugin
	capabilities PluginCapabilities
}

func (p *capablePlugin) Capabilities() PluginCapabilities {
	return p.capabilities
}

func TestServerInitPluginFailureIsolation(t *testing.T) {
	Reset()

	failing := &failingPlugin{
		testPlugin: testPlugin{
			name:    "failing",
			version: "v1",
			healthy: false,
		},
		initErr: fmt.Errorf("connection refused"),
	}
	Register(failing)

	working := &testPlugin{
		name:    "working",
		version: "v1",
		healthy: true,
	}
	Register(working)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"failing": {},
			"working": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err, "server Init should not fail when a plugin fails")

	// Only the working plugin should be in the initialized list
	assert.Equal(t, 1, len(server.Plugins()))
	assert.Equal(t, "working", server.Plugins()[0].Name())
	assert.True(t, working.initCalled)
	assert.True(t, failing.initCalled)

	// Verify /api/plugins shows both plugins (one with error status)
	router := server.MountRoutes()
	req := httptest.NewRequest("GET", "/api/plugins", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Plugins []struct {
			Name    string        `json:"name"`
			Healthy bool          `json:"healthy"`
			Status  *PluginStatus `json:"status,omitempty"`
		} `json:"plugins"`
		Count int `json:"count"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count)

	// Find the failed plugin in the response
	var foundFailing bool
	for _, p := range response.Plugins {
		if p.Name == "failing" {
			foundFailing = true
			assert.False(t, p.Healthy)
			require.NotNil(t, p.Status)
			assert.False(t, p.Status.Initialized)
			assert.False(t, p.Status.Serving)
			assert.Contains(t, p.Status.LastError, "connection refused")
		}
	}
	assert.True(t, foundFailing, "failed plugin should appear in /api/plugins response")

	Reset()
}

func TestServerPluginsEndpointWithCapabilities(t *testing.T) {
	Reset()

	plugin := &capablePlugin{
		testPlugin: testPlugin{
			name:        "models",
			version:     "v1alpha1",
			description: "Model catalog",
			healthy:     true,
		},
		capabilities: PluginCapabilities{
			EntityKinds:  []string{"Model", "ModelVersion", "ModelArtifact"},
			ListEntities: true,
			GetEntity:    true,
			ListSources:  true,
			Artifacts:    true,
		},
	}
	Register(plugin)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"models": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()
	req := httptest.NewRequest("GET", "/api/plugins", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Plugins []struct {
			Name         string              `json:"name"`
			EntityKinds  []string            `json:"entityKinds,omitempty"`
			Capabilities *PluginCapabilities `json:"capabilities,omitempty"`
		} `json:"plugins"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	require.Equal(t, 1, len(response.Plugins))
	p := response.Plugins[0]
	assert.Equal(t, "models", p.Name)

	// Verify capabilities are included
	require.NotNil(t, p.Capabilities)
	assert.True(t, p.Capabilities.ListEntities)
	assert.True(t, p.Capabilities.GetEntity)
	assert.True(t, p.Capabilities.ListSources)
	assert.True(t, p.Capabilities.Artifacts)
	assert.Equal(t, []string{"Model", "ModelVersion", "ModelArtifact"}, p.Capabilities.EntityKinds)

	// Verify entityKinds is also set at top level
	assert.Equal(t, []string{"Model", "ModelVersion", "ModelArtifact"}, p.EntityKinds)

	Reset()
}

func TestServerReadyEndpointWithFailedPlugin(t *testing.T) {
	Reset()

	failing := &failingPlugin{
		testPlugin: testPlugin{
			name:    "broken",
			version: "v1",
			healthy: false,
		},
		initErr: fmt.Errorf("database unavailable"),
	}
	Register(failing)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"broken": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err, "server Init should succeed even with failed plugin")

	router := server.MountRoutes()
	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var response struct {
		Status     string                       `json:"status"`
		Components map[string]map[string]string `json:"components"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "not_ready", response.Status)
	assert.Contains(t, response.Components, "plugins")
	assert.Equal(t, "degraded", response.Components["plugins"]["status"])

	Reset()
}

func TestServerLivezReturnsAliveAndUptime(t *testing.T) {
	Reset()

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	req := httptest.NewRequest("GET", "/livez", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "alive", response["status"])
	assert.NotEmpty(t, response["uptime"])

	Reset()
}

func TestServerReadyEndpointComponentBreakdown(t *testing.T) {
	Reset()

	plugin := &testPlugin{
		name:    "test",
		version: "v1",
		healthy: true,
	}
	Register(plugin)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"test": {},
		},
	}

	// No DB configured
	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Status     string         `json:"status"`
		Components map[string]any `json:"components"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response.Status)

	// Verify component structure
	assert.Contains(t, response.Components, "database")
	assert.Contains(t, response.Components, "initial_load")
	assert.Contains(t, response.Components, "plugins")

	// DB should be not_configured when no DB is provided
	dbComp, ok := response.Components["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "not_configured", dbComp["status"])

	// Initial load should be complete after Init
	loadComp, ok := response.Components["initial_load"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "complete", loadComp["status"])

	// Plugins should be healthy
	pluginsComp, ok := response.Components["plugins"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "healthy", pluginsComp["status"])

	Reset()
}

func TestServerHealthzAndLivezAreSameHandler(t *testing.T) {
	Reset()

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	// Both endpoints should return the same structure
	for _, path := range []string{"/healthz", "/livez"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]string
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "alive", response["status"])
			assert.Contains(t, response, "uptime")
		})
	}

	Reset()
}

func TestServerReadyEndpointBeforeInit(t *testing.T) {
	Reset()

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	// Do NOT call Init - simulate pre-init state

	// Manually mount routes to test readyz before init completes
	router := server.MountRoutes()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should be not_ready because initialLoadDone is false
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var response struct {
		Status     string         `json:"status"`
		Components map[string]any `json:"components"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", response.Status)

	loadComp, ok := response.Components["initial_load"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "pending", loadComp["status"])

	Reset()
}
