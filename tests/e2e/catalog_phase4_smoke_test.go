// Package e2e contains Phase 4 smoke tests for the catalog server.
// These tests cover livez/readyz health endpoints, multi-layer validation,
// revision history, and refresh-after-apply behavior.
//
// Run with:
//
//	CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/e2e/ -v -run TestPhase4 -count=1
package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// catalogAvailable skips the test if the catalog server is not reachable.
func catalogAvailable(t *testing.T) {
	t.Helper()
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(serverURL() + "/livez")
	if err != nil {
		t.Skip("catalog server not available at " + serverURL())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Skip("catalog server not ready")
	}
}

// --- Phase 4: Health Endpoint Tests ---

// TestPhase4Livez verifies that GET /livez returns 200 with "alive" status and uptime.
func TestPhase4Livez(t *testing.T) {
	catalogAvailable(t)

	body, code := doGet(t, "/livez", nil)
	if code != 200 {
		t.Fatalf("expected 200 from /livez, got %d: %s", code, body)
	}

	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing livez response: %v", err)
	}

	if resp["status"] != "alive" {
		t.Errorf("expected status 'alive', got %q", resp["status"])
	}
	if resp["uptime"] == "" {
		t.Error("expected non-empty 'uptime' field in livez response")
	}
}

// TestPhase4Readyz verifies that GET /readyz returns 200 with component breakdown.
func TestPhase4Readyz(t *testing.T) {
	catalogAvailable(t)

	body, code := doGet(t, "/readyz", nil)
	if code != 200 {
		t.Fatalf("expected 200 from /readyz, got %d: %s", code, body)
	}

	var resp struct {
		Status     string `json:"status"`
		Components struct {
			Database    map[string]string `json:"database"`
			InitialLoad map[string]string `json:"initial_load"`
			Plugins     map[string]string `json:"plugins"`
		} `json:"components"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing readyz response: %v", err)
	}

	if resp.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", resp.Status)
	}

	// Verify component breakdown structure.
	if resp.Components.Database == nil {
		t.Error("expected 'database' component in readyz response")
	} else if _, ok := resp.Components.Database["status"]; !ok {
		t.Error("expected 'status' key in database component")
	}

	if resp.Components.InitialLoad == nil {
		t.Error("expected 'initial_load' component in readyz response")
	} else if resp.Components.InitialLoad["status"] != "complete" {
		t.Errorf("expected initial_load status 'complete', got %q", resp.Components.InitialLoad["status"])
	}

	if resp.Components.Plugins == nil {
		t.Error("expected 'plugins' component in readyz response")
	} else if _, ok := resp.Components.Plugins["status"]; !ok {
		t.Error("expected 'status' key in plugins component")
	}
}

// TestPhase4HealthzAlias verifies that GET /healthz returns the same response as /livez.
func TestPhase4HealthzAlias(t *testing.T) {
	catalogAvailable(t)

	body, code := doGet(t, "/healthz", nil)
	if code != 200 {
		t.Fatalf("expected 200 from /healthz, got %d: %s", code, body)
	}

	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing healthz response: %v", err)
	}

	// /healthz is aliased to the livez handler, so it should return "alive".
	if resp["status"] != "alive" {
		t.Errorf("expected status 'alive' from /healthz (livez alias), got %q", resp["status"])
	}
	if resp["uptime"] == "" {
		t.Error("expected non-empty 'uptime' field in /healthz response")
	}
}

// --- Phase 4: Detailed Validation Tests ---

const mcpBasePath = "/api/mcp_catalog/v1alpha1"

// TestPhase4ValidateEndpointValidYAML verifies that the detailed validate endpoint
// accepts valid YAML content and returns a valid result with layer breakdown.
func TestPhase4ValidateEndpointValidYAML(t *testing.T) {
	catalogAvailable(t)

	payload := map[string]any{
		"id":   "validate-test",
		"name": "Validate Test Source",
		"type": "yaml",
		"properties": map[string]any{
			"content": "servers:\n  - name: test-server\n    description: A test server\n",
		},
	}

	body, code, _ := doPost(t, mcpBasePath+"/sources/mcp-default:validate", payload, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from detailed validate, got %d: %s", code, body)
	}

	var result struct {
		Valid        bool `json:"valid"`
		Errors       []struct {
			Field   string `json:"field,omitempty"`
			Message string `json:"message"`
		} `json:"errors,omitempty"`
		LayerResults []struct {
			Layer string `json:"layer"`
			Valid bool   `json:"valid"`
		} `json:"layerResults,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing validate response: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid=true for valid YAML, got false; errors: %v", result.Errors)
	}

	// Should have layer results from the multi-layer validator.
	if len(result.LayerResults) == 0 {
		t.Error("expected layerResults in validation response")
	}

	// Verify the yaml_parse layer passed.
	for _, lr := range result.LayerResults {
		if lr.Layer == "yaml_parse" && !lr.Valid {
			t.Errorf("expected yaml_parse layer to pass for valid YAML")
		}
	}
}

