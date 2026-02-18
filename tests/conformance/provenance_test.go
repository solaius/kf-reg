package conformance

import (
	"fmt"
	"net/http"
	"testing"
)

// TestGovernanceProvenance tests that version responses include provenance information
// when available.
func TestGovernanceProvenance(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)
	skipIfNoGovernance(t)

	for _, tp := range governanceTestPlugins() {
		plugin, kind := tp.Plugin, tp.Kind
		t.Run(plugin, func(t *testing.T) {
			testGovernanceProvenance(t, plugin, kind)
		})
	}
}

func testGovernanceProvenance(t *testing.T, plugin, kind string) {
	name := fmt.Sprintf("provenance-test-%s", testSeqNum())

	// Create governance record.
	resp := doRequest(t, http.MethodGet, governanceURL(plugin, kind, name), nil, defaultHeaders())
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Create a version.
	versionBody := map[string]any{
		"versionLabel": "v1.0",
		"reason":       "provenance test",
	}
	resp = doRequest(t, http.MethodPost, governanceVersionsURL(plugin, kind, name), versionBody, defaultHeaders())
	requireStatus(t, resp, http.StatusCreated)

	var version versionResponse
	decodeJSON(t, resp, &version)

	t.Run("version_has_standard_fields", func(t *testing.T) {
		if version.VersionID == "" {
			t.Error("expected non-empty versionId")
		}
		if version.VersionLabel != "v1.0" {
			t.Errorf("expected versionLabel=v1.0, got %s", version.VersionLabel)
		}
		if version.CreatedAt == "" {
			t.Error("expected non-empty createdAt")
		}
		if version.CreatedBy == "" {
			t.Error("expected non-empty createdBy")
		}
	})

	t.Run("version_provenance_may_be_present", func(t *testing.T) {
		// Provenance fields are optional -- they may be populated by the catalog server
		// when the asset has source provenance data. For conformance, we only verify
		// the structure is valid when present.
		if version.Provenance != nil {
			prov := version.Provenance
			t.Logf("provenance present: sourceType=%s sourceUri=%s sourceId=%s",
				prov.SourceType, prov.SourceURI, prov.SourceID)

			if prov.Integrity != nil {
				t.Logf("integrity: verified=%v method=%s", prov.Integrity.Verified, prov.Integrity.Method)
			}
		} else {
			t.Log("provenance is nil (expected for manually created versions)")
		}
	})

	t.Run("version_detail_from_list", func(t *testing.T) {
		// List versions and verify structure matches.
		resp := doRequest(t, http.MethodGet, governanceVersionsURL(plugin, kind, name), nil, defaultHeaders())
		requireStatus(t, resp, http.StatusOK)

		var list versionListResponse
		decodeJSON(t, resp, &list)

		if list.TotalSize < 1 {
			t.Fatalf("expected at least 1 version, got totalSize=%d", list.TotalSize)
		}

		found := false
		for _, v := range list.Versions {
			if v.VersionID == version.VersionID {
				found = true
				if v.VersionLabel != version.VersionLabel {
					t.Errorf("version label mismatch in list: got %s, want %s", v.VersionLabel, version.VersionLabel)
				}
				// If provenance is populated in the create response, it should also appear in list.
				if version.Provenance != nil && v.Provenance == nil {
					t.Log("note: provenance present in create response but not in list")
				}
				break
			}
		}
		if !found {
			t.Errorf("version %s not found in list", version.VersionID)
		}
	})
}
