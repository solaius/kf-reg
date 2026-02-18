package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// --- Governance conformance types ---

type governanceResponse struct {
	AssetRef   assetRef           `json:"assetRef"`
	Governance governanceOverlay  `json:"governance"`
}

type assetRef struct {
	Plugin string `json:"plugin"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
}

type governanceOverlay struct {
	Owner       *ownerInfo      `json:"owner,omitempty"`
	Team        *teamInfo       `json:"team,omitempty"`
	SLA         *slaInfo        `json:"sla,omitempty"`
	Risk        *riskInfo       `json:"risk,omitempty"`
	IntendedUse *intendedUse    `json:"intendedUse,omitempty"`
	Compliance  *complianceInfo `json:"compliance,omitempty"`
	Lifecycle   *lifecycleInfo  `json:"lifecycle,omitempty"`
	Audit       *auditMetadata  `json:"audit,omitempty"`
}

type ownerInfo struct {
	Principal   string `json:"principal,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
}

type teamInfo struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

type slaInfo struct {
	Tier          string `json:"tier,omitempty"`
	ResponseHours int    `json:"responseHours,omitempty"`
}

type riskInfo struct {
	Level      string   `json:"level,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type intendedUse struct {
	Summary      string   `json:"summary,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Restrictions []string `json:"restrictions,omitempty"`
}

type complianceInfo struct {
	Tags     []string `json:"tags,omitempty"`
	Controls []string `json:"controls,omitempty"`
}

type lifecycleInfo struct {
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
	ChangedBy string `json:"changedBy,omitempty"`
	ChangedAt string `json:"changedAt,omitempty"`
}

type auditMetadata struct {
	LastReviewedAt    string `json:"lastReviewedAt,omitempty"`
	ReviewCadenceDays int    `json:"reviewCadenceDays,omitempty"`
}

