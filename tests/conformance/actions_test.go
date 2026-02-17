package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func testActions(t *testing.T, p pluginInfo) {
	t.Helper()

	if p.CapabilitiesV2 == nil || len(p.CapabilitiesV2.Actions) == 0 {
		t.Skip("no actions defined")
	}

	for _, entity := range p.CapabilitiesV2.Entities {
		if len(entity.Actions) == 0 {
			continue
		}

		// Get an entity name to test actions on.
		var listResp map[string]any
		getJSON(t, entity.Endpoints.List, &listResp)

		items, ok := listResp["items"].([]any)
		if !ok || len(items) == 0 {
			t.Logf("no items for %s to test actions, skipping", entity.Kind)
			continue
		}

		first, ok := items[0].(map[string]any)
		if !ok {
			continue
		}
		entityName, _ := first["name"].(string)
		if entityName == "" {
			continue
		}

		// Test each declared action.
		for _, actionID := range entity.Actions {
			t.Run(fmt.Sprintf("%s/%s/%s", entity.Plural, entityName, actionID), func(t *testing.T) {
				// Find the action definition.
				var actionDef *actionDefinition
				for i, a := range p.CapabilitiesV2.Actions {
					if a.ID == actionID {
						actionDef = &p.CapabilitiesV2.Actions[i]
						break
					}
				}
				if actionDef == nil {
					t.Skipf("action %q not found in plugin actions", actionID)
					return
				}

				// Only test asset-scoped actions here.
				if actionDef.Scope != "asset" {
					t.Skipf("action %q has scope %q, skipping (not asset-scoped)", actionID, actionDef.Scope)
					return
				}

				// Build action request. Use dry-run when supported.
				req := map[string]any{
					"action": actionID,
					"dryRun": actionDef.SupportsDryRun,
				}

				// Add minimal params based on known action types.
				switch actionID {
				case "tag":
					req["params"] = map[string]any{"tags": []string{"conformance-test"}}
				case "annotate":
					req["params"] = map[string]any{"annotations": map[string]string{"test": "conformance"}}
				case "deprecate":
					// No special params needed.
				}

				body, _ := json.Marshal(req)

				// Build action URL using the management entities route.
				actionURL := fmt.Sprintf("%s/entities/%s:action", p.BasePath, entityName)

				httpReq, err := http.NewRequest(http.MethodPost, serverURL+actionURL, bytes.NewReader(body))
				if err != nil {
					t.Fatalf("build request for %s failed: %v", actionURL, err)
				}
				httpReq.Header.Set("Content-Type", "application/json")
				httpReq.Header.Set("X-User-Role", "operator")

				resp, err := http.DefaultClient.Do(httpReq)
				if err != nil {
					t.Fatalf("POST %s failed: %v", actionURL, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusMethodNotAllowed {
					t.Skipf("POST %s returned 405 (action route not mounted for this entity)", actionURL)
				}
				if resp.StatusCode != http.StatusOK {
					respBody, _ := io.ReadAll(resp.Body)
					t.Fatalf("POST %s returned %d: %s", actionURL, resp.StatusCode, string(respBody))
				}

				var result map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("response is not valid JSON: %v", err)
				}

				// Verify action result structure.
				if _, ok := result["action"]; !ok {
					t.Error("response missing 'action' field")
				}
				if _, ok := result["status"]; !ok {
					t.Error("response missing 'status' field")
				}

				status, _ := result["status"].(string)
				validStatuses := map[string]bool{
					"completed": true,
					"dry-run":   true,
					"error":     true,
				}
				if !validStatuses[status] {
					t.Errorf("unexpected action status %q", status)
				}

				// If dry-run was requested, status should be "dry-run".
				if actionDef.SupportsDryRun {
					if status != "dry-run" {
						t.Logf("note: dry-run requested but got status %q", status)
					}
				}
			})
		}
	}
}