// TestPhase4ValidateEndpointInvalidYAML verifies that the detailed validate endpoint
// detects invalid YAML content and returns a parse error.
func TestPhase4ValidateEndpointInvalidYAML(t *testing.T) {
	catalogAvailable(t)

	payload := map[string]any{
		"id":   "validate-bad-yaml",
		"name": "Bad YAML Source",
		"type": "yaml",
		"properties": map[string]any{
			"content": "this: is: not: valid:\n  yaml: [broken\n",
		},
	}

	body, code, _ := doPost(t, mcpBasePath+"/sources/mcp-default:validate", payload, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from detailed validate (validation returns 200 with errors), got %d: %s", code, body)
	}

	var result struct {
		Valid        bool `json:"valid"`
		Errors       []struct {
			Field   string `json:"field,omitempty"`
			Message string `json:"message"`
		} `json:"errors,omitempty"`
		LayerResults []struct {
			Layer  string `json:"layer"`
			Valid  bool   `json:"valid"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors,omitempty"`
		} `json:"layerResults,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing validate response: %v", err)
	}

	if result.Valid {
		t.Error("expected valid=false for invalid YAML, got true")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one validation error for invalid YAML")
	}

	// The yaml_parse layer should have failed.
	var foundYAMLLayer bool
	for _, lr := range result.LayerResults {
		if lr.Layer == "yaml_parse" {
			foundYAMLLayer = true
			if lr.Valid {
				t.Error("expected yaml_parse layer to fail for invalid YAML")
			}
		}
	}
	if !foundYAMLLayer {
		t.Error("expected yaml_parse layer in layerResults")
	}
}

// TestPhase4ValidateEndpointUnknownFields verifies that the detailed validate endpoint
// detects unknown fields via strict YAML decoding and reports them in the strict_fields layer.
func TestPhase4ValidateEndpointUnknownFields(t *testing.T) {
	catalogAvailable(t)

	// YAML that parses fine but has fields not in the strict schema.
	payload := map[string]any{
		"id":   "validate-unknown-fields",
		"name": "Unknown Fields Source",
		"type": "yaml",
		"properties": map[string]any{
			"content": "id: test\nname: test\nunknown_field_xyz: should_not_exist\n",
		},
	}

	body, code, _ := doPost(t, mcpBasePath+"/sources/mcp-default:validate", payload, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from detailed validate, got %d: %s", code, body)
	}

	var result struct {
		Valid        bool `json:"valid"`
		Errors       []struct {
			Field   string `json:"field,omitempty"`
			Message string `json:"message"`
		} `json:"errors,omitempty"`
		LayerResults []struct {
			Layer  string `json:"layer"`
			Valid  bool   `json:"valid"`
			Errors []struct {
				Field   string `json:"field,omitempty"`
				Message string `json:"message"`
			} `json:"errors,omitempty"`
		} `json:"layerResults,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing validate response: %v", err)
	}

	// The yaml_parse layer should pass (the YAML is syntactically valid).
	// The strict_fields layer should report unknown fields.
	var yamlParsePassed, strictFieldsFailed bool
	for _, lr := range result.LayerResults {
		switch lr.Layer {
		case "yaml_parse":
			if lr.Valid {
				yamlParsePassed = true
			}
		case "strict_fields":
			if !lr.Valid {
				strictFieldsFailed = true
			}
		}
	}

	if !yamlParsePassed {
		t.Error("expected yaml_parse layer to pass for syntactically valid YAML")
	}
	if !strictFieldsFailed {
		t.Error("expected strict_fields layer to fail for unknown fields")
	}
}

// --- Phase 4: Apply Validation Tests ---

