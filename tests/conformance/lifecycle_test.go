package conformance

import (
	"fmt"
	"net/http"
	"testing"
)

// TestGovernanceLifecycle tests lifecycle state transitions through the governance action API.
func TestGovernanceLifecycle(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		plugin, kind := tp.Plugin, tp.Kind
		t.Run(plugin, func(t *testing.T) {
			testGovernanceLifecycle(t, plugin, kind)
		})
	}
}

func testGovernanceLifecycle(t *testing.T, plugin, kind string) {
	// Helper to set lifecycle state via the action endpoint.
	setLifecycleState := func(t *testing.T, name, state, reason string) (*http.Response, actionResult) {
		t.Helper()
		body := map[string]any{
			"params": map[string]any{
				"state":  state,
				"reason": reason,
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())
		var result actionResult
		decodeJSON(t, resp, &result)
		return resp, result
	}

	t.Run("draft_to_approved", func(t *testing.T) {
		name := fmt.Sprintf("lc-draft-approved-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "draft")

		resp, result := setLifecycleState(t, name, "approved", "ready for production")

		// The transition may succeed directly (200) or create an approval request (202).
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200 or 202, got %d", resp.StatusCode)
		}

		if result.Action != "lifecycle.setState" {
			t.Errorf("expected action=lifecycle.setState, got %s", result.Action)
		}
		if result.Status != "completed" && result.Status != "pending-approval" {
			t.Errorf("expected status completed or pending-approval, got %s", result.Status)
		}
	})

	t.Run("approved_to_deprecated", func(t *testing.T) {
		name := fmt.Sprintf("lc-approved-deprecated-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "approved")

		resp, result := setLifecycleState(t, name, "deprecated", "end of life")

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200 or 202, got %d", resp.StatusCode)
		}
		if result.Status != "completed" && result.Status != "pending-approval" {
			t.Errorf("expected status completed or pending-approval, got %s", result.Status)
		}
	})

	t.Run("deprecated_to_archived", func(t *testing.T) {
		name := fmt.Sprintf("lc-deprecated-archived-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "deprecated")

		resp, result := setLifecycleState(t, name, "archived", "no longer needed")

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200 or 202, got %d", resp.StatusCode)
		}
		if result.Status != "completed" && result.Status != "pending-approval" {
			t.Errorf("expected status completed or pending-approval, got %s", result.Status)
		}
	})

	t.Run("draft_to_archived_denied", func(t *testing.T) {
		name := fmt.Sprintf("lc-draft-archived-deny-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "draft")

		body := map[string]any{
			"params": map[string]any{
				"state":  "archived",
				"reason": "skip steps",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for draft->archived, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("draft_to_deprecated_denied", func(t *testing.T) {
		name := fmt.Sprintf("lc-draft-deprecated-deny-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "draft")

		body := map[string]any{
			"params": map[string]any{
				"state":  "deprecated",
				"reason": "skip steps",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for draft->deprecated, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("archived_to_approved_denied", func(t *testing.T) {
		name := fmt.Sprintf("lc-archived-approved-deny-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "archived")

		body := map[string]any{
			"params": map[string]any{
				"state":  "approved",
				"reason": "skip steps",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for archived->approved, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("lifecycle_deprecate_convenience", func(t *testing.T) {
		name := fmt.Sprintf("lc-deprecate-conv-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "approved")

		body := map[string]any{
			"params": map[string]any{
				"reason": "convenience deprecation",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.deprecate"), body, defaultHeaders())

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200 or 202, got %d", resp.StatusCode)
		}

		var result actionResult
		decodeJSON(t, resp, &result)
		if result.Action != "lifecycle.deprecate" {
			t.Errorf("expected action=lifecycle.deprecate, got %s", result.Action)
		}
	})

	t.Run("lifecycle_restore", func(t *testing.T) {
		name := fmt.Sprintf("lc-restore-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "archived")

		body := map[string]any{
			"params": map[string]any{
				"targetState": "deprecated",
				"reason":      "restore from archive",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.restore"), body, defaultHeaders())

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200 or 202, got %d", resp.StatusCode)
		}

		var result actionResult
		decodeJSON(t, resp, &result)
		if result.Action != "lifecycle.restore" {
			t.Errorf("expected action=lifecycle.restore, got %s", result.Action)
		}
	})

	t.Run("invalid_action_name", func(t *testing.T) {
		name := fmt.Sprintf("lc-invalid-action-%s", testSeqNum())
		ensureGovernanceAtState(t, plugin, kind, name, "draft")

		body := map[string]any{
			"params": map[string]any{},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.nonexistent"), body, defaultHeaders())

		// Unknown lifecycle actions should return 400.
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for unknown action, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})
}
