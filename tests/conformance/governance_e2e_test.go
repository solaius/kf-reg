package conformance

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestGovernanceE2EFullLifecycle exercises the complete governance lifecycle:
//  1. Create asset -> set governance metadata (owner, risk)
//  2. Attempt approve -> approval request created (or immediate if no policy gate)
//  3. Approve -> lifecycle transitions to approved
//  4. Create version v1.0 -> bind to dev -> promote to stage
//  5. Create version v2.0 -> bind to dev -> promote to prod
//  6. Rollback prod to v1.0
//  7. Deprecate -> archive
//  8. Verify full audit trail
func TestGovernanceE2EFullLifecycle(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		plugin, kind := tp.Plugin, tp.Kind
		t.Run(plugin, func(t *testing.T) {
			testGovernanceE2EFullLifecycle(t, plugin, kind)
		})
	}
}

func testGovernanceE2EFullLifecycle(t *testing.T, plugin, kind string) {
	name := fmt.Sprintf("e2e-full-%s", testSeqNum())

	//
	// Step 1: Create asset and set governance metadata.
	//
	t.Log("Step 1: Create governance record and set metadata")
	resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)

	var initial governanceResponse
	decodeJSON(t, resp, &initial)
	if initial.Governance.Lifecycle == nil || initial.Governance.Lifecycle.State != "draft" {
		t.Fatalf("expected initial state=draft, got %v", initial.Governance.Lifecycle)
	}

	// Set owner and risk.
	patch := governanceOverlay{
		Owner: &ownerInfo{
			Principal:   "e2e-alice@example.com",
			DisplayName: "E2E Alice",
			Email:       "e2e-alice@example.com",
		},
		Team: &teamInfo{
			Name: "ml-ops",
			ID:   "team-e2e",
		},
		Risk: &riskInfo{
			Level:      "high",
			Categories: []string{"pii"},
		},
		SLA: &slaInfo{
			Tier:          "gold",
			ResponseHours: 2,
		},
		Compliance: &complianceInfo{
			Tags:     []string{"gdpr"},
			Controls: []string{"AC-1"},
		},
	}
	resp = doRequest(t, http.MethodPatch, governanceURL(plugin, kind, name), patch, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)

	var patched governanceResponse
	decodeJSON(t, resp, &patched)
	if patched.Governance.Owner == nil || patched.Governance.Owner.Principal != "e2e-alice@example.com" {
		t.Fatalf("PATCH did not update owner")
	}

	//
	// Step 2: Attempt approve (may be gated or immediate).
	//
	t.Log("Step 2: Transition draft -> approved")
	approveBody := map[string]any{
		"params": map[string]any{
			"state":  "approved",
			"reason": "e2e test approval",
		},
	}
	resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), approveBody, defaultHeaders())

	isApproved := false
	approvalRequestID := ""

	switch resp.StatusCode {
	case http.StatusOK:
		// Direct approval, no gate.
		var result actionResult
		decodeJSON(t, resp, &result)
		if result.Status != "completed" {
			t.Fatalf("expected completed, got %s", result.Status)
		}
		isApproved = true
		t.Log("  -> Approved directly (no policy gate)")

	case http.StatusAccepted:
		// Approval request created.
		var result actionResult
		decodeJSON(t, resp, &result)
		if result.Status != "pending-approval" {
			t.Fatalf("expected pending-approval, got %s", result.Status)
		}
		approvalRequestID, _ = result.Data["requestId"].(string)
		t.Logf("  -> Approval request created: %s", approvalRequestID)

		//
		// Step 3: Submit approval decision.
		//
		t.Log("Step 3: Submit approval decision")
		if approvalRequestID != "" {
			reviewerHeaders := map[string]string{
				"X-User-Principal": "e2e-reviewer",
			}
			decisionBody := map[string]any{
				"verdict": "approve",
				"comment": "e2e test approval granted",
			}
			decisionURL := fmt.Sprintf("%s%s/approvals/%s/decisions", serverURL, govBasePath, approvalRequestID)
			resp = doRequest(t, http.MethodPost, decisionURL, decisionBody, reviewerHeaders)

			if resp.StatusCode == http.StatusOK {
				var decisionResp map[string]any
				decodeJSON(t, resp, &decisionResp)
				if autoExec, ok := decisionResp["autoExecuted"].(bool); ok && autoExec {
					isApproved = true
					t.Log("  -> Approval auto-executed")
				} else {
					t.Log("  -> Decision submitted, may need more approvals")
					// Try to verify if we're approved now.
					resp = doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
					requireStatus(t, resp, http.StatusOK)
					var gov governanceResponse
					decodeJSON(t, resp, &gov)
					if gov.Governance.Lifecycle != nil && gov.Governance.Lifecycle.State == "approved" {
						isApproved = true
					}
				}
			} else {
				resp.Body.Close()
				t.Logf("  -> Decision submission returned %d", resp.StatusCode)
			}
		}

	default:
		resp.Body.Close()
		t.Fatalf("unexpected status %d for approve transition", resp.StatusCode)
	}

	if !isApproved {
		t.Log("Asset not in approved state; some promotion tests will be skipped")
	}

	//
	// Step 4: Create version v1.0, bind to dev, promote to stage.
	//
	t.Log("Step 4: Create v1.0, bind to dev")
	v1Body := map[string]any{
		"versionLabel": "v1.0",
		"reason":       "initial release",
	}
	resp = doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, name), v1Body, defaultHeaders())
	requireStatus(t, resp, http.StatusCreated)

	var v1 versionResponse
	decodeJSON(t, resp, &v1)
	t.Logf("  -> Created version %s (%s)", v1.VersionLabel, v1.VersionID)

	// Bind v1 to dev.
	bindDevBody := map[string]any{"versionId": v1.VersionID}
	resp = doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, name, "dev"), bindDevBody, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	t.Log("  -> Bound v1.0 to dev")

	// Promote dev -> stage (requires approved state).
	if isApproved {
		promoteBody := map[string]any{
			"params": map[string]any{
				"fromEnv": "dev",
				"toEnv":   "stage",
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "promotion.promote"), promoteBody, defaultHeaders())
		if resp.StatusCode == http.StatusOK {
			t.Log("  -> Promoted v1.0 from dev to stage")
		} else {
			t.Logf("  -> Promote dev->stage returned %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	//
	// Step 5: Create version v2.0, bind to dev, promote to prod.
	//
	t.Log("Step 5: Create v2.0, bind to dev")
	v2Body := map[string]any{
		"versionLabel": "v2.0",
		"reason":       "major update",
	}
	resp = doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, name), v2Body, defaultHeaders())
	requireStatus(t, resp, http.StatusCreated)

	var v2 versionResponse
	decodeJSON(t, resp, &v2)
	t.Logf("  -> Created version %s (%s)", v2.VersionLabel, v2.VersionID)

	// Bind v2 to dev.
	bindDevBody2 := map[string]any{"versionId": v2.VersionID}
	resp = doRequest(t, http.MethodPut, governanceBindingEnvURL(plugin, kind, name, "dev"), bindDevBody2, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	t.Log("  -> Bound v2.0 to dev")

	// Promote dev -> prod.
	if isApproved {
		promoteBody := map[string]any{
			"params": map[string]any{
				"fromEnv": "dev",
				"toEnv":   "prod",
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "promotion.promote"), promoteBody, defaultHeaders())
		if resp.StatusCode == http.StatusOK {
			t.Log("  -> Promoted v2.0 from dev to prod")
		} else {
			t.Logf("  -> Promote dev->prod returned %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	//
	// Step 6: Rollback prod to v1.0.
	//
	if isApproved {
		t.Log("Step 6: Rollback prod to v1.0")
		rollbackBody := map[string]any{
			"params": map[string]any{
				"environment":     "prod",
				"targetVersionId": v1.VersionID,
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "promotion.rollback"), rollbackBody, defaultHeaders())
		if resp.StatusCode == http.StatusOK {
			var result actionResult
			decodeJSON(t, resp, &result)
			if result.Status != "completed" {
				t.Errorf("expected rollback status=completed, got %s", result.Status)
			}
			t.Log("  -> Rolled back prod to v1.0")
		} else {
			t.Logf("  -> Rollback returned %d", resp.StatusCode)
			resp.Body.Close()
		}

		// Verify prod binding is v1.
		resp = doRequest(t, http.MethodGet, governanceBindingsURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		var bindings bindingsResponse
		decodeJSON(t, resp, &bindings)

		for _, b := range bindings.Bindings {
			if b.Environment == "prod" {
				if b.VersionID != v1.VersionID {
					t.Errorf("after rollback, prod should be %s, got %s", v1.VersionID, b.VersionID)
				}
			}
		}
	}

	//
	// Step 7: Deprecate -> Archive.
	//
	if isApproved {
		t.Log("Step 7: Deprecate and archive")

		// Deprecate.
		deprecateBody := map[string]any{
			"params": map[string]any{
				"reason": "end of life",
			},
		}
		resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.deprecate"), deprecateBody, defaultHeaders())
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Logf("  -> Deprecate returned %d", resp.StatusCode)
			resp.Body.Close()
		} else {
			resp.Body.Close()
			t.Log("  -> Deprecated")

			// Archive.
			archiveBody := map[string]any{
				"params": map[string]any{
					"reason": "no longer needed",
				},
			}
			resp = doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.archive"), archiveBody, defaultHeaders())
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
				t.Log("  -> Archived")
			} else {
				t.Logf("  -> Archive returned %d (may require approval)", resp.StatusCode)
			}
			resp.Body.Close()
		}
	}

	//
	// Step 8: Verify full audit trail.
	//
	t.Log("Step 8: Verify audit trail")
	time.Sleep(200 * time.Millisecond)

	resp = doRequest(t, http.MethodGet, governanceHistoryURL(plugin, kind, name), nil, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)

	var history auditEventList
	decodeJSON(t, resp, &history)

	t.Logf("Audit trail has %d events (totalSize=%d)", len(history.Events), history.TotalSize)
	if history.TotalSize < 2 {
		t.Error("expected at least 2 audit events for the full lifecycle")
	}

	// Log all events for visibility.
	eventTypes := make(map[string]int)
	for _, event := range history.Events {
		eventTypes[event.EventType]++
		t.Logf("  [%s] %s actor=%s outcome=%s action=%s",
			event.CreatedAt, event.EventType, event.Actor, event.Outcome, event.Action)
	}

	// Verify we have at least a metadata change event.
	if eventTypes["governance.metadata.changed"] < 1 {
		t.Error("expected at least 1 governance.metadata.changed event")
	}

	// If we successfully transitioned, we should have lifecycle events.
	if isApproved {
		lifecycleEvents := eventTypes["governance.lifecycle.changed"] +
			eventTypes["governance.approval.requested"]
		if lifecycleEvents < 1 {
			t.Error("expected at least 1 lifecycle or approval event")
		}
	}

	// Verify version creation event.
	if eventTypes["governance.version.created"] < 1 {
		t.Error("expected at least 1 governance.version.created event")
	}

	t.Log("E2E full lifecycle test complete")
}
