package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GovernanceReport holds the results of governance checks.
type GovernanceReport struct {
	PluginDir string            `json:"pluginDir"`
	Passed    bool              `json:"passed"`
	Checks    []GovernanceCheck `json:"checks"`
}

// GovernanceCheck represents a single governance check result.
type GovernanceCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// RunGovernanceChecks validates a plugin directory meets governance requirements
// for the "supported plugin" designation.
func RunGovernanceChecks(pluginDir string) GovernanceReport {
	report := GovernanceReport{
		PluginDir: pluginDir,
		Passed:    true,
	}

	// Check 1: plugin.yaml exists and is valid
	report.addCheck(checkPluginYAML(pluginDir))

	// Check 2: catalog.yaml exists
	report.addCheck(checkCatalogYAML(pluginDir))

	// Check 3: Compatibility fields present
	report.addCheck(checkCompatibility(pluginDir))

	// Check 4: Ownership declared
	report.addCheck(checkOwnership(pluginDir))

	// Check 5: License present
	report.addCheck(checkLicense(pluginDir))

	// Check 6: Conformance tests exist
	report.addCheck(checkConformanceTests(pluginDir))

	// Check 7: Docs present
	report.addCheck(checkDocs(pluginDir))

	return report
}

func (r *GovernanceReport) addCheck(check GovernanceCheck) {
	r.Checks = append(r.Checks, check)
	if !check.Passed {
		r.Passed = false
	}
}

// checkPluginYAML loads plugin.yaml and validates it with ValidatePluginMetadata.
func checkPluginYAML(pluginDir string) GovernanceCheck {
	path := filepath.Join(pluginDir, "plugin.yaml")
	spec, err := LoadPluginMetadata(path)
	if err != nil {
		return GovernanceCheck{
			Name:    "plugin.yaml valid",
			Passed:  false,
			Message: err.Error(),
		}
	}

	errs := ValidatePluginMetadata(spec)
	if len(errs) > 0 {
		return GovernanceCheck{
			Name:    "plugin.yaml valid",
			Passed:  false,
			Message: strings.Join(errs, "; "),
		}
	}

	return GovernanceCheck{
		Name:   "plugin.yaml valid",
		Passed: true,
	}
}

// checkCatalogYAML checks that catalog.yaml exists in the plugin directory.
func checkCatalogYAML(pluginDir string) GovernanceCheck {
	path := filepath.Join(pluginDir, "catalog.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return GovernanceCheck{
			Name:    "catalog.yaml exists",
			Passed:  false,
			Message: "catalog.yaml not found",
		}
	}
	return GovernanceCheck{
		Name:   "catalog.yaml exists",
		Passed: true,
	}
}

// checkCompatibility checks that plugin.yaml has compatibility.catalogServer.minVersion
// and compatibility.frameworkApi fields.
func checkCompatibility(pluginDir string) GovernanceCheck {
	path := filepath.Join(pluginDir, "plugin.yaml")
	spec, err := LoadPluginMetadata(path)
	if err != nil {
		return GovernanceCheck{
			Name:    "compatibility fields present",
			Passed:  false,
			Message: fmt.Sprintf("cannot load plugin.yaml: %v", err),
		}
	}

	var missing []string
	if spec.Spec.Compatibility.CatalogServer.MinVersion == "" {
		missing = append(missing, "compatibility.catalogServer.minVersion")
	}
	if spec.Spec.Compatibility.FrameworkAPI == "" {
		missing = append(missing, "compatibility.frameworkApi")
	}

	if len(missing) > 0 {
		return GovernanceCheck{
			Name:    "compatibility fields present",
			Passed:  false,
			Message: fmt.Sprintf("missing: %s", strings.Join(missing, ", ")),
		}
	}

	return GovernanceCheck{
		Name:   "compatibility fields present",
		Passed: true,
	}
}

// checkOwnership checks that plugin.yaml has at least one owner with a team name.
func checkOwnership(pluginDir string) GovernanceCheck {
	path := filepath.Join(pluginDir, "plugin.yaml")
	spec, err := LoadPluginMetadata(path)
	if err != nil {
		return GovernanceCheck{
			Name:    "ownership declared",
			Passed:  false,
			Message: fmt.Sprintf("cannot load plugin.yaml: %v", err),
		}
	}

	if len(spec.Spec.Owners) == 0 {
		return GovernanceCheck{
			Name:    "ownership declared",
			Passed:  false,
			Message: "no owners declared in plugin.yaml",
		}
	}

	for i, owner := range spec.Spec.Owners {
		if owner.Team == "" {
			return GovernanceCheck{
				Name:    "ownership declared",
				Passed:  false,
				Message: fmt.Sprintf("owners[%d].team is empty", i),
			}
		}
	}

	return GovernanceCheck{
		Name:   "ownership declared",
		Passed: true,
	}
}

// checkLicense checks that plugin.yaml has a license field OR a LICENSE file exists
// in the plugin directory.
func checkLicense(pluginDir string) GovernanceCheck {
	// First check plugin.yaml for license field
	path := filepath.Join(pluginDir, "plugin.yaml")
	spec, err := LoadPluginMetadata(path)
	if err == nil && spec.Spec.License != "" {
		return GovernanceCheck{
			Name:   "license present",
			Passed: true,
		}
	}

	// Fall back to checking for a LICENSE file
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt"} {
		if _, err := os.Stat(filepath.Join(pluginDir, name)); err == nil {
			return GovernanceCheck{
				Name:   "license present",
				Passed: true,
			}
		}
	}

	return GovernanceCheck{
		Name:    "license present",
		Passed:  false,
		Message: "no license field in plugin.yaml and no LICENSE file found",
	}
}

// checkConformanceTests checks for conformance test files. It looks for a
// tests/conformance/ directory with *_test.go files, or any *_test.go file
// containing "conformance" or "Conformance" in its name.
func checkConformanceTests(pluginDir string) GovernanceCheck {
	// Check tests/conformance/ directory
	conformanceDir := filepath.Join(pluginDir, "tests", "conformance")
	entries, err := os.ReadDir(conformanceDir)
	if err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), "_test.go") {
				return GovernanceCheck{
					Name:   "conformance tests exist",
					Passed: true,
				}
			}
		}
	}

	// Walk top-level for any *conformance*_test.go
	entries, err = os.ReadDir(pluginDir)
	if err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if strings.Contains(name, "conformance") && strings.HasSuffix(name, "_test.go") {
				return GovernanceCheck{
					Name:   "conformance tests exist",
					Passed: true,
				}
			}
		}
	}

	return GovernanceCheck{
		Name:    "conformance tests exist",
		Passed:  false,
		Message: "no conformance test files found",
	}
}

// checkDocs checks that a README.md file exists in the plugin directory.
func checkDocs(pluginDir string) GovernanceCheck {
	readmePath := filepath.Join(pluginDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		return GovernanceCheck{
			Name:    "documentation present",
			Passed:  false,
			Message: "README.md not found",
		}
	}
	return GovernanceCheck{
		Name:   "documentation present",
		Passed: true,
	}
}
