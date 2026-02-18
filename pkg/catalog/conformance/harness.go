package conformance

import (
	"testing"
	"time"
)

// RunConformance runs the full conformance suite against a catalog server.
// It returns a ConformanceResult with detailed pass/fail/skip information
// for each test category.
func RunConformance(t *testing.T, cfg HarnessConfig) ConformanceResult {
	t.Helper()

	result := ConformanceResult{
		Timestamp:  time.Now(),
		ServerURL:  cfg.ServerURL,
		PluginName: cfg.PluginName,
	}

	WaitForReady(t, cfg.ServerURL)

	// Discover plugins.
	var response PluginsResponse
	GetJSON(t, cfg.ServerURL, "/api/plugins", &response)

	if response.Count == 0 {
		t.Fatal("no plugins found")
	}

	// Filter to specific plugin if configured.
	plugins := response.Plugins
	if cfg.PluginName != "" {
		var filtered []PluginInfo
		for _, p := range plugins {
			if p.Name == cfg.PluginName {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			t.Fatalf("plugin %q not found", cfg.PluginName)
		}
		plugins = filtered
	}

	shouldSkip := func(category string) bool {
		for _, s := range cfg.SkipCategories {
			if s == category {
				return true
			}
		}
		return false
	}

	for _, p := range plugins {
		p := p // capture range variable
		t.Run(p.Name, func(t *testing.T) {
			// Category A: Capabilities
			if !shouldSkip("capabilities") {
				t.Run("A_capabilities", func(t *testing.T) {
					catResult := runCategoryCapabilities(t, cfg.ServerURL, p)
					result.Categories = append(result.Categories, catResult)
				})
			}

			// Category B: List/Get endpoints, filters, pagination
			if !shouldSkip("list_get") {
				t.Run("B_list_get", func(t *testing.T) {
					catResult := runCategoryListGet(t, cfg.ServerURL, p)
					result.Categories = append(result.Categories, catResult)
				})
			}

			// Category C: Sources
			if !shouldSkip("sources") {
				t.Run("C_sources", func(t *testing.T) {
					catResult := runCategorySources(t, cfg.ServerURL, p)
					result.Categories = append(result.Categories, catResult)
				})
			}

			// Category D: Security
			if !shouldSkip("security") {
				t.Run("D_security", func(t *testing.T) {
					catResult := runCategorySecurity(t, cfg.ServerURL, p)
					result.Categories = append(result.Categories, catResult)
				})
			}

			// Category E: Observability
			if !shouldSkip("observability") {
				t.Run("E_observability", func(t *testing.T) {
					catResult := runCategoryObservability(t, cfg.ServerURL, p)
					result.Categories = append(result.Categories, catResult)
				})
			}

			// Category F: OpenAPI
			if !shouldSkip("openapi") {
				t.Run("F_openapi", func(t *testing.T) {
					catResult := runCategoryOpenAPI(t, cfg.ServerURL, p)
					result.Categories = append(result.Categories, catResult)
				})
			}
		})
	}

	// Sum totals.
	for _, cat := range result.Categories {
		result.TotalPassed += cat.Passed
		result.TotalFailed += cat.Failed
		result.TotalSkipped += cat.Skipped
	}

	return result
}
