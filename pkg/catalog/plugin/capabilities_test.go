package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v2Plugin implements CapabilitiesV2Provider directly.
type v2Plugin struct {
	testPlugin
	v2caps PluginCapabilitiesV2
}

func (p *v2Plugin) GetCapabilitiesV2() PluginCapabilitiesV2 {
	return p.v2caps
}

// v1OnlyPlugin implements only CapabilitiesProvider (V1).
type v1OnlyPlugin struct {
	testPlugin
	caps PluginCapabilities
}

func (p *v1OnlyPlugin) Capabilities() PluginCapabilities {
	return p.caps
}

// v1WithSourceManager implements V1 caps + SourceManager + RefreshProvider.
type v1WithSourceManager struct {
	v1OnlyPlugin
}

func (p *v1WithSourceManager) ListSources(_ context.Context) ([]SourceInfo, error) {
	return nil, nil
}
func (p *v1WithSourceManager) ValidateSource(_ context.Context, _ SourceConfigInput) (*ValidationResult, error) {
	return nil, nil
}
func (p *v1WithSourceManager) ApplySource(_ context.Context, _ SourceConfigInput) error { return nil }
func (p *v1WithSourceManager) EnableSource(_ context.Context, _ string, _ bool) error  { return nil }
func (p *v1WithSourceManager) DeleteSource(_ context.Context, _ string) error          { return nil }
func (p *v1WithSourceManager) Refresh(_ context.Context, _ string) (*RefreshResult, error) {
	return nil, nil
}
func (p *v1WithSourceManager) RefreshAll(_ context.Context) (*RefreshResult, error) { return nil, nil }

func TestBuildCapabilitiesV2_FromV2Provider(t *testing.T) {
	expected := PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: PluginMeta{
			Name:        "test",
			Version:     "v1",
			Description: "Test V2",
		},
		Entities: []EntityCapabilities{
			{
				Kind:        "Widget",
				Plural:      "widgets",
				DisplayName: "Widget",
			},
		},
	}

	p := &v2Plugin{
		testPlugin: testPlugin{
			name:        "test",
			version:     "v1",
			description: "Test V2",
		},
		v2caps: expected,
	}

	result := BuildCapabilitiesV2(p, "/api/test_catalog/v1")

	assert.Equal(t, expected.SchemaVersion, result.SchemaVersion)
	assert.Equal(t, expected.Plugin.Name, result.Plugin.Name)
	assert.Equal(t, expected.Plugin.Version, result.Plugin.Version)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, "Widget", result.Entities[0].Kind)
	assert.Equal(t, "widgets", result.Entities[0].Plural)
}

func TestBuildCapabilitiesV2_FromV1Capabilities(t *testing.T) {
	p := &v1OnlyPlugin{
		testPlugin: testPlugin{
			name:        "models",
			version:     "v1alpha1",
			description: "Model catalog",
		},
		caps: PluginCapabilities{
			EntityKinds:  []string{"Model", "ModelVersion"},
			ListEntities: true,
			GetEntity:    true,
		},
	}

	basePath := "/api/models_catalog/v1alpha1"
	result := BuildCapabilitiesV2(p, basePath)

	assert.Equal(t, "v1", result.SchemaVersion)
	assert.Equal(t, "models", result.Plugin.Name)
	assert.Equal(t, "v1alpha1", result.Plugin.Version)
	require.Len(t, result.Entities, 2)

	// First entity: Model
	assert.Equal(t, "Model", result.Entities[0].Kind)
	assert.Equal(t, "models", result.Entities[0].Plural)
	assert.Equal(t, basePath+"/models", result.Entities[0].Endpoints.List)
	assert.Equal(t, basePath+"/models/{name}", result.Entities[0].Endpoints.Get)

	// Second entity: ModelVersion
	assert.Equal(t, "ModelVersion", result.Entities[1].Kind)
	assert.Equal(t, "modelversions", result.Entities[1].Plural)
	assert.Equal(t, basePath+"/modelversions", result.Entities[1].Endpoints.List)
	assert.Equal(t, basePath+"/modelversions/{name}", result.Entities[1].Endpoints.Get)

	// No source capabilities
	assert.Nil(t, result.Sources)
}