// TestPhase4ApplyRejectsInvalid verifies that POST apply-source rejects invalid configs
// with 422 Unprocessable Entity and a validation result body.
func TestPhase4ApplyRejectsInvalid(t *testing.T) {
	catalogAvailable(t)

	// Missing required fields: no id, no name, no type.
	payload := map[string]any{
		"properties": map[string]any{},
	}

	body, code, _ := doPost(t, mcpBasePath+"/apply-source", payload, operatorHeaders())
	if code != 422 {
		t.Fatalf("expected 422 for invalid apply, got %d: %s", code, body)
	}

	var result struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Field   string `json:"field,omitempty"`
			Message string `json:"message"`
		} `json:"errors,omitempty"`
		LayerResults []struct {
			Layer string `json:"layer"`
			Valid bool   `json:"valid"`
		} `json:"layerResults,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing 422 response body: %v", err)
	}

	if result.Valid {
		t.Error("expected valid=false in 422 response")
	}
	if len(result.Errors) == 0 {
		t.Error("expected validation errors in 422 response")
	}

	// Check that required-field errors are present.
	fields := make(map[string]bool)
	for _, e := range result.Errors {
		fields[e.Field] = true
	}
	for _, required := range []string{"id", "name", "type"} {
		if !fields[required] {
			t.Errorf("expected validation error for required field %q", required)
		}
	}
}

// --- Phase 4: Revision History Tests ---

// TestPhase4ListRevisions verifies that GET revisions returns an array response.
func TestPhase4ListRevisions(t *testing.T) {
	catalogAvailable(t)

	body, code := doGet(t, mcpBasePath+"/sources/mcp-default/revisions", nil)
	if code != 200 {
		t.Fatalf("expected 200 from revisions endpoint, got %d: %s", code, body)
	}

	var resp struct {
		Revisions []struct {
			Version   string `json:"version"`
			Timestamp string `json:"timestamp,omitempty"`
		} `json:"revisions"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing revisions response: %v", err)
	}

	// The response should always have the revisions array (possibly empty).
	if resp.Revisions == nil {
		t.Error("expected 'revisions' array in response (even if empty)")
	}

	// Count should match the array length.
	if resp.Count != len(resp.Revisions) {
		t.Errorf("expected count=%d to match revisions length=%d", resp.Count, len(resp.Revisions))
	}
}

// --- Phase 4: Revision History + Rollback Tests ---

