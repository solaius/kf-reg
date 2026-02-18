package plugin

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginMetadataSpec is the top-level structure for plugin.yaml.
type PluginMetadataSpec struct {
	APIVersion string             `yaml:"apiVersion" json:"apiVersion"` // catalog.kubeflow.org/v1alpha1
	Kind       string             `yaml:"kind" json:"kind"`             // CatalogPlugin
	Metadata   PluginMetadataName `yaml:"metadata" json:"metadata"`
	Spec       PluginMetadataBody `yaml:"spec" json:"spec"`
}

// PluginMetadataName holds the plugin name.
type PluginMetadataName struct {
	Name string `yaml:"name" json:"name"`
}

// PluginMetadataBody holds the plugin specification fields.
type PluginMetadataBody struct {
	DisplayName   string            `yaml:"displayName" json:"displayName"`
	Description   string            `yaml:"description" json:"description"`
	Version       string            `yaml:"version" json:"version"`
	Owners        []OwnerRef        `yaml:"owners" json:"owners"`
	Compatibility CompatibilitySpec `yaml:"compatibility" json:"compatibility"`
	Providers     []string          `yaml:"providers" json:"providers"`
	License       string            `yaml:"license,omitempty" json:"license,omitempty"`
	Repository    string            `yaml:"repository,omitempty" json:"repository,omitempty"`
}

// OwnerRef identifies an owning team.
type OwnerRef struct {
	Team    string `yaml:"team" json:"team"`
	Contact string `yaml:"contact,omitempty" json:"contact,omitempty"`
}

// CompatibilitySpec declares version constraints for server and framework API.
type CompatibilitySpec struct {
	CatalogServer VersionRange `yaml:"catalogServer" json:"catalogServer"`
	FrameworkAPI  string       `yaml:"frameworkApi" json:"frameworkApi"`
}

// VersionRange represents a min/max version constraint.
type VersionRange struct {
	MinVersion string `yaml:"minVersion" json:"minVersion"`
	MaxVersion string `yaml:"maxVersion" json:"maxVersion"`
}

// LoadPluginMetadata reads and parses a plugin.yaml file from the given path.
func LoadPluginMetadata(path string) (*PluginMetadataSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin.yaml: %w", err)
	}

	var spec PluginMetadataSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse plugin.yaml: %w", err)
	}

	return &spec, nil
}

// ValidatePluginMetadata checks a PluginMetadataSpec for errors and returns
// a list of human-readable validation error strings. An empty slice means valid.
func ValidatePluginMetadata(spec *PluginMetadataSpec) []string {
	var errs []string

	if spec.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if spec.Kind == "" {
		errs = append(errs, "kind is required")
	} else if spec.Kind != "CatalogPlugin" {
		errs = append(errs, fmt.Sprintf("kind must be CatalogPlugin, got %q", spec.Kind))
	}
	if spec.Metadata.Name == "" {
		errs = append(errs, "metadata.name is required")
	}
	if spec.Spec.DisplayName == "" {
		errs = append(errs, "spec.displayName is required")
	}
	if spec.Spec.Description == "" {
		errs = append(errs, "spec.description is required")
	}
	if spec.Spec.Version == "" {
		errs = append(errs, "spec.version is required")
	} else if _, _, _, err := ParseSemver(spec.Spec.Version); err != nil {
		errs = append(errs, fmt.Sprintf("spec.version is not valid semver: %v", err))
	}
	if len(spec.Spec.Owners) == 0 {
		errs = append(errs, "spec.owners must have at least one entry")
	} else {
		for i, owner := range spec.Spec.Owners {
			if owner.Team == "" {
				errs = append(errs, fmt.Sprintf("spec.owners[%d].team is required", i))
			}
		}
	}
	if spec.Spec.Compatibility.FrameworkAPI == "" {
		errs = append(errs, "spec.compatibility.frameworkApi is required")
	}

	// Validate minVersion/maxVersion when both are strict semver
	min := spec.Spec.Compatibility.CatalogServer.MinVersion
	max := spec.Spec.Compatibility.CatalogServer.MaxVersion
	if min != "" && max != "" {
		// Only compare if both are strict semver (no wildcard like "1.x")
		if !strings.Contains(min, "x") && !strings.Contains(max, "x") {
			minMaj, minMin, minPatch, minErr := ParseSemver(min)
			maxMaj, maxMin, maxPatch, maxErr := ParseSemver(max)
			if minErr == nil && maxErr == nil {
				minVal := minMaj*1000000 + minMin*1000 + minPatch
				maxVal := maxMaj*1000000 + maxMin*1000 + maxPatch
				if minVal > maxVal {
					errs = append(errs, fmt.Sprintf("spec.compatibility.catalogServer.minVersion (%s) must be <= maxVersion (%s)", min, max))
				}
			}
		}
	}

	if len(spec.Spec.Providers) == 0 {
		errs = append(errs, "spec.providers must have at least one entry")
	}

	return errs
}

// ParseSemver parses a semver version string of the form "major.minor.patch"
// and returns the three integer components. Pre-release and build metadata
// suffixes are not supported.
func ParseSemver(version string) (major, minor, patch int, err error) {
	// Strip leading "v" if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected 3 dot-separated components, got %d in %q", len(parts), version)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	if major < 0 || minor < 0 || patch < 0 {
		return 0, 0, 0, fmt.Errorf("version components must be non-negative")
	}

	return major, minor, patch, nil
}

// BumpVersion increments a semver version string by the specified part
// ("major", "minor", or "patch") and returns the new version string.
func BumpVersion(version string, part string) (string, error) {
	major, minor, patch, err := ParseSemver(version)
	if err != nil {
		return "", fmt.Errorf("cannot bump version: %w", err)
	}

	switch part {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", fmt.Errorf("invalid bump part %q: must be major, minor, or patch", part)
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}
