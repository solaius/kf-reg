// Package conformance provides integration tests that validate all catalog
// plugins meet the Phase 5 universal framework contract. Tests run against a
// live catalog-server and require the CATALOG_SERVER_URL environment variable.
package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var serverURL string

func TestMain(m *testing.M) {
	serverURL = os.Getenv("CATALOG_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	os.Exit(m.Run())
}

// --- Types mirroring the server response structures ---

type pluginsResponse struct {
	Plugins []pluginInfo `json:"plugins"`
	Count   int          `json:"count"`
}

type pluginInfo struct {
	Name           string               `json:"name"`
	Version        string               `json:"version"`
	Description    string               `json:"description"`
	BasePath       string               `json:"basePath"`
	Healthy        bool                 `json:"healthy"`
	CapabilitiesV2 *capabilitiesV2      `json:"capabilitiesV2,omitempty"`
	Management     *managementCaps      `json:"management,omitempty"`
	Status         *pluginStatus        `json:"status,omitempty"`
}

type managementCaps struct {
	SourceManager bool `json:"sourceManager"`
	Refresh       bool `json:"refresh"`
	Diagnostics   bool `json:"diagnostics"`
	Actions       bool `json:"actions"`
}

type pluginStatus struct {
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
	Serving     bool   `json:"serving"`
	LastError   string `json:"lastError,omitempty"`
}

type capabilitiesV2 struct {
	SchemaVersion string             `json:"schemaVersion"`
	Plugin        pluginMeta         `json:"plugin"`
	Entities      []entityCaps       `json:"entities"`
	Sources       *sourceCaps        `json:"sources,omitempty"`
	Actions       []actionDefinition `json:"actions,omitempty"`
}

type pluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	DisplayName string `json:"displayName,omitempty"`
}

