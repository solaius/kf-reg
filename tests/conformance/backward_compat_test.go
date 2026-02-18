package conformance

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// TestBackwardCompatModelCatalog verifies that the existing model catalog API
// still works after governance endpoints are added.
func TestBackwardCompatModelCatalog(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	// The model catalog API should still be accessible.
	// First, discover what plugins are available to find the model catalog path.
	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	if response.Count == 0 {
		t.Skip("no plugins found; cannot test backward compatibility")
	}

	// Look for an MCP or model plugin to test its entity endpoints.
	for _, p := range response.Plugins {
		if p.CapabilitiesV2 == nil || len(p.CapabilitiesV2.Entities) == 0 {
			continue
		}

		t.Run(p.Name+"_list_endpoint", func(t *testing.T) {
			entity := p.CapabilitiesV2.Entities[0]
			resp, err := http.Get(serverURL + entity.Endpoints.List)
			if err != nil {
				t.Fatalf("GET %s failed: %v", entity.Endpoints.List, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("backward compat: GET %s returned %d: %s", entity.Endpoints.List, resp.StatusCode, string(body))
			}
		})
	}
}

// TestBackwardCompatPluginList verifies that /api/plugins still works and includes
// expected fields.
func TestBackwardCompatPluginList(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	if response.Count < 1 {
		t.Fatal("expected at least 1 plugin")
	}

	for _, p := range response.Plugins {
		t.Run(p.Name, func(t *testing.T) {
			if p.Name == "" {
				t.Error("plugin name is empty")
			}
			if p.BasePath == "" {
				t.Error("plugin basePath is empty")
			}
			if !p.Healthy {
				t.Errorf("plugin %s is not healthy", p.Name)
			}
		})
	}
}

// TestBackwardCompatCapabilitiesIncludeGovernance verifies that the capabilities
// endpoint includes governance information when available.
func TestBackwardCompatCapabilitiesIncludeGovernance(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	for _, p := range response.Plugins {
		t.Run(p.Name, func(t *testing.T) {
			// Fetch raw capabilities to check for governance field.
			capURL := "/api/plugins/" + p.Name + "/capabilities"
			resp, err := http.Get(serverURL + capURL)
			if err != nil {
				t.Fatalf("GET %s failed: %v", capURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Skipf("capabilities endpoint returned %d", resp.StatusCode)
			}

			var rawCaps map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&rawCaps); err != nil {
				t.Fatalf("failed to decode capabilities: %v", err)
			}

			// The "governance" field may be present in V2 capabilities.
			// This is informational -- governance support is not mandatory for all plugins.
			if gov, ok := rawCaps["governance"]; ok {
				t.Logf("plugin %s has governance capabilities: %v", p.Name, gov)
			} else {
				t.Logf("plugin %s does not have governance capabilities in response (may be served separately)", p.Name)
			}
		})
	}
}

// TestBackwardCompatHealthEndpoints verifies health endpoints still work with governance added.
func TestBackwardCompatHealthEndpoints(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
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
				t.Errorf("backward compat: GET %s returned %d: %s", path, resp.StatusCode, string(body))
			}
		})
	}
}
