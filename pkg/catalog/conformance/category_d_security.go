package conformance

import (
	"fmt"
	"testing"
)

func runCategorySecurity(t *testing.T, serverURL string, p PluginInfo) CategoryResult {
	t.Helper()

	cat := CategoryResult{Name: "security"}

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

	testPrefix := fmt.Sprintf("%s.security", p.Name)

	// Basic health endpoint check (does not require auth).
	t.Run("health_accessible", func(t *testing.T) {
		var result map[string]any
		GetJSON(t, serverURL, "/healthz", &result)

		status, _ := result["status"].(string)
		if status == "" {
			t.Error("healthz response missing status field")
			record(testPrefix+".healthz", "failed", "no status")
		} else {
			record(testPrefix+".healthz", "passed", "")
		}
	})

	// RBAC and tenancy tests require special server configuration
	// (auth providers, namespace headers, etc.) and are skipped by default.
	record(testPrefix+".rbac", "skipped", "RBAC validation requires auth-enabled server configuration")
	record(testPrefix+".tenancy", "skipped", "tenancy validation requires multi-tenant server configuration")

	return cat
}