type actionResult struct {
	Action  string         `json:"action"`
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

type auditEventList struct {
	Events        []auditEvent `json:"events"`
	NextPageToken string       `json:"nextPageToken,omitempty"`
	TotalSize     int          `json:"totalSize"`
}

type auditEvent struct {
	ID            string         `json:"id"`
	CorrelationID string         `json:"correlationId"`
	EventType     string         `json:"eventType"`
	Actor         string         `json:"actor"`
	AssetUID      string         `json:"assetUid"`
	VersionID     string         `json:"versionId,omitempty"`
	Action        string         `json:"action,omitempty"`
	Outcome       string         `json:"outcome"`
	Reason        string         `json:"reason,omitempty"`
	OldValue      map[string]any `json:"oldValue,omitempty"`
	NewValue      map[string]any `json:"newValue,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     string         `json:"createdAt"`
}

type versionResponse struct {
	VersionID     string          `json:"versionId"`
	VersionLabel  string          `json:"versionLabel"`
	CreatedAt     string          `json:"createdAt"`
	CreatedBy     string          `json:"createdBy"`
	ContentDigest string          `json:"contentDigest,omitempty"`
	Provenance    *provenanceInfo `json:"provenance,omitempty"`
}

type provenanceInfo struct {
	SourceType string         `json:"sourceType,omitempty"`
	SourceURI  string         `json:"sourceUri,omitempty"`
	SourceID   string         `json:"sourceId,omitempty"`
	RevisionID string         `json:"revisionId,omitempty"`
	ObservedAt string         `json:"observedAt,omitempty"`
	Integrity  *integrityInfo `json:"integrity,omitempty"`
}

type integrityInfo struct {
	Verified bool   `json:"verified"`
	Method   string `json:"method,omitempty"`
	Details  string `json:"details,omitempty"`
}

type versionListResponse struct {
	Versions      []versionResponse `json:"versions"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
	TotalSize     int               `json:"totalSize"`
}

type bindingResponse struct {
	Environment       string `json:"environment"`
	VersionID         string `json:"versionId"`
	BoundAt           string `json:"boundAt"`
	BoundBy           string `json:"boundBy"`
	PreviousVersionID string `json:"previousVersionId,omitempty"`
}

type bindingsResponse struct {
	Bindings []bindingResponse `json:"bindings"`
}

type approvalRequest struct {
	ID             string             `json:"id"`
	AssetRef       assetRef           `json:"assetRef"`
	Action         string             `json:"action"`
	ActionParams   map[string]any     `json:"actionParams,omitempty"`
	PolicyID       string             `json:"policyId"`
	RequiredCount  int                `json:"requiredCount"`
	Status         string             `json:"status"`
	Requester      string             `json:"requester"`
	Reason         string             `json:"reason,omitempty"`
	Decisions      []approvalDecision `json:"decisions,omitempty"`
	ResolvedAt     string             `json:"resolvedAt,omitempty"`
	ResolvedBy     string             `json:"resolvedBy,omitempty"`
	ResolutionNote string             `json:"resolutionNote,omitempty"`
	ExpiresAt      string             `json:"expiresAt,omitempty"`
	CreatedAt      string             `json:"createdAt"`
}

type approvalDecision struct {
	ID        string `json:"id"`
	RequestID string `json:"requestId"`
	Reviewer  string `json:"reviewer"`
	Verdict   string `json:"verdict"`
	Comment   string `json:"comment,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type approvalRequestList struct {
	Requests      []approvalRequest `json:"requests"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
	TotalSize     int               `json:"totalSize"`
}

// --- Governance test helpers ---

const govBasePath = "/api/governance/v1alpha1"

// governanceURL builds a governance asset URL.
func governanceURL(plugin, kind, name string) string {
	return fmt.Sprintf("%s%s/assets/%s/%s/%s", serverURL, govBasePath, plugin, kind, name)
}

// governanceActionURL builds a governance action URL.
func governanceActionURL(plugin, kind, name, action string) string {
	return fmt.Sprintf("%s/actions/%s", governanceURL(plugin, kind, name), action)
}

// governanceVersionsURL builds a versions URL.
func governanceVersionsURL(plugin, kind, name string) string {
	return fmt.Sprintf("%s/versions", governanceURL(plugin, kind, name))
}

// governanceBindingsURL builds a bindings URL.
func governanceBindingsURL(plugin, kind, name string) string {
	return fmt.Sprintf("%s/bindings", governanceURL(plugin, kind, name))
}

// governanceBindingEnvURL builds a binding environment URL.
func governanceBindingEnvURL(plugin, kind, name, env string) string {
	return fmt.Sprintf("%s/bindings/%s", governanceURL(plugin, kind, name), env)
}

// governanceHistoryURL builds an audit history URL.
func governanceHistoryURL(plugin, kind, name string) string {
	return fmt.Sprintf("%s/history", governanceURL(plugin, kind, name))
}

// doRequest makes an HTTP request with headers and optional body.
func doRequest(t *testing.T, method, url string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	return resp
}

// requireStatus checks the HTTP response status and fatally fails with the body on mismatch.
func requireStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// decodeJSON decodes the response body into v.
func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

// defaultHeaders returns standard governance test headers.
func defaultHeaders() map[string]string {
	return map[string]string{
		"X-User-Principal": "conformance-test",
	}
}

// --- Governance CRUD tests ---

// TestGovernanceCRUD tests the governance GET and PATCH endpoints.
func TestGovernanceCRUD(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		plugin, kind := tp.Plugin, tp.Kind
		t.Run(plugin, func(t *testing.T) {
			testGovernanceCRUD(t, plugin, kind)
		})
	}
}

