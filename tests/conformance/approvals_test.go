package conformance

import (
	"fmt"
	"net/http"
	"testing"
)

// TestGovernanceApprovals tests the approval flow through the governance API.
func TestGovernanceApprovals(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		t.Run(tp.Plugin, func(t *testing.T) {
			testGovernanceApprovals(t, tp.Plugin, tp.Kind)
		})
	}
}

func testGovernanceApprovals(t *testing.T, plugin, kind string) {
	t.Helper()

	approvalsURL := fmt.Sprintf("%s%s/approvals", serverURL, govBasePath)

	t.Run("list_approvals_initially_empty_or_valid", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, approvalsURL, nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var list approvalRequestList
		decodeJSON(t, resp, &list)

		// The list should be valid (may or may not have pre-existing approvals).
		if list.Requests == nil {
			t.Error("requests should not be nil (should be empty array)")
		}
		if list.TotalSize < 0 {
			t.Errorf("totalSize should be >= 0, got %d", list.TotalSize)
		}
	})

	t.Run("gated_transition_creates_approval", func(t *testing.T) {
		name := fmt.Sprintf("approval-test-%s", testSeqNum())

		// Create default governance record at draft state.
		resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		// Set high risk to increase chance of approval gate matching.
		patch := governanceOverlay{
			Risk: &riskInfo{
				Level:      "critical",
				Categories: []string{"sensitive"},
			},
		}
		resp = doRequest(t, http.MethodPatch, governanceURL(plugin, kind, name), patch, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		// Attempt draft->approved transition.
		body := map[string]any{
			"params": map[string]any{
				"state":  "approved",
				"reason": "needs approval gate test",
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())

		if resp.StatusCode == http.StatusAccepted {
			// An approval request was created. Verify we can find it.
			var result actionResult
			decodeJSON(t, resp, &result)

			if result.Status != "pending-approval" {
				t.Errorf("expected status=pending-approval, got %s", result.Status)
			}

			requestID, _ := result.Data["requestId"].(string)
			if requestID == "" {
				t.Fatal("expected requestId in data when pending-approval")
			}

			// Get the approval request.
			approvalGetURL := fmt.Sprintf("%s/%s", approvalsURL, requestID)
			resp2 := doRequest(t, http.MethodGet, approvalGetURL, nil, defaultHeaders())
			requireStatus(t, resp2, http.StatusOK)

			var approval approvalRequest
			decodeJSON(t, resp2, &approval)

			if approval.ID != requestID {
				t.Errorf("expected approval ID=%s, got %s", requestID, approval.ID)
			}
			if approval.Status != "pending" {
				t.Errorf("expected approval status=pending, got %s", approval.Status)
			}
			if approval.AssetRef.Name != name {
				t.Errorf("expected approval assetRef.name=%s, got %s", name, approval.AssetRef.Name)
			}

			// Submit a decision.
			t.Run("submit_decision", func(t *testing.T) {
				decisionHeaders := map[string]string{
					"X-User-Principal": "reviewer-alice",
				}
				decisionBody := map[string]any{
					"verdict": "approve",
					"comment": "looks good",
				}
				decisionURL := fmt.Sprintf("%s/%s/decisions", approvalsURL, requestID)
				resp3 := doRequest(t, http.MethodPost, decisionURL, decisionBody, decisionHeaders)

				if resp3.StatusCode != http.StatusOK {
					t.Logf("submit decision returned %d (may need more approvals or policy-specific handling)", resp3.StatusCode)
				}
				resp3.Body.Close()
			})

			// Test cancel.
			t.Run("cancel_approval", func(t *testing.T) {
				// Create a new asset for an isolated cancel test.
				cancelName := fmt.Sprintf("approval-cancel-%s", testSeqNum())
				resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, cancelName), nil, defaultHeaders())
				requireStatus(t, resp, http.StatusOK)
				resp.Body.Close()

				// Set high risk.
				patchResp := doRequest(t, http.MethodPatch, governanceURL(plugin, kind, cancelName), governanceOverlay{
					Risk: &riskInfo{Level: "critical"},
				}, defaultHeaders())
				requireStatus(t, patchResp, http.StatusOK)
				patchResp.Body.Close()

				// Attempt gated transition.
				body := map[string]any{
					"params": map[string]any{"state": "approved", "reason": "cancel test"},
				}
				resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, cancelName, "lifecycle.setState"), body, defaultHeaders())
				if resp.StatusCode != http.StatusAccepted {
					t.Skip("no approval gate triggered for cancel test")
				}

				var cancelResult actionResult
				decodeJSON(t, resp, &cancelResult)
				cancelReqID, _ := cancelResult.Data["requestId"].(string)
				if cancelReqID == "" {
					t.Skip("no requestId for cancel test")
				}

				cancelURL := fmt.Sprintf("%s/%s/cancel", approvalsURL, cancelReqID)
				cancelBody := map[string]any{
					"reason": "changed my mind",
				}
				cancelResp := doRequest(t, http.MethodPost, cancelURL, cancelBody, defaultHeaders())

				if cancelResp.StatusCode != http.StatusOK {
					t.Errorf("expected 200 for cancel, got %d", cancelResp.StatusCode)
				}
				cancelResp.Body.Close()
			})
		} else if resp.StatusCode == http.StatusOK {
			// No approval gate configured; transition was immediate.
			var result actionResult
			decodeJSON(t, resp, &result)
			t.Logf("transition executed directly (no approval gate): status=%s", result.Status)
		} else {
			body, _ := fmt.Printf("")
			_ = body
			t.Logf("unexpected status %d for gated transition test", resp.StatusCode)
			resp.Body.Close()
		}
	})

	t.Run("list_approvals_with_filter", func(t *testing.T) {
		// Filter by status=pending.
		filterURL := fmt.Sprintf("%s?status=pending&pageSize=5", approvalsURL)
		resp := doRequest(t, http.MethodGet, filterURL, nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var list approvalRequestList
		decodeJSON(t, resp, &list)

		// Verify each returned request has pending status.
		for _, req := range list.Requests {
			if req.Status != "pending" {
				t.Errorf("filtered by status=pending but got request with status=%s", req.Status)
			}
		}
	})
}

// TestGovernancePolicies tests the policies endpoint.
func TestGovernancePolicies(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	policiesURL := fmt.Sprintf("%s%s/policies", serverURL, govBasePath)
	resp := doRequest(t, http.MethodGet, policiesURL, nil, defaultHeaders())

	// Policies endpoint may or may not be mounted. If no approval engine
	// is configured, this may return 404.
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("policies endpoint not mounted (approval engine not configured)")
	}
	requireStatus(t, resp, http.StatusOK)

	var result map[string]any
	decodeJSON(t, resp, &result)

	policies, ok := result["policies"]
	if !ok {
		t.Error("response missing 'policies' field")
	}
	if policies == nil {
		t.Log("policies list is nil (no policies configured)")
	}
}
