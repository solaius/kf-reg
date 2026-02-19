package conformance

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func testEndpoints(t *testing.T, p pluginInfo) {
	t.Helper()

	if p.CapabilitiesV2 == nil {
		t.Skip("no V2 capabilities, skipping endpoint tests")
	}

	for _, entity := range p.CapabilitiesV2.Entities {
		t.Run("list_"+entity.Plural, func(t *testing.T) {
			resp, err := http.Get(serverURL + entity.Endpoints.List)
			if err != nil {
				t.Fatalf("GET %s failed: %v", entity.Endpoints.List, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("GET %s returned %d: %s", entity.Endpoints.List, resp.StatusCode, string(body))
			}

			// Verify response is valid JSON with expected structure.
			var result map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			// Should have items array.
			if _, hasItems := result["items"]; !hasItems {
				if _, hasSize := result["size"]; !hasSize {
					t.Log("warning: response has neither 'items' nor 'size' field")
				}
			}

			// If items is present it should be an array.
			if items, ok := result["items"]; ok {
				if _, isArr := items.([]any); !isArr {
					t.Error("'items' field is not an array")
				}
			}

			// Size should be a number.
			if size, ok := result["size"]; ok {
				if _, isNum := size.(float64); !isNum {
					t.Error("'size' field is not a number")
				}
			}
		})

		t.Run("get_"+entity.Plural+"_first", func(t *testing.T) {
			useManagementEndpoint := strings.Count(entity.Endpoints.Get, "{") > 1

			// Get the list to find an entity name.
			var listResp map[string]any
			getJSON(t, entity.Endpoints.List, &listResp)

			items, ok := listResp["items"].([]any)
			if !ok || len(items) == 0 {
				t.Skip("no items to test get endpoint")
			}

			first, ok := items[0].(map[string]any)
			if !ok {
				t.Fatal("first item is not a JSON object")
			}

			name, _ := first["name"].(string)
			if name == "" {
				t.Skip("first item has no name field")
			}

			// Build the get URL — use management endpoint for multi-param patterns.
			var getURL string
			if useManagementEndpoint {
				// Extract basePath by finding the part before the last path segment.
				listPath := entity.Endpoints.List
				lastSlash := strings.LastIndex(listPath, "/")
				basePath := listPath[:lastSlash]
				getURL = basePath + "/management/entities/" + name
			} else {
				getURL = strings.Replace(entity.Endpoints.Get, "{name}", name, 1)
			}

			resp, err := http.Get(serverURL + getURL)
			if err != nil {
				t.Fatalf("GET %s failed: %v", getURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("GET %s returned %d: %s", getURL, resp.StatusCode, string(body))
			}

			// Verify the detail response is valid JSON.
			var detail map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
				t.Fatalf("detail response is not valid JSON: %v", err)
			}

			// The detail should have a name field matching what we requested.
			detailName, _ := detail["name"].(string)
			if detailName != name {
				t.Errorf("detail name %q does not match requested %q", detailName, name)
			}
		})

		t.Run("get_"+entity.Plural+"_not_found", func(t *testing.T) {
			useManagementEndpoint := strings.Count(entity.Endpoints.Get, "{") > 1

			var getURL string
			if useManagementEndpoint {
				listPath := entity.Endpoints.List
				lastSlash := strings.LastIndex(listPath, "/")
				basePath := listPath[:lastSlash]
				getURL = basePath + "/entities/nonexistent-entity-12345"
			} else {
				getURL = strings.Replace(entity.Endpoints.Get, "{name}", "nonexistent-entity-12345", 1)
			}

			resp, err := http.Get(serverURL + getURL)
			if err != nil {
				t.Fatalf("GET %s failed: %v", getURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				t.Errorf("expected non-200 for nonexistent entity, got %d", resp.StatusCode)
			}
			// Acceptable responses: 404, 400, or 500 (though 404 is preferred).
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("note: nonexistent entity returned %d (expected 404): %s", resp.StatusCode, string(body))
			}
		})
	}
}
