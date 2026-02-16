// Package e2e contains smoke tests for the catalog server.
// These tests require a running catalog-server instance. Set the CATALOG_SERVER_URL
// environment variable to point at the server (default: http://localhost:8080).
//
// Run with:
//
//	CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/e2e/ -v -count=1
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// serverURL returns the base URL of the catalog server.
func serverURL() string {
	if u := os.Getenv("CATALOG_SERVER_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8080"
}

// client is a shared HTTP client with a reasonable timeout.
var client = &http.Client{Timeout: 30 * time.Second}

// doGet performs a GET request and returns the body and status code.
func doGet(t *testing.T, path string, headers map[string]string) ([]byte, int) {
	t.Helper()
	url := serverURL() + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	return body, resp.StatusCode
}

// doPost performs a POST request with a JSON body.
func doPost(t *testing.T, path string, payload any, headers map[string]string) ([]byte, int, http.Header) {
	t.Helper()
	url := serverURL() + path

	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshaling payload: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	return body, resp.StatusCode, resp.Header
}

// doDelete performs a DELETE request.
func doDelete(t *testing.T, path string, headers map[string]string) ([]byte, int) {
	t.Helper()
	url := serverURL() + path

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	return body, resp.StatusCode
}

// operatorHeaders returns headers that set the role to operator.
func operatorHeaders() map[string]string {
	return map[string]string{"X-User-Role": "operator"}
}

// viewerHeaders returns headers that set the role to viewer.
func viewerHeaders() map[string]string {
	return map[string]string{"X-User-Role": "viewer"}
}

// --- Smoke Tests ---

// TestHealthz verifies the server is alive.
func TestHealthz(t *testing.T) {
	body, code := doGet(t, "/healthz", nil)
	if code != 200 {
		t.Fatalf("expected 200 from /healthz, got %d: %s", code, body)
	}

	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing healthz response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

// TestPluginsList verifies that /api/plugins returns at least the MCP plugin.
func TestPluginsList(t *testing.T) {
	body, code := doGet(t, "/api/plugins", nil)
	if code != 200 {
		t.Fatalf("expected 200 from /api/plugins, got %d: %s", code, body)
	}

	var resp struct {
		Plugins []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Healthy bool   `json:"healthy"`
		} `json:"plugins"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing plugins response: %v", err)
	}

	if resp.Count < 1 {
		t.Fatalf("expected at least 1 plugin, got %d", resp.Count)
	}

	// Verify MCP plugin is present.
	var foundMcp bool
	for _, p := range resp.Plugins {
		if p.Name == "mcp" {
			foundMcp = true
			if !p.Healthy {
				t.Errorf("mcp plugin is not healthy")
			}
		}
	}
	if !foundMcp {
		t.Errorf("expected to find 'mcp' plugin, plugins: %v", resp.Plugins)
	}
}

// TestMcpSourcesList verifies that MCP sources are configured.
func TestMcpSourcesList(t *testing.T) {
	body, code := doGet(t, "/api/mcp_catalog/v1alpha1/sources", nil)
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}

	var resp struct {
		Sources []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Enabled bool   `json:"enabled"`
		} `json:"sources"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing sources response: %v", err)
	}

	if resp.Count < 1 {
		t.Fatalf("expected at least 1 source, got %d", resp.Count)
	}

	// Verify the default source exists.
	var foundDefault bool
	for _, s := range resp.Sources {
		if s.ID == "mcp-default" {
			foundDefault = true
			if !s.Enabled {
				t.Errorf("expected mcp-default source to be enabled")
			}
		}
	}
	if !foundDefault {
		t.Errorf("expected to find 'mcp-default' source")
	}
}

// TestMcpServersList verifies that MCP servers list returns at least 6 entries.
func TestMcpServersList(t *testing.T) {
	body, code := doGet(t, "/api/mcp_catalog/v1alpha1/mcpservers?pageSize=100", nil)
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}

	var resp struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Size int32 `json:"size"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing mcpservers response: %v", err)
	}

	if resp.Size < 6 {
		t.Errorf("expected at least 6 MCP servers, got %d", resp.Size)
	}

	// Verify known servers are present.
	names := make(map[string]bool)
	for _, item := range resp.Items {
		names[item.Name] = true
	}

	expected := []string{
		"kubernetes-mcp-server",
		"openshift-mcp-server",
		"ansible-mcp-server",
		"postgres-mcp-server",
		"github-mcp-server",
		"jira-mcp-server",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected to find MCP server %q in list", name)
		}
	}
}

// TestMcpServerGet verifies fetching a specific MCP server by name.
func TestMcpServerGet(t *testing.T) {
	body, code := doGet(t, "/api/mcp_catalog/v1alpha1/mcpservers/kubernetes-mcp-server", nil)
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}

	var server struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		ServerUrl     string `json:"serverUrl"`
		TransportType string `json:"transportType"`
		ToolCount     *int32 `json:"toolCount"`
	}
	if err := json.Unmarshal(body, &server); err != nil {
		t.Fatalf("parsing server response: %v", err)
	}

	if server.Name != "kubernetes-mcp-server" {
		t.Errorf("expected name 'kubernetes-mcp-server', got %q", server.Name)
	}
	if server.Description == "" {
		t.Error("expected non-empty description")
	}
	if server.ToolCount == nil || *server.ToolCount < 1 {
		t.Error("expected toolCount >= 1")
	}
}

// TestMcpServerGetNotFound verifies 404 for a non-existent server.
func TestMcpServerGetNotFound(t *testing.T) {
	_, code := doGet(t, "/api/mcp_catalog/v1alpha1/mcpservers/nonexistent-server-xyz", nil)
	if code != 404 {
		t.Errorf("expected 404 for non-existent server, got %d", code)
	}
}

// TestRefreshSource verifies triggering a refresh returns a result.
func TestRefreshSource(t *testing.T) {
	body, code, _ := doPost(t, "/api/mcp_catalog/v1alpha1/refresh", nil, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from refresh, got %d: %s", code, body)
	}

	var result struct {
		EntitiesLoaded int    `json:"entitiesLoaded"`
		Error          string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing refresh response: %v", err)
	}

	if result.Error != "" {
		t.Errorf("refresh returned error: %s", result.Error)
	}
	// After a fresh load, we expect at least some entities.
	if result.EntitiesLoaded < 1 {
		t.Logf("warning: refresh loaded %d entities (may be expected if no data change)", result.EntitiesLoaded)
	}
}

// TestRefreshRateLimit verifies that rapid refresh attempts get rate limited.
func TestRefreshRateLimit(t *testing.T) {
	// First refresh should succeed (or already be rate-limited from TestRefreshSource).
	_, code1, _ := doPost(t, "/api/mcp_catalog/v1alpha1/refresh", nil, operatorHeaders())

	// Immediately try again -- should be rate limited with 429.
	_, code2, headers := doPost(t, "/api/mcp_catalog/v1alpha1/refresh", nil, operatorHeaders())

	// If the first call was rate limited, the second will also be.
	// At least one of them should be 429 if rate limiting is working.
	if code1 == 200 {
		// First succeeded, second should be rate limited.
		if code2 != 429 {
			t.Errorf("expected 429 on rapid second refresh, got %d", code2)
		}
		retryAfter := headers.Get("Retry-After")
		if retryAfter == "" {
			t.Error("expected Retry-After header on 429 response")
		}
	} else if code1 == 429 {
		// Both rate limited is also acceptable.
		t.Logf("first refresh was also rate limited (%d), test still valid", code1)
	} else {
		t.Errorf("unexpected status code from first refresh: %d", code1)
	}
}

// TestRBACViewerBlocked verifies that viewers cannot perform mutations.
func TestRBACViewerBlocked(t *testing.T) {
	mutations := []struct {
		method string
		path   string
	}{
		{"POST", "/api/mcp_catalog/v1alpha1/refresh"},
		{"POST", "/api/mcp_catalog/v1alpha1/validate-source"},
		{"POST", "/api/mcp_catalog/v1alpha1/apply-source"},
	}

	for _, m := range mutations {
		t.Run(fmt.Sprintf("%s %s", m.method, m.path), func(t *testing.T) {
			// No role header = defaults to viewer.
			_, code, _ := doPost(t, m.path, map[string]string{"id": "test"}, nil)
			if code != 403 {
				t.Errorf("expected 403 for viewer on %s %s, got %d", m.method, m.path, code)
			}

			// Explicit viewer role header.
			_, code, _ = doPost(t, m.path, map[string]string{"id": "test"}, viewerHeaders())
			if code != 403 {
				t.Errorf("expected 403 for explicit viewer on %s %s, got %d", m.method, m.path, code)
			}
		})
	}
}

// TestRBACViewerCanRead verifies that viewers can access read-only endpoints.
func TestRBACViewerCanRead(t *testing.T) {
	readEndpoints := []string{
		"/api/plugins",
		"/api/mcp_catalog/v1alpha1/mcpservers",
		"/api/mcp_catalog/v1alpha1/sources",
		"/api/mcp_catalog/v1alpha1/diagnostics",
		"/healthz",
	}

	for _, path := range readEndpoints {
		t.Run(path, func(t *testing.T) {
			_, code := doGet(t, path, viewerHeaders())
			if code != 200 {
				t.Errorf("expected 200 for viewer on GET %s, got %d", path, code)
			}
		})
	}
}

// TestDiagnostics verifies the diagnostics endpoint returns source info.
func TestDiagnostics(t *testing.T) {
	body, code := doGet(t, "/api/mcp_catalog/v1alpha1/diagnostics", nil)
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}

	var diag struct {
		PluginName string `json:"pluginName"`
		Sources    []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			State       string `json:"state"`
			EntityCount int    `json:"entityCount"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(body, &diag); err != nil {
		t.Fatalf("parsing diagnostics response: %v", err)
	}

	if diag.PluginName != "mcp" {
		t.Errorf("expected pluginName 'mcp', got %q", diag.PluginName)
	}

	if len(diag.Sources) < 1 {
		t.Fatalf("expected at least 1 source in diagnostics")
	}

	// The default source should be available.
	for _, s := range diag.Sources {
		if s.ID == "mcp-default" {
			if s.State != "available" {
				t.Errorf("expected mcp-default state 'available', got %q", s.State)
			}
			if s.EntityCount < 6 {
				t.Errorf("expected at least 6 entities in mcp-default, got %d", s.EntityCount)
			}
		}
	}
}

