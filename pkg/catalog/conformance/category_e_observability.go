package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func runCategoryObservability(t *testing.T, serverURL string, p PluginInfo) CategoryResult {
	t.Helper()

	cat := CategoryResult{Name: "observability"}

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

	testPrefix := fmt.Sprintf("%s.observability", p.Name)

	// Check health endpoints exist.
	for _, path := range []string{"/healthz", "/livez", "/readyz"} {
		endpointName := path[1:] // strip leading slash
		t.Run(endpointName, func(t *testing.T) {
			resp, err := http.Get(serverURL + path)
			if err != nil {
				t.Fatalf("GET %s failed: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("GET %s returned %d: %s", path, resp.StatusCode, string(body))
				record(testPrefix+"."+endpointName, "failed", fmt.Sprintf("status %d", resp.StatusCode))
				return
			}

			var result map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			status, _ := result["status"].(string)
			if status == "" {
				t.Error("response missing 'status' field")
				record(testPrefix+"."+endpointName, "failed", "no status field")
			} else {
				record(testPrefix+"."+endpointName, "passed", "")
			}
		})
	}

	// Check readyz components.
	t.Run("readyz_components", func(t *testing.T) {
		var result map[string]any
		GetJSON(t, serverURL, "/readyz", &result)

		components, ok := result["components"].(map[string]any)
		if !ok {
			t.Error("readyz response missing 'components' object")
			record(testPrefix+".readyz_components", "failed", "no components object")
			return
		}

		for _, key := range []string{"database", "initial_load", "plugins"} {
			comp, ok := components[key].(map[string]any)
			if !ok {
				t.Errorf("readyz missing component %q", key)
				record(testPrefix+".readyz."+key, "failed", "missing")
				continue
			}
			status, _ := comp["status"].(string)
			if status == "" {
				t.Errorf("component %q has no status", key)
				record(testPrefix+".readyz."+key, "failed", "no status")
			} else {
				record(testPrefix+".readyz."+key, "passed", "")
			}
		}
	})

	return cat
}