// TestPhase4ListRevisionsAfterApply verifies that applying a source twice
// produces at least 2 revisions in the revision history.
func TestPhase4ListRevisionsAfterApply(t *testing.T) {
	catalogAvailable(t)

	const sourceID = "e2e-phase4-rev-source"

	// Apply version 1.
	payload1 := map[string]any{
		"id":      sourceID,
		"name":    "Rev Test v1",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/rev-v1.yaml",
		},
	}
	body, code, _ := doPost(t, mcpBasePath+"/apply-source", payload1, operatorHeaders())
	if code != 200 {
		t.Fatalf("apply v1: expected 200, got %d: %s", code, body)
	}

	// Apply version 2 (different name).
	payload2 := map[string]any{
		"id":      sourceID,
		"name":    "Rev Test v2",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/rev-v2.yaml",
		},
	}
	body, code, _ = doPost(t, mcpBasePath+"/apply-source", payload2, operatorHeaders())
	if code != 200 {
		t.Fatalf("apply v2: expected 200, got %d: %s", code, body)
	}

	// List revisions -- expect at least 2.
	body, code = doGet(t, mcpBasePath+"/sources/"+sourceID+"/revisions", nil)
	if code != 200 {
		t.Fatalf("expected 200 from revisions, got %d: %s", code, body)
	}

	var resp struct {
		Revisions []struct {
			Version   string `json:"version"`
			Timestamp string `json:"timestamp,omitempty"`
			Size      int64  `json:"size,omitempty"`
		} `json:"revisions"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parsing revisions: %v", err)
	}

	if resp.Count < 2 {
		t.Errorf("expected at least 2 revisions after 2 applies, got %d", resp.Count)
	}

	// Each revision should have a non-empty version string.
	for i, rev := range resp.Revisions {
		if rev.Version == "" {
			t.Errorf("revision[%d] has empty version", i)
		}
	}

	// Clean up.
	_, delCode := doDelete(t, fmt.Sprintf("%s/sources/%s", mcpBasePath, sourceID), operatorHeaders())
	if delCode != 200 {
		t.Logf("cleanup delete returned %d", delCode)
	}
}

// TestPhase4Rollback verifies that rolling back to a previous revision restores
// the prior configuration. The flow is:
//  1. Apply source config v1
//  2. Record revision version
//  3. Apply source config v2 (different name)
//  4. Rollback to v1's version
//  5. Verify the source has the v1 name
func TestPhase4Rollback(t *testing.T) {
	catalogAvailable(t)

	const sourceID = "e2e-phase4-rollback"

	// Step 1: Apply v1.
	payload1 := map[string]any{
		"id":      sourceID,
		"name":    "Rollback Test v1",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/rollback-v1.yaml",
		},
	}
	body, code, _ := doPost(t, mcpBasePath+"/apply-source", payload1, operatorHeaders())
	if code != 200 {
		t.Fatalf("apply v1: expected 200, got %d: %s", code, body)
	}

	// Step 2: Get the current revision version after v1.
	body, code = doGet(t, mcpBasePath+"/sources/"+sourceID+"/revisions", nil)
	if code != 200 {
		t.Fatalf("revisions after v1: expected 200, got %d: %s", code, body)
	}

	var revResp struct {
		Revisions []struct {
			Version string `json:"version"`
		} `json:"revisions"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &revResp); err != nil {
		t.Fatalf("parsing revisions: %v", err)
	}

	if revResp.Count < 1 {
		t.Fatalf("expected at least 1 revision after apply, got %d", revResp.Count)
	}

	// Use the most recent revision as our rollback target after applying v2.
	v1Version := revResp.Revisions[revResp.Count-1].Version
	t.Logf("v1 revision version: %s", v1Version)

	// Step 3: Apply v2 with a different name.
	payload2 := map[string]any{
		"id":      sourceID,
		"name":    "Rollback Test v2",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/rollback-v2.yaml",
		},
	}
	body, code, _ = doPost(t, mcpBasePath+"/apply-source", payload2, operatorHeaders())
	if code != 200 {
		t.Fatalf("apply v2: expected 200, got %d: %s", code, body)
	}

	// Verify we are now on v2.
	body, code = doGet(t, mcpBasePath+"/sources", nil)
	if code != 200 {
		t.Fatalf("sources list: expected 200, got %d: %s", code, body)
	}

	type sourceEntry struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var listResp struct {
		Sources []sourceEntry `json:"sources"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("parsing sources: %v", err)
	}
	for _, s := range listResp.Sources {
		if s.ID == sourceID && s.Name != "Rollback Test v2" {
			t.Errorf("expected source name 'Rollback Test v2' before rollback, got %q", s.Name)
		}
	}

	// Step 4: Rollback to v1.
	rollbackPayload := map[string]any{
		"version": v1Version,
	}
	body, code, _ = doPost(t, mcpBasePath+"/sources/"+sourceID+":rollback", rollbackPayload, operatorHeaders())
	if code != 200 {
		t.Fatalf("rollback: expected 200, got %d: %s", code, body)
	}

	var rollbackResp struct {
		Status      string `json:"status"`
		Version     string `json:"version"`
		ReinitError string `json:"reinitError,omitempty"`
	}
	if err := json.Unmarshal(body, &rollbackResp); err != nil {
		t.Fatalf("parsing rollback response: %v", err)
	}

	if rollbackResp.Status != "rolled_back" {
		t.Errorf("expected status 'rolled_back', got %q", rollbackResp.Status)
	}
	if rollbackResp.Version == "" {
		t.Error("expected non-empty version in rollback response")
	}
	if rollbackResp.ReinitError != "" {
		t.Logf("rollback reinit warning: %s", rollbackResp.ReinitError)
	}

	// Step 5: Verify source reflects v1 config after rollback.
	body, code = doGet(t, mcpBasePath+"/sources", nil)
	if code != 200 {
		t.Fatalf("sources after rollback: expected 200, got %d: %s", code, body)
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("parsing sources after rollback: %v", err)
	}
	for _, s := range listResp.Sources {
		if s.ID == sourceID {
			if s.Name != "Rollback Test v1" {
				t.Errorf("expected source name 'Rollback Test v1' after rollback, got %q", s.Name)
			}
		}
	}

	// Clean up.
	_, delCode := doDelete(t, fmt.Sprintf("%s/sources/%s", mcpBasePath, sourceID), operatorHeaders())
	if delCode != 200 {
		t.Logf("cleanup delete returned %d", delCode)
	}
}

// TestPhase4RollbackNotFound verifies that rolling back to a non-existent version
// returns 404.
func TestPhase4RollbackNotFound(t *testing.T) {
	catalogAvailable(t)

	payload := map[string]any{
		"version": "nonexistent-version-abc123",
	}
	_, code, _ := doPost(t, mcpBasePath+"/sources/mcp-default:rollback", payload, operatorHeaders())
	// Expect 404 (revision not found) or 400 (no config store).
	if code != 404 && code != 400 {
		t.Errorf("expected 404 or 400 for rollback to nonexistent version, got %d", code)
	}
}

// --- Phase 4: Apply with Refresh Tests ---

// TestPhase4ApplyWithRefreshAfterApply verifies that POST apply-source with
// refreshAfterApply=true includes a refreshResult in the response.
func TestPhase4ApplyWithRefreshAfterApply(t *testing.T) {
	catalogAvailable(t)

	const sourceID = "e2e-phase4-refresh-source"

	// Apply a valid source with refreshAfterApply=true.
	enabled := true
	refreshAfterApply := true
	payload := map[string]any{
		"id":                sourceID,
		"name":              "Phase4 Refresh Test Source",
		"type":              "yaml",
		"enabled":           enabled,
		"refreshAfterApply": refreshAfterApply,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/phase4-test.yaml",
		},
	}

	body, code, _ := doPost(t, mcpBasePath+"/apply-source", payload, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200 from apply-source with refreshAfterApply, got %d: %s", code, body)
	}

	var result struct {
		Status        string `json:"status"`
		RefreshResult *struct {
			SourceID       string `json:"sourceId,omitempty"`
			EntitiesLoaded int    `json:"entitiesLoaded"`
			Duration       any    `json:"duration"` // may be number (nanoseconds) or string
			Error          string `json:"error,omitempty"`
		} `json:"refreshResult,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing apply response: %v", err)
	}

	if result.Status != "applied" {
		t.Errorf("expected status 'applied', got %q", result.Status)
	}

	if result.RefreshResult == nil {
		t.Error("expected refreshResult in response when refreshAfterApply=true")
	} else {
		// The refresh result should reference our source.
		if result.RefreshResult.SourceID != sourceID {
			t.Errorf("expected refreshResult.sourceId=%q, got %q", sourceID, result.RefreshResult.SourceID)
		}
		t.Logf("refreshResult: sourceId=%s entitiesLoaded=%d error=%q",
			result.RefreshResult.SourceID, result.RefreshResult.EntitiesLoaded, result.RefreshResult.Error)
	}

	// Clean up: delete the test source.
	_, delCode := doDelete(t, fmt.Sprintf("%s/sources/%s", mcpBasePath, sourceID), operatorHeaders())
	if delCode != 200 {
		t.Logf("cleanup delete returned %d", delCode)
	}
}