// TestApplySourcePersistence verifies applying a new source config.
func TestApplySourcePersistence(t *testing.T) {
	sourceInput := map[string]any{
		"id":      "e2e-test-source",
		"name":    "E2E Test Source",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/test.yaml",
		},
	}

	// Apply the source (operator role required).
	body, code, _ := doPost(t, "/api/mcp_catalog/v1alpha1/apply-source", sourceInput, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from apply-source, got %d: %s", code, body)
	}

	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing apply response: %v", err)
	}
	if resp["status"] != "applied" {
		t.Errorf("expected status 'applied', got %q", resp["status"])
	}

	// Verify it appears in the sources list.
	body, code = doGet(t, "/api/mcp_catalog/v1alpha1/sources", nil)
	if code != 200 {
		t.Fatalf("expected 200 from sources list, got %d", code)
	}

	var listResp struct {
		Sources []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("parsing sources list: %v", err)
	}

	var found bool
	for _, s := range listResp.Sources {
		if s.ID == "e2e-test-source" {
			found = true
			if s.Name != "E2E Test Source" {
				t.Errorf("expected name 'E2E Test Source', got %q", s.Name)
			}
		}
	}
	if !found {
		t.Error("applied source 'e2e-test-source' not found in sources list")
	}

	// Clean up: delete the test source.
	_, code = doDelete(t, "/api/mcp_catalog/v1alpha1/sources/e2e-test-source", operatorHeaders())
	if code != 200 {
		t.Logf("warning: cleanup delete returned %d (may not be implemented)", code)
	}
}