func TestBuildCapabilitiesV2_WithSourceManager(t *testing.T) {
	p := &v1WithSourceManager{
		v1OnlyPlugin: v1OnlyPlugin{
			testPlugin: testPlugin{
				name:        "mcp",
				version:     "v1alpha1",
				description: "MCP catalog",
			},
			caps: PluginCapabilities{
				EntityKinds:  []string{"McpServer"},
				ListEntities: true,
				GetEntity:    true,
				ListSources:  true,
			},
		},
	}

	result := BuildCapabilitiesV2(p, "/api/mcp_catalog/v1alpha1")

	require.NotNil(t, result.Sources)
	assert.True(t, result.Sources.Manageable)
	assert.True(t, result.Sources.Refreshable)
}

func TestBuildCapabilitiesV2_BarePlugin(t *testing.T) {
	p := &testPlugin{
		name:        "bare",
		version:     "v1",
		description: "Bare plugin",
	}

	result := BuildCapabilitiesV2(p, "/api/bare_catalog/v1")

	assert.Equal(t, "v1", result.SchemaVersion)
	assert.Equal(t, "bare", result.Plugin.Name)
	assert.Empty(t, result.Entities)
	assert.Nil(t, result.Sources)
	assert.Empty(t, result.Actions)
}

func TestBuildCapabilitiesV2_V1ListOnlyNoGet(t *testing.T) {
	p := &v1OnlyPlugin{
		testPlugin: testPlugin{
			name:    "readonly",
			version: "v1",
		},
		caps: PluginCapabilities{
			EntityKinds:  []string{"Thing"},
			ListEntities: true,
			GetEntity:    false,
		},
	}

	result := BuildCapabilitiesV2(p, "/api/readonly_catalog/v1")

	require.Len(t, result.Entities, 1)
	assert.Equal(t, "/api/readonly_catalog/v1/things", result.Entities[0].Endpoints.List)
	assert.Empty(t, result.Entities[0].Endpoints.Get)
}

func TestPluginCapabilitiesV2_JSONRoundTrip(t *testing.T) {
	caps := PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: PluginMeta{
			Name:        "mcp",
			Version:     "v1alpha1",
			Description: "MCP Servers catalog",
			DisplayName: "MCP Servers",
			Icon:        "server",
		},
		Entities: []EntityCapabilities{
			{
				Kind:        "McpServer",
				Plural:      "mcpservers",
				DisplayName: "MCP Server",
				Endpoints: EntityEndpoints{
					List: "/api/mcp_catalog/v1alpha1/mcpservers",
					Get:  "/api/mcp_catalog/v1alpha1/mcpservers/{name}",
				},
				Fields: EntityFields{
					Columns: []V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true},
					},
					FilterFields: []V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "LIKE"}},
					},
					DetailFields: []V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
					},
				},
				UIHints: &EntityUIHints{
					Icon:           "server",
					NameField:      "name",
					DetailSections: []string{"Overview"},
				},
				Actions: []string{"refresh"},
			},
		},
		Sources: &SourceCapabilities{
			Manageable:  true,
			Refreshable: true,
			Types:       []string{"yaml"},
		},
		Actions: []ActionDefinition{
			{
				ID:          "refresh",
				DisplayName: "Refresh",
				Description: "Refresh data",
				Scope:       "source",
				Idempotent:  true,
			},
		},
	}

	data, err := json.Marshal(caps)
	require.NoError(t, err)

	var decoded PluginCapabilitiesV2
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, caps.SchemaVersion, decoded.SchemaVersion)
	assert.Equal(t, caps.Plugin.Name, decoded.Plugin.Name)
	assert.Equal(t, caps.Plugin.DisplayName, decoded.Plugin.DisplayName)
	require.Len(t, decoded.Entities, 1)
	assert.Equal(t, "McpServer", decoded.Entities[0].Kind)
	assert.Equal(t, "mcpservers", decoded.Entities[0].Plural)
	require.Len(t, decoded.Entities[0].Fields.Columns, 1)
	assert.Equal(t, "name", decoded.Entities[0].Fields.Columns[0].Name)
	require.NotNil(t, decoded.Sources)
	assert.True(t, decoded.Sources.Manageable)
	require.Len(t, decoded.Actions, 1)
	assert.Equal(t, "refresh", decoded.Actions[0].ID)
}

