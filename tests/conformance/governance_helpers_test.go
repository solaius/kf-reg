package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testSeq is an atomic counter for generating unique test asset names.
var testSeq int64

// testRunPrefix is a unique prefix for this test binary invocation to avoid
// name collisions with stale DB records from prior runs.
var testRunPrefix = fmt.Sprintf("%d", time.Now().UnixMilli()%100000)

// testSeqNum returns a unique sequence number for naming test assets.
// The returned value includes a per-run prefix to avoid DB collisions.
func testSeqNum() string {
	n := atomic.AddInt64(&testSeq, 1)
	return fmt.Sprintf("%s-%d", testRunPrefix, n)
}

// governanceAvailable caches whether the governance API is reachable.
var (
	governanceOnce      sync.Once
	governanceReachable bool
)

// governanceTestPlugin identifies a plugin+kind pair for governance tests.
type governanceTestPlugin struct {
	Plugin string
	Kind   string
}

// governanceTestPlugins returns the set of plugins to test governance against.
// Since governance is plugin-agnostic, testing with 2+ plugins proves it works
// across asset types.
func governanceTestPlugins() []governanceTestPlugin {
	return []governanceTestPlugin{
		{Plugin: "mcp", Kind: "mcpserver"},
		{Plugin: "agents", Kind: "Agent"},
	}
}

// ensureGovernanceAtState ensures an asset governance record exists and walks it
// through valid transitions to reach the target state. If any transition is
// gated by an approval policy (202 response), it auto-approves the request
// to allow the state machine to proceed.
func ensureGovernanceAtState(t *testing.T, plugin, kind, name, targetState string) {
	t.Helper()

	// First, GET governance to auto-create (starts at "draft").
	resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Walk the asset through valid transitions to reach the target state.
	transitions := map[string][]string{
		"draft":      {},
		"approved":   {"approved"},
		"deprecated": {"approved", "deprecated"},
		"archived":   {"approved", "deprecated", "archived"},
	}
	for _, intermediate := range transitions[targetState] {
		body := map[string]any{
			"params": map[string]any{
				"state":  intermediate,
				"reason": "setup for test",
			},
		}
		resp := doRequest(t, http.MethodPost, governanceActionURL(plugin, kind, name, "lifecycle.setState"), body, defaultHeaders())
		status := resp.StatusCode

		if status == http.StatusOK {
			resp.Body.Close()
			continue
		}

		if status == http.StatusAccepted {
			// Approval is required -- auto-approve the request.
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var result struct {
				Data struct {
					RequestID string `json:"requestId"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &result); err != nil || result.Data.RequestID == "" {
				t.Fatalf("setup: got 202 for %s but could not parse requestId: %s", intermediate, string(respBody))
			}

			// Submit approval decision.
			decisionURL := fmt.Sprintf("%s%s/approvals/%s/decisions", serverURL, govBasePath, result.Data.RequestID)
			decision := map[string]any{
				"reviewer": "conformance-auto-approver",
				"verdict":  "approve",
				"comment":  "auto-approved for conformance test setup",
			}
			approverHeaders := map[string]string{"X-User-Principal": "conformance-auto-approver"}
			decResp := doRequest(t, http.MethodPost, decisionURL, decision, approverHeaders)
			if decResp.StatusCode != http.StatusOK && decResp.StatusCode != http.StatusCreated {
				decBody, _ := io.ReadAll(decResp.Body)
				decResp.Body.Close()
				t.Fatalf("setup: failed to auto-approve request %s for state %s: %d %s",
					result.Data.RequestID, intermediate, decResp.StatusCode, string(decBody))
			}
			decResp.Body.Close()
			continue
		}

		resp.Body.Close()
		t.Fatalf("setup: failed to transition to %s, got status %d", intermediate, status)
	}
}

// skipIfNoGovernance skips the test if the governance API is not available.
// It probes the governance base path once and caches the result.
func skipIfNoGovernance(t *testing.T) {
	t.Helper()
	governanceOnce.Do(func() {
		probeURL := serverURL + govBasePath + "/assets/mcp/mcpserver/__governance_probe__"
		resp, err := http.Get(probeURL)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		// If we get anything other than 404 from the outer router (i.e., the
		// governance route is mounted), consider it available. The governance
		// handler itself returns 200 with default data even for unknown assets.
		governanceReachable = resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated
	})
	if !governanceReachable {
		t.Skip("governance API not available (routes not mounted)")
	}
}