func testGovernanceCRUD(t *testing.T, plugin, kind string) {
	name := fmt.Sprintf("gov-crud-%s", testSeqNum())

	t.Run("get_default", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var gov governanceResponse
		decodeJSON(t, resp, &gov)

		if gov.AssetRef.Plugin != plugin {
			t.Errorf("expected plugin=%s, got %s", plugin, gov.AssetRef.Plugin)
		}
		if gov.AssetRef.Kind != kind {
			t.Errorf("expected kind=%s, got %s", kind, gov.AssetRef.Kind)
		}
		if gov.AssetRef.Name != name {
			t.Errorf("expected name=%s, got %s", name, gov.AssetRef.Name)
		}

		// Default lifecycle state should be draft.
		if gov.Governance.Lifecycle == nil {
			t.Fatal("lifecycle is nil on default governance")
		}
		if gov.Governance.Lifecycle.State != "draft" {
			t.Errorf("expected default lifecycle state=draft, got %s", gov.Governance.Lifecycle.State)
		}

		// Default risk should be medium.
		if gov.Governance.Risk == nil {
			t.Fatal("risk is nil on default governance")
		}
		if gov.Governance.Risk.Level != "medium" {
			t.Errorf("expected default risk level=medium, got %s", gov.Governance.Risk.Level)
		}
	})

	t.Run("patch_governance", func(t *testing.T) {
		patch := governanceOverlay{
			Owner: &ownerInfo{
				Principal:   "alice@example.com",
				DisplayName: "Alice",
				Email:       "alice@example.com",
			},
			Team: &teamInfo{
				Name: "ml-platform",
				ID:   "team-42",
			},
			SLA: &slaInfo{
				Tier:          "gold",
				ResponseHours: 4,
			},
			Risk: &riskInfo{
				Level:      "high",
				Categories: []string{"pii", "bias"},
			},
			Compliance: &complianceInfo{
				Tags:     []string{"sox", "gdpr"},
				Controls: []string{"AC-1", "AC-2"},
			},
		}

		resp := doRequest(t, http.MethodPatch, governanceURL(plugin, kind, name), patch, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var gov governanceResponse
		decodeJSON(t, resp, &gov)

		// Verify updated fields.
		if gov.Governance.Owner == nil || gov.Governance.Owner.Principal != "alice@example.com" {
			t.Errorf("expected owner principal alice@example.com, got %v", gov.Governance.Owner)
		}
		if gov.Governance.Team == nil || gov.Governance.Team.Name != "ml-platform" {
			t.Errorf("expected team name ml-platform, got %v", gov.Governance.Team)
		}
		if gov.Governance.SLA == nil || gov.Governance.SLA.Tier != "gold" {
			t.Errorf("expected SLA tier gold, got %v", gov.Governance.SLA)
		}
		if gov.Governance.Risk == nil || gov.Governance.Risk.Level != "high" {
			t.Errorf("expected risk level high, got %v", gov.Governance.Risk)
		}
		if gov.Governance.Compliance == nil || len(gov.Governance.Compliance.Tags) != 2 {
			t.Errorf("expected 2 compliance tags, got %v", gov.Governance.Compliance)
		}
	})

	t.Run("get_after_patch", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var gov governanceResponse
		decodeJSON(t, resp, &gov)

		if gov.Governance.Owner == nil || gov.Governance.Owner.Principal != "alice@example.com" {
			t.Errorf("GET after PATCH: expected owner principal alice@example.com, got %v", gov.Governance.Owner)
		}
		if gov.Governance.Risk == nil || gov.Governance.Risk.Level != "high" {
			t.Errorf("GET after PATCH: expected risk level high, got %v", gov.Governance.Risk)
		}
		if gov.Governance.Team == nil || gov.Governance.Team.Name != "ml-platform" {
			t.Errorf("GET after PATCH: expected team name ml-platform, got %v", gov.Governance.Team)
		}
	})

	t.Run("patch_is_additive", func(t *testing.T) {
		// Patch only the SLA field; other fields should remain unchanged.
		patch := governanceOverlay{
			SLA: &slaInfo{
				Tier:          "silver",
				ResponseHours: 8,
			},
		}

		resp := doRequest(t, http.MethodPatch, governanceURL(plugin, kind, name), patch, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var gov governanceResponse
		decodeJSON(t, resp, &gov)

		// SLA should be updated.
		if gov.Governance.SLA == nil || gov.Governance.SLA.Tier != "silver" {
			t.Errorf("expected SLA tier silver, got %v", gov.Governance.SLA)
		}

		// Owner should still be alice from the previous patch.
		if gov.Governance.Owner == nil || gov.Governance.Owner.Principal != "alice@example.com" {
			t.Errorf("additive patch: expected owner principal alice@example.com to persist, got %v", gov.Governance.Owner)
		}

		// Risk should still be high.
		if gov.Governance.Risk == nil || gov.Governance.Risk.Level != "high" {
			t.Errorf("additive patch: expected risk level high to persist, got %v", gov.Governance.Risk)
		}
	})
}
