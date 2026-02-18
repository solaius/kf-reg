package conformance

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ConformanceResult holds the results of a conformance run.
type ConformanceResult struct {
	Timestamp    time.Time        `json:"timestamp"`
	ServerURL    string           `json:"serverURL"`
	PluginName   string           `json:"pluginName,omitempty"`
	Categories   []CategoryResult `json:"categories"`
	TotalPassed  int              `json:"totalPassed"`
	TotalFailed  int              `json:"totalFailed"`
	TotalSkipped int              `json:"totalSkipped"`
}

// CategoryResult holds results for a single test category.
type CategoryResult struct {
	Name    string       `json:"name"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Skipped int          `json:"skipped"`
	Tests   []TestResult `json:"tests"`
}

// TestResult holds the result of a single test.
type TestResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "passed", "failed", "skipped"
	Message string `json:"message,omitempty"`
}

// ToJSON returns the result as formatted JSON.
func (r *ConformanceResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Summary returns a human-readable summary.
func (r *ConformanceResult) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Conformance Report -- %s\n", r.Timestamp.Format(time.RFC3339)))
	if r.PluginName != "" {
		sb.WriteString(fmt.Sprintf("Plugin: %s\n", r.PluginName))
	}
	sb.WriteString(fmt.Sprintf("Server: %s\n", r.ServerURL))
	sb.WriteString(fmt.Sprintf("Total: %d passed, %d failed, %d skipped\n\n",
		r.TotalPassed, r.TotalFailed, r.TotalSkipped))

	for _, cat := range r.Categories {
		sb.WriteString(fmt.Sprintf("  [%s] %d passed, %d failed, %d skipped\n",
			cat.Name, cat.Passed, cat.Failed, cat.Skipped))
		for _, test := range cat.Tests {
			icon := "PASS"
			if test.Status == "failed" {
				icon = "FAIL"
			} else if test.Status == "skipped" {
				icon = "SKIP"
			}
			sb.WriteString(fmt.Sprintf("    [%s] %s", icon, test.Name))
			if test.Message != "" {
				sb.WriteString(fmt.Sprintf(" -- %s", test.Message))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
