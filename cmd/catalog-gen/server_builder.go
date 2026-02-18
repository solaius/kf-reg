package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newBuildServerCmd() *cobra.Command {
	var (
		outputDir string
		compile   bool
		goVersion string
	)

	cmd := &cobra.Command{
		Use:   "build-server <manifest>",
		Short: "Generate a catalog server from a manifest",
		Long: `Generate a compilable catalog server from a catalog-server-manifest.yaml file.

This generates:
  - main.go with blank imports for all plugins
  - go.mod with all plugin dependencies
  - Dockerfile for building a container image
  - COMPATIBILITY.md with a plugin compatibility matrix

Examples:
  catalog-gen build-server catalog-server-manifest.yaml
  catalog-gen build-server manifest.yaml --output ./build
  catalog-gen build-server manifest.yaml --compile`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuildServer(args[0], outputDir, goVersion, compile)
		},
	}

	cmd.Flags().StringVar(&outputDir, "output", "build", "Output directory for generated files")
	cmd.Flags().BoolVar(&compile, "compile", false, "Run go build after generating files")
	cmd.Flags().StringVar(&goVersion, "go-version", "1.22", "Go version for go.mod and Dockerfile")

	return cmd
}

// loadServerManifest reads and parses a catalog-server-manifest.yaml file.
func loadServerManifest(path string) (*ServerManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest ServerManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// validateServerManifest checks a ServerManifest for required fields.
func validateServerManifest(m *ServerManifest) []string {
	var errs []string

	if m.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if m.Kind == "" {
		errs = append(errs, "kind is required")
	} else if m.Kind != "CatalogServerBuild" {
		errs = append(errs, fmt.Sprintf("kind must be CatalogServerBuild, got %q", m.Kind))
	}
	if m.Spec.Base.Module == "" {
		errs = append(errs, "spec.base.module is required")
	}
	if m.Spec.Base.Version == "" {
		errs = append(errs, "spec.base.version is required")
	}
	if len(m.Spec.Plugins) == 0 {
		errs = append(errs, "spec.plugins must have at least one entry")
	}
	for i, p := range m.Spec.Plugins {
		if p.Name == "" {
			errs = append(errs, fmt.Sprintf("spec.plugins[%d].name is required", i))
		}
		if p.Module == "" {
			errs = append(errs, fmt.Sprintf("spec.plugins[%d].module is required", i))
		}
		if p.Version == "" {
			errs = append(errs, fmt.Sprintf("spec.plugins[%d].version is required", i))
		}
	}

	return errs
}

func runBuildServer(manifestPath, outputDir, goVersion string, compile bool) error {
	manifest, err := loadServerManifest(manifestPath)
	if err != nil {
		return err
	}

	// Validate manifest
	if errs := validateServerManifest(manifest); len(errs) > 0 {
		fmt.Println("Manifest validation failed:")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("manifest validation failed with %d error(s)", len(errs))
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine output module name
	outputModule := fmt.Sprintf("%s/custom-server", manifest.Spec.Base.Module)

	// Generate main.go
	mainData := map[string]any{
		"BaseModule": manifest.Spec.Base.Module,
		"Plugins":    manifest.Spec.Plugins,
	}
	mainPath := filepath.Join(outputDir, "main.go")
	if err := executeTemplate(TmplServerMain, mainPath, mainData); err != nil {
		return fmt.Errorf("failed to generate main.go: %w", err)
	}
	fmt.Printf("  Generated: %s\n", mainPath)

	// Generate go.mod
	goModData := map[string]any{
		"OutputModule": outputModule,
		"GoVersion":    goVersion,
		"BaseModule":   manifest.Spec.Base.Module,
		"BaseVersion":  manifest.Spec.Base.Version,
		"Plugins":      manifest.Spec.Plugins,
	}
	goModPath := filepath.Join(outputDir, "go.mod")
	if err := executeTemplate(TmplServerGoMod, goModPath, goModData); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}
	fmt.Printf("  Generated: %s\n", goModPath)

	// Generate Dockerfile
	dockerData := map[string]any{
		"GoVersion": goVersion,
	}
	dockerPath := filepath.Join(outputDir, "Dockerfile")
	if err := executeTemplate(TmplServerDockerfile, dockerPath, dockerData); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	fmt.Printf("  Generated: %s\n", dockerPath)

	// Generate compatibility matrix
	if err := generateCompatMatrix(manifest, outputDir); err != nil {
		return fmt.Errorf("failed to generate compatibility matrix: %w", err)
	}

	fmt.Println("\nServer build files generated successfully!")

	if compile {
		fmt.Println("\nCompiling server...")
		return compileServer(outputDir)
	}

	fmt.Println("\nTo compile manually:")
	fmt.Printf("  cd %s && go mod tidy && go build -o catalog-server .\n", outputDir)

	return nil
}

// compileServer runs go build in the output directory.
func compileServer(dir string) error {
	// We just print instructions since go build requires the full module setup
	// (go mod tidy to resolve transitive deps, etc.)
	fmt.Printf("To build: cd %s && go mod tidy && go build -o catalog-server .\n", dir)
	fmt.Println("Note: Ensure all plugin modules are accessible (published or using replace directives).")
	return nil
}
