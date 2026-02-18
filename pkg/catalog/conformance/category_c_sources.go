package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func runCategorySources(t *testing.T, serverURL string, p PluginInfo) CategoryResult {
	t.Helper()

	cat := CategoryResult{Name: "sources"}

	record := func(name, status, msg string) {
		cat.Tests = append(cat.Tests, TestResult{Name: name, Status: status, Message: msg})
		switch status {
		case "passed":
			cat.Passed++
		case "failed":
			cat.Failed++
		case "skipped":
			cat.Skipped++
		}
	}

	testPrefix := fmt.Sprintf("%s.sources", p.Name)

	// Check if plugin has source management capability.
	if p.CapabilitiesV2 == nil || p.CapabilitiesV2.Sources == nil {
		record(testPrefix+".present", "skipped", "no source capabilities declared")
		return cat
	}

	if !p.CapabilitiesV2.Sources.Manageable {
		record(testPrefix+".manageable", "skipped", "sources not manageable")
		return cat
	}

	record(testPrefix+".manageable", "passed", "")

	// Test sources listing via management endpoint.
	sourcesURL := fmt.Sprintf("%s/sources", p.BasePath)
	t.Run("list_sources", func(t *testing.T) {
		resp, err := http.Get(serverURL + sourcesURL)
		if err != nil {
			t.Fatalf("GET %s failed: %v", sourcesURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			record(testPrefix+".list", "skipped", "sources endpoint not available")
			t.Skipf("GET %s returned 404", sourcesURL)
			return
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("GET %s returned %d: %s", sourcesURL, resp.StatusCode, string(body))
			record(testPrefix+".list", "failed", fmt.Sprintf("status %d", resp.StatusCode))
			return
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("sources response is not valid JSON: %v", err)
		}

		// Should have sources array.
		if sources, ok := result["sources"]; ok {
			if arr, isArr := sources.([]any); isArr {
				record(testPrefix+".list", "passed", fmt.Sprintf("%d sources", len(arr)))
			} else {
				t.Error("'sources' field is not an array")
				record(testPrefix+".list", "failed", "sources not array")
			}
		} else {
			record(testPrefix+".list", "passed", "response has no sources key")
		}
	})

	// Check if refresh is supported.
	if p.CapabilitiesV2.Sources.Refreshable {
		record(testPrefix+".refreshable", "passed", "refresh supported")
	} else {
		record(testPrefix+".refreshable", "skipped", "refresh not supported")
	}

	return cat
}
