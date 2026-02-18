package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// docsFiles lists the docs to generate with their template and output filename.
var docsFiles = []struct {
	tmpl     string
	filename string
}{
	{TmplDocsReadme, "README.md"},
	{TmplDocsProviderGuide, "provider-guide.md"},
	{TmplDocsSchemaGuide, "schema-guide.md"},
	{TmplDocsTesting, "testing.md"},
	{TmplDocsPublishing, "publishing.md"},
}

// generateDocsKit generates the documentation kit for a plugin.
// Files are created once and can be edited. Existing files are not overwritten.
func generateDocsKit(config CatalogConfig) error {
	docsDir := "docs"
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	entityName := config.Spec.Entity.Name
	packageName := filepath.Base(config.Metadata.Name)

	// Build provider list
	var providers []string
	for _, p := range config.Spec.Providers {
		providers = append(providers, p.Type)
	}
	if len(providers) == 0 {
		providers = []string{"yaml"}
	}

	data := map[string]any{
		"Name":            packageName,
		"DisplayName":     capitalize(packageName),
		"EntityName":      entityName,
		"EntityNameLower": strings.ToLower(entityName),
		"Package":         config.Spec.Package,
		"BasePath":        config.Spec.API.BasePath,
		"Providers":       providers,
	}

	createdCount := 0
	skippedCount := 0

	for _, df := range docsFiles {
		outputPath := filepath.Join(docsDir, df.filename)

		// Don't overwrite existing docs
		if _, err := os.Stat(outputPath); err == nil {
			skippedCount++
			continue
		}

		if err := executeTemplate(df.tmpl, outputPath, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", df.filename, err)
		}
		createdCount++
	}

	if createdCount > 0 {
		fmt.Printf("  Created: docs/ (%d files)\n", createdCount)
	}
	if skippedCount > 0 {
		fmt.Printf("  Skipped: docs/ (%d files already exist)\n", skippedCount)
	}

	return nil
}

// docsKitComplete checks if the docs directory has all required doc files.
func docsKitComplete(dir string) []string {
	var missing []string

	requiredDocs := []string{
		"README.md",
		"provider-guide.md",
		"schema-guide.md",
		"testing.md",
		"publishing.md",
	}

	docsDir := filepath.Join(dir, "docs")
	for _, doc := range requiredDocs {
		path := filepath.Join(docsDir, doc)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, fmt.Sprintf("docs/%s is missing", doc))
		}
	}

	return missing
}
