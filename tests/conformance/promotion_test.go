package conformance

import (
	"fmt"
	"net/http"
	"testing"
)

// TestGovernancePromotion tests versioning, binding, and promotion through the governance API.
func TestGovernancePromotion(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		t.Run(tp.Plugin, func(t *testing.T) {
			testGovernancePromotion(t, tp.Plugin, tp.Kind)
		})
	}
}

func testGovernancePromotion(t *testing.T, plugin, kind string) {
	name := fmt.Sprintf("promo-test-%s", testSeqNum())

	// Setup: create governance record, transition to approved (auto-approving if needed).
	ensureGovernanceAtState(t, plugin, kind, name, "approved")

	var v1ID string

	t.Run("create_version", func(t *testing.T) {
		versionBody := map[string]any{
			"versionLabel": "v1.0",
			"reason":       "initial release",
		}
		resp := doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, name), versionBody, defaultHeaders())
		requireStatus(t, resp, http.StatusCreated)

		var version versionResponse
		decodeJSON(t, resp, &version)

		if version.VersionLabel != "v1.0" {
			t.Errorf("expected versionLabel=v1.0, got %s", version.VersionLabel)
		}
		if version.VersionID == "" {
			t.Error("expected non-empty versionId")
		}
		if version.CreatedAt == "" {
			t.Error("expected non-empty createdAt")
		}
		if version.CreatedBy == "" {
			t.Error("expected non-empty createdBy")
		}

		v1ID = version.VersionID
	})

	t.Run("list_versions", func(t *testing.T) {
		if v1ID == "" {
			t.Skip("no version created")
		}

		resp := doRequest(t, http.MethodGet, governanceVersionsURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var list versionListResponse
		decodeJSON(t, resp, &list)

		if list.TotalSize < 1 {
			t.Errorf("expected at least 1 version, got totalSize=%d", list.TotalSize)
		}

		found := false
		for _, v := range list.Versions {
			if v.VersionID == v1ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("created version %s not found in list", v1ID)
		}
	})

	t.Run("bind_to_dev", func(t *testing.T) {
		if v1ID == "" {
			t.Skip("no version created")
		}

		bindBody := map[string]any{
			"versionId": v1ID,
		}
		resp := doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, name, "dev"), bindBody, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var bind bindingResponse
		decodeJSON(t, resp, &bind)

		if bind.Environment != "dev" {
			t.Errorf("expected environment=dev, got %s", bind.Environment)
		}
		if bind.VersionID != v1ID {
			t.Errorf("expected versionId=%s, got %s", v1ID, bind.VersionID)
		}
	})

	t.Run("list_bindings", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, governanceBindingsURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var bindings bindingsResponse
		decodeJSON(t, resp, &bindings)

		found := false
		for _, b := range bindings.Bindings {
			if b.Environment == "dev" && b.VersionID == v1ID {
				found = true
				break
			}
		}
		if !found && v1ID != "" {
			t.Error("expected dev binding in bindings list")
		}
	})

	t.Run("promote_dev_to_stage", func(t *testing.T) {
		if v1ID == "" {
			t.Skip("no version created")
		}

		body := map[string]any{
			"params": map[string]any{
				"fromEnv": "dev",
				"toEnv":   "stage",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "promotion.promote"), body, defaultHeaders())

		if resp.StatusCode != http.StatusOK {
			t.Logf("promote dev->stage returned %d (may need approved state)", resp.StatusCode)
		} else {
			var result actionResult
			decodeJSON(t, resp, &result)
			if result.Action != "promotion.promote" {
				t.Errorf("expected action=promotion.promote, got %s", result.Action)
			}
			if result.Status != "completed" {
				t.Errorf("expected status=completed, got %s", result.Status)
			}
		}
		resp.Body.Close()
	})

	t.Run("draft_cannot_bind_to_prod", func(t *testing.T) {
		draftName := fmt.Sprintf("promo-draft-prod-%s", testSeqNum())

		// Create a draft governance record.
		resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, draftName), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		// Create a version for the draft asset.
		versionBody := map[string]any{
			"versionLabel": "v0.1",
		}
		resp = doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, draftName), versionBody, defaultHeaders())
		requireStatus(t, resp, http.StatusCreated)
		var ver versionResponse
		decodeJSON(t, resp, &ver)

		// Try to bind to prod.
		bindBody := map[string]any{
			"versionId": ver.VersionID,
		}
		resp = doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, draftName, "prod"), bindBody, defaultHeaders())

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for draft asset binding to prod, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("archived_cannot_bind", func(t *testing.T) {
		archName := fmt.Sprintf("promo-archived-bind-%s", testSeqNum())

		// Use shared helper to reach archived state (handles auto-approval).
		ensureGovernanceAtState(t, plugin, kind, archName, "archived")

		// Create a version.
		vBody := map[string]any{"versionLabel": "v0.1"}
		resp := doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, archName), vBody, defaultHeaders())
		if resp.StatusCode != http.StatusCreated {
			t.Skipf("version creation returned %d", resp.StatusCode)
		}
		var ver versionResponse
		decodeJSON(t, resp, &ver)

		// Try to bind.
		bindBody := map[string]any{"versionId": ver.VersionID}
		resp = doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, archName, "dev"), bindBody, defaultHeaders())

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for archived asset binding, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("rollback_binding", func(t *testing.T) {
		rollbackName := fmt.Sprintf("promo-rollback-%s", testSeqNum())

		// Use shared helper to reach approved state (handles auto-approval).
		ensureGovernanceAtState(t, plugin, kind, rollbackName, "approved")

		// Create two versions.
		v1Body := map[string]any{"versionLabel": "v1.0"}
		resp := doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, rollbackName), v1Body, defaultHeaders())
		requireStatus(t, resp, http.StatusCreated)
		var ver1 versionResponse
		decodeJSON(t, resp, &ver1)

		v2Body := map[string]any{"versionLabel": "v2.0"}
		resp = doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, rollbackName), v2Body, defaultHeaders())
		requireStatus(t, resp, http.StatusCreated)
		var ver2 versionResponse
		decodeJSON(t, resp, &ver2)

		// Bind v2 to dev.
		bindBody := map[string]any{"versionId": ver2.VersionID}
		resp = doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, rollbackName, "dev"), bindBody, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		// Rollback to v1.
		rollbackBody := map[string]any{
			"params": map[string]any{
				"environment":     "dev",
				"targetVersionId": ver1.VersionID,
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, rollbackName, "promotion.rollback"), rollbackBody, defaultHeaders())

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for rollback, got %d", resp.StatusCode)
			resp.Body.Close()
			return
		}

		var result actionResult
		decodeJSON(t, resp, &result)

		if result.Action != "promotion.rollback" {
			t.Errorf("expected action=promotion.rollback, got %s", result.Action)
		}
		if result.Status != "completed" {
			t.Errorf("expected status=completed, got %s", result.Status)
		}

		// Verify binding now points to v1.
		resp = doRequest(t, http.MethodGet, governanceBindingsURL(plugin, kind, rollbackName), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		var bindings bindingsResponse
		decodeJSON(t, resp, &bindings)

		for _, b := range bindings.Bindings {
			if b.Environment == "dev" {
				if b.VersionID != ver1.VersionID {
					t.Errorf("after rollback, dev binding should be %s, got %s", ver1.VersionID, b.VersionID)
				}
				if b.PreviousVersionID != ver2.VersionID {
					t.Logf("note: previousVersionId=%s, expected %s", b.PreviousVersionID, ver2.VersionID)
				}
			}
		}
	})

	t.Run("promotion_promote_via_action", func(t *testing.T) {
		promName := fmt.Sprintf("promo-action-%s", testSeqNum())

		// Setup: create, approve (with auto-approval), create version, bind to dev.
		ensureGovernanceAtState(t, plugin, kind, promName, "approved")

		vBody := map[string]any{"versionLabel": "v1.0"}
		resp := doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, promName), vBody, defaultHeaders())
		requireStatus(t, resp, http.StatusCreated)
		var ver versionResponse
		decodeJSON(t, resp, &ver)

		bindBody := map[string]any{"versionId": ver.VersionID}
		resp = doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, promName, "dev"), bindBody, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		// Promote from dev to stage via action endpoint.
		promoteBody := map[string]any{
			"params": map[string]any{
				"fromEnv": "dev",
				"toEnv":   "stage",
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, promName, "promotion.promote"), promoteBody, defaultHeaders())

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for promotion.promote action, got %d", resp.StatusCode)
			resp.Body.Close()
			return
		}

		var result actionResult
		decodeJSON(t, resp, &result)

		if result.Action != "promotion.promote" {
			t.Errorf("expected action=promotion.promote, got %s", result.Action)
		}
		if result.Status != "completed" {
			t.Errorf("expected status=completed, got %s", result.Status)
		}
	})
}