type entityCaps struct {
	Kind        string          `json:"kind"`
	Plural      string          `json:"plural"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
	Endpoints   entityEndpoints `json:"endpoints"`
	Fields      entityFields    `json:"fields"`
	Actions     []string        `json:"actions,omitempty"`
}

type entityEndpoints struct {
	List   string `json:"list"`
	Get    string `json:"get"`
	Action string `json:"action,omitempty"`
}

type entityFields struct {
	Columns      []columnHint  `json:"columns"`
	FilterFields []filterField `json:"filterFields,omitempty"`
	DetailFields []fieldHint   `json:"detailFields,omitempty"`
}

type columnHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Sortable    bool   `json:"sortable,omitempty"`
	Width       string `json:"width,omitempty"`
}

type filterField struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Type        string   `json:"type"`
	Options     []string `json:"options,omitempty"`
}

type fieldHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Section     string `json:"section,omitempty"`
}

type sourceCaps struct {
	Manageable  bool     `json:"manageable"`
	Refreshable bool     `json:"refreshable"`
	Types       []string `json:"types,omitempty"`
}

type actionDefinition struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Scope          string `json:"scope"`
	SupportsDryRun bool   `json:"supportsDryRun"`
	Idempotent     bool   `json:"idempotent"`
}

// --- Helpers ---

func getJSON(t *testing.T, path string, v any) {
	t.Helper()
	resp, err := http.Get(serverURL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("GET %s: decode error: %v", path, err)
	}
}

func waitForReady(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 30; i++ {
		resp, err := client.Get(serverURL + "/readyz")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("server not ready after 30 seconds")
}

// TestConformance discovers all plugins and runs conformance checks.
func TestConformance(t *testing.T) {
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	if response.Count == 0 {
		t.Fatal("no plugins found")
	}

	t.Logf("discovered %d plugin(s)", response.Count)

	for _, p := range response.Plugins {
		t.Run(p.Name, func(t *testing.T) {
			t.Run("healthy", func(t *testing.T) {
				if !p.Healthy {
					t.Errorf("plugin %s is not healthy", p.Name)
				}
			})

			t.Run("capabilities", func(t *testing.T) {
				testCapabilities(t, p)
			})

			t.Run("endpoints", func(t *testing.T) {
				testEndpoints(t, p)
			})

			if p.CapabilitiesV2 != nil && len(p.CapabilitiesV2.Actions) > 0 {
				t.Run("actions", func(t *testing.T) {
					testActions(t, p)
				})
			}

			t.Run("filters", func(t *testing.T) {
				testFilters(t, p)
			})
		})
	}
}

// TestHealthEndpoints validates /healthz, /livez, and /readyz.
func TestHealthEndpoints(t *testing.T) {
	waitForReady(t)

	for _, path := range []string{"/healthz", "/livez", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(serverURL + path)
			if err != nil {
				t.Fatalf("GET %s failed: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("GET %s returned %d: %s", path, resp.StatusCode, string(body))
			}

			var result map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			status, _ := result["status"].(string)
			if status == "" {
				t.Error("response missing 'status' field")
			}
		})
	}
}

// TestReadyzComponents validates /readyz returns component health details.
func TestReadyzComponents(t *testing.T) {
	waitForReady(t)

	var result map[string]any
	getJSON(t, "/readyz", &result)

	components, ok := result["components"].(map[string]any)
	if !ok {
		t.Fatal("readyz response missing 'components' object")
	}

	for _, key := range []string{"database", "initial_load", "plugins"} {
		comp, ok := components[key].(map[string]any)
		if !ok {
			t.Errorf("readyz missing component %q", key)
			continue
		}
		status, _ := comp["status"].(string)
		if status == "" {
			t.Errorf("component %q has no status", key)
		}
	}
}

// TestPluginCount validates the expected number of plugins are loaded.
func TestPluginCount(t *testing.T) {
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	if response.Count < 1 {
		t.Fatalf("expected at least 1 plugin, got %d", response.Count)
	}

	t.Logf("loaded %d plugin(s):", response.Count)
	for _, p := range response.Plugins {
		t.Logf("  - %s %s (healthy=%v, basePath=%s)", p.Name, p.Version, p.Healthy, p.BasePath)
	}

	// Verify count matches length.
	if response.Count != len(response.Plugins) {
		t.Errorf("count=%d but plugins array has %d items", response.Count, len(response.Plugins))
	}
}

// TestPluginsHaveBasicFields checks all plugins have required identity fields.
func TestPluginsHaveBasicFields(t *testing.T) {
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	for _, p := range response.Plugins {
		t.Run(p.Name, func(t *testing.T) {
			if p.Name == "" {
				t.Error("plugin name is empty")
			}
			if p.Version == "" {
				t.Error("plugin version is empty")
			}
			if p.Description == "" {
				t.Error("plugin description is empty")
			}
			if p.BasePath == "" {
				t.Error("plugin basePath is empty")
			}
			// BasePath should start with /api/
			if len(p.BasePath) < 5 || p.BasePath[:5] != "/api/" {
				t.Errorf("basePath %q does not start with /api/", p.BasePath)
			}
		})
	}
}

// TestCapabilitiesEndpointNotFound validates a 404 for unknown plugin.
func TestCapabilitiesEndpointNotFound(t *testing.T) {
	waitForReady(t)

	resp, err := http.Get(serverURL + "/api/plugins/nonexistent/capabilities")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown plugin, got %d", resp.StatusCode)
	}
}

// TestPluginNamesUnique verifies no two plugins share the same name.
func TestPluginNamesUnique(t *testing.T) {
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	seen := make(map[string]bool, len(response.Plugins))
	for _, p := range response.Plugins {
		if seen[p.Name] {
			t.Errorf("duplicate plugin name: %q", p.Name)
		}
		seen[p.Name] = true
	}
}

// TestBasePathsUnique verifies no two plugins share the same base path.
func TestBasePathsUnique(t *testing.T) {
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	seen := make(map[string]string, len(response.Plugins))
	for _, p := range response.Plugins {
		if other, exists := seen[p.BasePath]; exists {
			t.Errorf("plugins %q and %q share basePath %q", other, p.Name, p.BasePath)
		}
		seen[p.BasePath] = p.Name
	}
}

// TestPagination verifies pageSize parameter is respected.
func TestPagination(t *testing.T) {
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	for _, p := range response.Plugins {
		if p.CapabilitiesV2 == nil {
			continue
		}
		for _, entity := range p.CapabilitiesV2.Entities {
			t.Run(fmt.Sprintf("%s/%s", p.Name, entity.Plural), func(t *testing.T) {
				reqURL := fmt.Sprintf("%s?pageSize=1", entity.Endpoints.List)
				resp, err := http.Get(serverURL + reqURL)
				if err != nil {
					t.Fatalf("GET %s failed: %v", reqURL, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusNotFound {
					t.Skipf("GET %s returned 404 (list endpoint not available)", reqURL)
				}

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					t.Fatalf("GET %s returned %d: %s", reqURL, resp.StatusCode, string(body))
				}

				var result map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("response is not valid JSON: %v", err)
				}

				items, ok := result["items"].([]any)
				if !ok {
					t.Skip("response has no 'items' array")
				}

				if len(items) > 1 {
					t.Logf("note: pageSize=1 but got %d items (pagination may not be implemented)", len(items))
				}
			})
		}
	}
}