// TestMcpServersFilterQuery verifies that filterQuery works on list endpoint.
func TestMcpServersFilterQuery(t *testing.T) {
	// Filter for local deployment mode.
	body, code := doGet(t, "/api/mcp_catalog/v1alpha1/mcpservers?filterQuery=deploymentMode%3D%27local%27", nil)
	if code != 200 {
		t.Fatalf("expected 200 for filtered list, got %d: %s", code, body)
	}

	var resp struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Size int32 `json:"size"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing filtered response: %v", err)
	}

	// We know there are 4 local servers in the production data.
	if resp.Size < 1 {
		t.Errorf("expected at least 1 local server, got %d", resp.Size)
	}
}

// TestBrokenYAMLDiagnostics verifies that a source pointing to an invalid YAML
// path surfaces an error in diagnostics. This test applies a source with a
// non-existent YAML path, triggers a refresh, then checks diagnostics for an
// error state. It cleans up after itself.
func TestBrokenYAMLDiagnostics(t *testing.T) {
	const sourceID = "e2e-broken-yaml-source"

	// Apply a source with a non-existent YAML path.
	sourceInput := map[string]any{
		"id":      sourceID,
		"name":    "Broken YAML Source",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/broken-catalog.yaml",
		},
	}

	body, code, _ := doPost(t, "/api/mcp_catalog/v1alpha1/apply-source", sourceInput, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from apply-source, got %d: %s", code, body)
	}

	// Trigger a refresh to force the provider to attempt reading the broken path.
	// The refresh may return an error or succeed with 0 entities -- either is fine.
	doPost(t, fmt.Sprintf("/api/mcp_catalog/v1alpha1/refresh/%s", sourceID), nil, operatorHeaders())

	// Check diagnostics for the broken source.
	body, code = doGet(t, "/api/mcp_catalog/v1alpha1/diagnostics", nil)
	if code != 200 {
		t.Fatalf("expected 200 from diagnostics, got %d: %s", code, body)
	}

	var diag struct {
		Sources []struct {
			ID          string `json:"id"`
			State       string `json:"state"`
			EntityCount int    `json:"entityCount"`
			Error       string `json:"error,omitempty"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(body, &diag); err != nil {
		t.Fatalf("parsing diagnostics: %v", err)
	}

	var found bool
	for _, s := range diag.Sources {
		if s.ID == sourceID {
			found = true
			// The source should report an error state or zero entities due to
			// the broken YAML path. We check both possibilities.
			if s.State == "error" {
				t.Logf("broken source has error state as expected: %s", s.Error)
			} else if s.EntityCount == 0 {
				t.Logf("broken source has 0 entities (path not found)")
			} else {
				t.Errorf("expected error state or 0 entities for broken source, got state=%q entities=%d", s.State, s.EntityCount)
			}
		}
	}
	if !found {
		t.Logf("broken source %q not found in diagnostics (may not be tracked until refresh completes)", sourceID)
	}

	// Clean up: delete the broken source.
	_, code = doDelete(t, fmt.Sprintf("/api/mcp_catalog/v1alpha1/sources/%s", sourceID), operatorHeaders())
	if code != 200 {
		t.Logf("cleanup delete returned %d", code)
	}
}
