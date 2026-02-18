package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
	"github.com/spf13/cobra"
)

// CompatMatrixEntry holds the compatibility data for one plugin in the matrix.
type CompatMatrixEntry struct {
	Name         string
	Module       string
	Version      string
	MinServer    string
	MaxServer    string
	FrameworkAPI string
	Compatible   string // "Yes", "No", or "Unknown"
}

// checkPluginCompatibility evaluates whether a plugin is compatible with a
// given server version. It returns "Yes", "No", or "Unknown".
func checkPluginCompatibility(serverVersion string, entry CompatMatrixEntry) string {
	if entry.MinServer == "" && entry.MaxServer == "" {
		return "Unknown"
	}

	sMaj, sMin, sPatch, sErr := plugin.ParseSemver(serverVersion)
	if sErr != nil {
		return "Unknown"
	}
	serverVal := sMaj*1000000 + sMin*1000 + sPatch

	// Check minVersion
	if entry.MinServer != "" && !strings.Contains(entry.MinServer, "x") {
		minMaj, minMin, minPatch, err := plugin.ParseSemver(entry.MinServer)
		if err == nil {
			minVal := minMaj*1000000 + minMin*1000 + minPatch
			if serverVal < minVal {
				return "No"
			}
		}
	}

	// Check maxVersion
	if entry.MaxServer != "" && !strings.Contains(entry.MaxServer, "x") {
		maxMaj, maxMin, maxPatch, err := plugin.ParseSemver(entry.MaxServer)
		if err == nil {
			maxVal := maxMaj*1000000 + maxMin*1000 + maxPatch
			if serverVal > maxVal {
				return "No"
			}
		}
	}

	// Handle wildcard maxVersion like "1.x" -- check major version only
	if entry.MaxServer != "" && strings.Contains(entry.MaxServer, "x") {
		parts := strings.Split(entry.MaxServer, ".")
		if len(parts) >= 1 {
			var maxMaj int
			if _, err := fmt.Sscanf(parts[0], "%d", &maxMaj); err == nil {
				if sMaj > maxMaj {
					return "No"
				}
			}
		}
	}

	return "Yes"
}

// buildCompatMatrix creates CompatMatrixEntry values from a ServerManifest,
// loading each plugin's plugin.yaml metadata if available, falling back to
// "Unknown" compatibility when metadata is not found.
func buildCompatMatrix(manifest *ServerManifest) []CompatMatrixEntry {
	var entries []CompatMatrixEntry

	for _, p := range manifest.Spec.Plugins {
		entry := CompatMatrixEntry{
			Name:         p.Name,
			Module:       p.Module,
			Version:      p.Version,
			MinServer:    "-",
			MaxServer:    "-",
			FrameworkAPI: "-",
			Compatible:   "Unknown",
		}

		// Try to load plugin.yaml from plugin directory (convention: catalog/plugins/<name>)
		pluginDir := filepath.Join("catalog", "plugins", p.Name)
		meta, err := loadPluginConfig(pluginDir)
		if err == nil && meta != nil {
			entry.MinServer = meta.Spec.Compatibility.CatalogServer.MinVersion
			entry.MaxServer = meta.Spec.Compatibility.CatalogServer.MaxVersion
			entry.FrameworkAPI = meta.Spec.Compatibility.FrameworkAPI
			if entry.MinServer == "" {
				entry.MinServer = "-"
			}
			if entry.MaxServer == "" {
				entry.MaxServer = "-"
			}
			if entry.FrameworkAPI == "" {
				entry.FrameworkAPI = "-"
			}

			entry.Compatible = checkPluginCompatibility(manifest.Spec.Base.Version, entry)
		}

		entries = append(entries, entry)
	}

	return entries
}

// generateCompatMatrix generates the COMPATIBILITY.md file from a manifest.
func generateCompatMatrix(manifest *ServerManifest, outputDir string) error {
	entries := buildCompatMatrix(manifest)

	data := map[string]any{
		"GeneratedAt": time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		"BaseModule":  manifest.Spec.Base.Module,
		"BaseVersion": manifest.Spec.Base.Version,
		"Plugins":     entries,
	}

	outputPath := filepath.Join(outputDir, "COMPATIBILITY.md")
	if err := executeTemplate(TmplServerCompatMatrix, outputPath, data); err != nil {
		return fmt.Errorf("failed to generate compatibility matrix: %w", err)
	}
	fmt.Printf("  Generated: %s\n", outputPath)

	return nil
}

func newCompatMatrixCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "compat-matrix <manifest>",
		Short: "Generate a compatibility matrix from a server manifest",
		Long: `Generate a compatibility matrix markdown file from a catalog-server-manifest.yaml.

The matrix shows each plugin's version constraints, framework API level, and
whether it is compatible with the base server version declared in the manifest.

Plugin metadata is loaded from catalog/plugins/<name>/plugin.yaml when available.

Examples:
  catalog-gen compat-matrix catalog-server-manifest.yaml
  catalog-gen compat-matrix manifest.yaml --output matrix.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompatMatrixCmd(args[0], outputPath)
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path (stdout if empty)")

	return cmd
}

func runCompatMatrixCmd(manifestPath, outputPath string) error {
	manifest, err := loadServerManifest(manifestPath)
	if err != nil {
		return err
	}

	if errs := validateServerManifest(manifest); len(errs) > 0 {
		fmt.Println("Manifest validation failed:")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("manifest validation failed with %d error(s)", len(errs))
	}

	entries := buildCompatMatrix(manifest)

	data := map[string]any{
		"GeneratedAt": time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		"BaseModule":  manifest.Spec.Base.Module,
		"BaseVersion": manifest.Spec.Base.Version,
		"Plugins":     entries,
	}

	if outputPath == "" {
		// Write to stdout using the template
		content, err := templateFS.ReadFile(TmplServerCompatMatrix)
		if err != nil {
			return fmt.Errorf("failed to read compat matrix template: %w", err)
		}
		tmpl, err := template.New("compat").Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse compat matrix template: %w", err)
		}
		return tmpl.Execute(os.Stdout, data)
	}

	// Write to file
	if err := executeTemplate(TmplServerCompatMatrix, outputPath, data); err != nil {
		return fmt.Errorf("failed to generate compatibility matrix: %w", err)
	}
	fmt.Printf("Generated: %s\n", outputPath)

	return nil
}