// TestPhase4ApplyWithoutRefreshAfterApply verifies that apply-source without
// refreshAfterApply does NOT include a refreshResult.
func TestPhase4ApplyWithoutRefreshAfterApply(t *testing.T) {
	catalogAvailable(t)

	const sourceID = "e2e-phase4-no-refresh"

	payload := map[string]any{
		"id":      sourceID,
		"name":    "Phase4 No Refresh Test",
		"type":    "yaml",
		"enabled": true,
		"properties": map[string]any{
			"yamlCatalogPath": "/nonexistent/no-refresh-test.yaml",
		},
	}

	body, code, _ := doPost(t, mcpBasePath+"/apply-source", payload, operatorHeaders())
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}

	var result struct {
		Status        string `json:"status"`
		RefreshResult *struct {
			SourceID string `json:"sourceId,omitempty"`
		} `json:"refreshResult,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing apply response: %v", err)
	}

	if result.Status != "applied" {
		t.Errorf("expected status 'applied', got %q", result.Status)
	}
	if result.RefreshResult != nil {
		t.Error("expected no refreshResult when refreshAfterApply was not set")
	}

	// Clean up.
	_, delCode := doDelete(t, fmt.Sprintf("%s/sources/%s", mcpBasePath, sourceID), operatorHeaders())
	if delCode != 200 {
		t.Logf("cleanup delete returned %d", delCode)
	}
}