func TestCapabilitiesEndpoint_ReturnsV2(t *testing.T) {
	Reset()

	expected := PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: PluginMeta{
			Name:        "mytest",
			Version:     "v1",
			Description: "Test plugin for caps",
		},
		Entities: []EntityCapabilities{
			{
				Kind:        "Widget",
				Plural:      "widgets",
				DisplayName: "Widget",
			},
		},
	}

	p := &v2Plugin{
		testPlugin: testPlugin{
			name:        "mytest",
			version:     "v1",
			description: "Test plugin for caps",
			healthy:     true,
		},
		v2caps: expected,
	}
	Register(p)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"mytest": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	req := httptest.NewRequest("GET", "/api/plugins/mytest/capabilities", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result PluginCapabilitiesV2
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "v1", result.SchemaVersion)
	assert.Equal(t, "mytest", result.Plugin.Name)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, "Widget", result.Entities[0].Kind)

	Reset()
}

func TestCapabilitiesEndpoint_NotFound(t *testing.T) {
	Reset()

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	req := httptest.NewRequest("GET", "/api/plugins/nonexistent/capabilities", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var result map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "nonexistent")

	Reset()
}

func TestPluginsEndpoint_IncludesV2Capabilities(t *testing.T) {
	Reset()

	v2caps := PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: PluginMeta{
			Name:    "caps_test",
			Version: "v1",
		},
		Entities: []EntityCapabilities{
			{Kind: "Item", Plural: "items", DisplayName: "Item"},
		},
	}

	p := &v2Plugin{
		testPlugin: testPlugin{
			name:        "caps_test",
			version:     "v1",
			description: "V2 caps test",
			healthy:     true,
		},
		v2caps: v2caps,
	}
	Register(p)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"caps_test": {},
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
			Name           string                `json:"name"`
			CapabilitiesV2 *PluginCapabilitiesV2 `json:"capabilitiesV2,omitempty"`
		} `json:"plugins"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	require.Len(t, response.Plugins, 1)
	require.NotNil(t, response.Plugins[0].CapabilitiesV2)
	assert.Equal(t, "v1", response.Plugins[0].CapabilitiesV2.SchemaVersion)
	require.Len(t, response.Plugins[0].CapabilitiesV2.Entities, 1)
	assert.Equal(t, "Item", response.Plugins[0].CapabilitiesV2.Entities[0].Kind)

	Reset()
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Model", "models"},
		{"McpServer", "mcpservers"},
		{"ModelVersion", "modelversions"},
		{"item", "items"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, pluralize(tt.input))
		})
	}
}

// Ensure the test uses chi for URL params in capabilities handler.
func TestCapabilitiesEndpoint_V1FallbackBuild(t *testing.T) {
	Reset()

	p := &v1OnlyPlugin{
		testPlugin: testPlugin{
			name:        "v1only",
			version:     "v1alpha1",
			description: "V1 only plugin",
			healthy:     true,
		},
		caps: PluginCapabilities{
			EntityKinds:  []string{"Resource"},
			ListEntities: true,
			GetEntity:    true,
		},
	}
	Register(p)

	cfg := &CatalogSourcesConfig{
		Catalogs: map[string]CatalogSection{
			"v1only": {},
		},
	}

	server := NewServer(cfg, []string{}, nil, nil)
	err := server.Init(context.Background())
	require.NoError(t, err)

	router := server.MountRoutes()

	// Request capabilities endpoint
	req := httptest.NewRequest("GET", "/api/plugins/v1only/capabilities", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result PluginCapabilitiesV2
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "v1", result.SchemaVersion)
	assert.Equal(t, "v1only", result.Plugin.Name)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, "Resource", result.Entities[0].Kind)
	assert.Equal(t, "resources", result.Entities[0].Plural)
	basePath := "/api/v1only_catalog/v1alpha1"
	assert.Equal(t, basePath+"/resources", result.Entities[0].Endpoints.List)
	assert.Equal(t, basePath+"/resources/{name}", result.Entities[0].Endpoints.Get)

	// No source capabilities on this plugin
	assert.Nil(t, result.Sources)

	Reset()
}

// Verify that the chi.URLParam is not needed as an unused import.
var _ chi.Router = chi.NewRouter()
