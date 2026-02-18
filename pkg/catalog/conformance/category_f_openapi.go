package conformance

import (
	"fmt"
	"testing"
)

func runCategoryOpenAPI(t *testing.T, serverURL string, p PluginInfo) CategoryResult {
	t.Helper()

	cat := CategoryResult{Name: "openapi"}

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

	testPrefix := fmt.Sprintf("%s.openapi", p.Name)

	// OpenAPI spec availability is optional for external plugins.
	// We record it as a skipped check with guidance.
	record(testPrefix+".spec", "skipped",
		"OpenAPI spec validation requires plugin to expose an OpenAPI document; test framework does not mandate this")

	// Check that plugin has well-formed base path that could host an OpenAPI spec.
	if p.BasePath == "" {
		record(testPrefix+".basePath", "failed", "plugin has no basePath")
	} else if len(p.BasePath) < 5 || p.BasePath[:5] != "/api/" {
		record(testPrefix+".basePath", "failed", fmt.Sprintf("basePath %q does not start with /api/", p.BasePath))
	} else {
		record(testPrefix+".basePath", "passed", "")
	}

	return cat
}
