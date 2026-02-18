package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newValidateCmd() *cobra.Command {
	var governanceFlag bool

	cmd := &cobra.Command{
		Use:   "validate [dir]",
		Short: "Validate a plugin directory meets baseline requirements",
		Long: `Validate that a plugin directory contains valid plugin.yaml and catalog.yaml
files and that all required fields are present and correct.

Checks performed:
  - plugin.yaml exists and is valid YAML
  - All required fields are present
  - Version follows semver
  - Compatibility fields follow rules (minVersion <= maxVersion)
  - Owners field is non-empty
  - catalog.yaml exists and is valid YAML
  - catalog.yaml has required fields

Use --governance to run full governance checks for the "supported plugin"
designation, which additionally checks compatibility fields, ownership,
license, conformance tests, and documentation.

Example:
  catalog-gen validate
  catalog-gen validate ./catalog/plugins/mcp
  catalog-gen validate --governance ./catalog/plugins/mcp`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			if governanceFlag {
				return runGovernance(dir)
			}
			return runValidate(dir)
		},
	}

	cmd.Flags().BoolVar(&governanceFlag, "governance", false, "Run full governance checks for supported plugin designation")

	return cmd
}

func runValidate(dir string) error {
	var allErrors []string
	passCount := 0
	failCount := 0

	record := func(check string, errs []string) {
		if len(errs) == 0 {
			fmt.Printf("  PASS  %s\n", check)
			passCount++
		} else {
			fmt.Printf("  FAIL  %s\n", check)
			failCount++
			for _, e := range errs {
				fmt.Printf("        - %s\n", e)
			}
			allErrors = append(allErrors, errs...)
		}
	}

	fmt.Printf("Validating plugin directory: %s\n\n", dir)

	// Check plugin.yaml
	pluginPath := filepath.Join(dir, "plugin.yaml")
	pluginSpec, pluginLoadErr := plugin.LoadPluginMetadata(pluginPath)

	if pluginLoadErr != nil {
		record("plugin.yaml exists and parses", []string{pluginLoadErr.Error()})
	} else {
		record("plugin.yaml exists and parses", nil)

		// Validate plugin.yaml fields
		validationErrs := plugin.ValidatePluginMetadata(pluginSpec)
		record("plugin.yaml fields valid", validationErrs)
	}

	// Check catalog.yaml
	catalogPath := filepath.Join(dir, "catalog.yaml")
	catalogErrs := validateCatalogYAML(catalogPath)
	record("catalog.yaml exists and is valid", catalogErrs)

	// Check conformance tests exist
	if conformanceTestExists(dir) {
		record("conformance tests exist", nil)
	} else {
		record("conformance tests exist", []string{"no conformance test files found in tests/conformance/"})
	}

	// Check docs kit completeness
	docsErrs := docsKitComplete(dir)
	record("docs kit complete", docsErrs)

	// Summary
	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("  %d passed, %d failed\n", passCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("validation failed with %d error(s)", len(allErrors))
	}

	fmt.Println("\nAll checks passed.")
	return nil
}

// runGovernance runs full governance checks and prints the report.
func runGovernance(dir string) error {
	fmt.Printf("Running governance checks: %s\n\n", dir)

	report := plugin.RunGovernanceChecks(dir)

	for _, check := range report.Checks {
		if check.Passed {
			fmt.Printf("  PASS  %s\n", check.Name)
		} else {
			fmt.Printf("  FAIL  %s\n", check.Name)
			if check.Message != "" {
				fmt.Printf("        - %s\n", check.Message)
			}
		}
	}

	passCount := 0
	failCount := 0
	for _, check := range report.Checks {
		if check.Passed {
			passCount++
		} else {
			failCount++
		}
	}

	fmt.Printf("\n--- Governance Summary ---\n")
	fmt.Printf("  %d passed, %d failed\n", passCount, failCount)

	if !report.Passed {
		return fmt.Errorf("governance checks failed (%d check(s) did not pass)", failCount)
	}

	fmt.Println("\nAll governance checks passed. Plugin meets supported designation requirements.")
	return nil
}

// validateCatalogYAML checks that catalog.yaml exists and has required fields.
func validateCatalogYAML(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{"catalog.yaml not found"}
		}
		return []string{fmt.Sprintf("failed to read catalog.yaml: %v", err)}
	}

	var config CatalogConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return []string{fmt.Sprintf("failed to parse catalog.yaml: %v", err)}
	}

	var errs []string
	if config.APIVersion == "" {
		errs = append(errs, "catalog.yaml: apiVersion is required")
	}
	if config.Kind == "" {
		errs = append(errs, "catalog.yaml: kind is required")
	}
	if config.Metadata.Name == "" {
		errs = append(errs, "catalog.yaml: metadata.name is required")
	}
	if config.Spec.Entity.Name == "" {
		errs = append(errs, "catalog.yaml: spec.entity.name is required")
	}
	if config.Spec.Package == "" {
		errs = append(errs, "catalog.yaml: spec.package is required")
	}

	return errs
}
