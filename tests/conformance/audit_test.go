package conformance

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestGovernanceAudit tests the audit trail through the governance history API.
func TestGovernanceAudit(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		plugin, kind := tp.Plugin, tp.Kind
		t.Run(plugin, func(t *testing.T) {
			testGovernanceAudit(t, plugin, kind)
		})
	}
}

func testGovernanceAudit(t *testing.T, plugin, kind string) {
	name := fmt.Sprintf("audit-test-%s", testSeqNum())

	// Create governance record (this may or may not emit an audit event).
	resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	t.Run("audit_after_patch", func(t *testing.T) {
		// Patch governance metadata.
		patch := governanceOverlay{
			Owner: &ownerInfo{
				Principal:   "audit-alice@example.com",
				DisplayName: "Audit Alice",
			},
			Risk: &riskInfo{
				Level: "high",
			},
		}
		resp := doRequest(t, http.MethodPatch, governanceURL(plugin, kind, name), patch, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		// Small delay to ensure audit event is committed.
		time.Sleep(100 * time.Millisecond)

		// Check audit history.
		resp = doRequest(t, http.MethodGet, governanceHistoryURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var history auditEventList
		decodeJSON(t, resp, &history)

		if history.TotalSize < 1 {
			t.Error("expected at least 1 audit event after PATCH")
		}

		foundPatchEvent := false
		for _, event := range history.Events {
			if event.EventType == "governance.metadata.changed" && event.Action == "patch" {
				foundPatchEvent = true
				if event.Outcome != "success" {
					t.Errorf("expected outcome=success for patch event, got %s", event.Outcome)
				}
				if event.ID == "" {
					t.Error("audit event ID is empty")
				}
				if event.CreatedAt == "" {
					t.Error("audit event createdAt is empty")
				}
				break
			}
		}
		if !foundPatchEvent {
			t.Error("audit history does not contain governance.metadata.changed event after PATCH")
		}
	})

	t.Run("audit_after_lifecycle_transition", func(t *testing.T) {
		// Transition draft -> approved.
		body := map[string]any{
			"params": map[string]any{
				"state":  "approved",
				"reason": "audit lifecycle test",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Skipf("lifecycle transition returned %d, cannot test audit", resp.StatusCode)
		}
		resp.Body.Close()

		time.Sleep(100 * time.Millisecond)

		resp = doRequest(t, http.MethodGet, governanceHistoryURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var history auditEventList
		decodeJSON(t, resp, &history)

		// Look for a lifecycle or approval event.
		foundLifecycleEvent := false
		for _, event := range history.Events {
			if event.EventType == "governance.lifecycle.changed" ||
				event.EventType == "governance.approval.requested" {
				foundLifecycleEvent = true
				break
			}
		}
		if !foundLifecycleEvent {
			t.Error("audit history does not contain lifecycle event after state transition")
		}
	})

	t.Run("audit_after_version_create", func(t *testing.T) {
		versionBody := map[string]any{
			"versionLabel": "v1.0-audit",
			"reason":       "audit version test",
		}
		resp := doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, name), versionBody, defaultHeaders())
		if resp.StatusCode != http.StatusCreated {
			t.Skipf("version creation returned %d, cannot test audit", resp.StatusCode)
		}
		resp.Body.Close()

		time.Sleep(100 * time.Millisecond)

		resp = doRequest(t, http.MethodGet, governanceHistoryURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var history auditEventList
		decodeJSON(t, resp, &history)

		foundVersionEvent := false
		for _, event := range history.Events {
			if event.EventType == "governance.version.created" {
				foundVersionEvent = true
				if event.VersionID == "" {
					t.Error("version.created audit event has empty versionId")
				}
				break
			}
		}
		if !foundVersionEvent {
			t.Error("audit history does not contain governance.version.created event")
		}
	})

	t.Run("audit_pagination", func(t *testing.T) {
		// Request with pageSize=1 to test pagination.
		url := fmt.Sprintf("%s?pageSize=1", governanceHistoryURL(plugin, kind, name))
		resp := doRequest(t, http.MethodGet, url, nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var page1 auditEventList
		decodeJSON(t, resp, &page1)

		if len(page1.Events) > 1 {
			t.Errorf("requested pageSize=1 but got %d events", len(page1.Events))
		}

		if page1.TotalSize > 1 && page1.NextPageToken == "" {
			t.Log("note: totalSize > 1 but no nextPageToken (pagination may not be fully implemented)")
		}

		// If there is a next page, fetch it.
		if page1.NextPageToken != "" {
			url2 := fmt.Sprintf("%s?pageSize=1&pageToken=%s", governanceHistoryURL(plugin, kind, name), page1.NextPageToken)
			resp2 := doRequest(t, http.MethodGet, url2, nil, defaultHeaders())
			requireStatus(t, resp2, http.StatusOK)

			var page2 auditEventList
			decodeJSON(t, resp2, &page2)

			if len(page2.Events) > 1 {
				t.Errorf("page 2: requested pageSize=1 but got %d events", len(page2.Events))
			}

			// Ensure events are different (no overlap).
			if len(page1.Events) > 0 && len(page2.Events) > 0 {
				if page1.Events[0].ID == page2.Events[0].ID {
					t.Error("page 1 and page 2 returned the same event")
				}
			}
		}
	})

	t.Run("audit_events_ordered_newest_first", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, governanceHistoryURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var history auditEventList
		decodeJSON(t, resp, &history)

		if len(history.Events) < 2 {
			t.Skip("need at least 2 audit events to verify ordering")
		}

		// Verify events are ordered newest-first (createdAt descending).
		for i := 0; i < len(history.Events)-1; i++ {
			t1, err1 := time.Parse(time.RFC3339, history.Events[i].CreatedAt)
			t2, err2 := time.Parse(time.RFC3339, history.Events[i+1].CreatedAt)
			if err1 != nil || err2 != nil {
				t.Logf("cannot parse timestamps at index %d/%d: %v, %v", i, i+1, err1, err2)
				continue
			}
			if t1.Before(t2) {
				t.Errorf("events not in newest-first order: event[%d].createdAt=%s < event[%d].createdAt=%s",
					i, history.Events[i].CreatedAt, i+1, history.Events[i+1].CreatedAt)
			}
		}
	})
}
