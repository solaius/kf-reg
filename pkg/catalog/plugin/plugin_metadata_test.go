package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPluginMetadata_Valid(t *testing.T) {
	content := `apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: agents
spec:
  displayName: Agents
  description: "Catalog of agent definitions"
  version: "0.1.0"
  owners:
    - team: ai-platform
      contact: "#ai-platform"
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.0.0"
    frameworkApi: v1alpha1
  providers:
    - yaml
  license: Apache-2.0
  repository: https://github.com/kubeflow/model-registry
`
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	spec, err := LoadPluginMetadata(path)
	require.NoError(t, err)

	assert.Equal(t, "catalog.kubeflow.org/v1alpha1", spec.APIVersion)
	assert.Equal(t, "CatalogPlugin", spec.Kind)
	assert.Equal(t, "agents", spec.Metadata.Name)
	assert.Equal(t, "Agents", spec.Spec.DisplayName)
	assert.Equal(t, "Catalog of agent definitions", spec.Spec.Description)
	assert.Equal(t, "0.1.0", spec.Spec.Version)
	require.Len(t, spec.Spec.Owners, 1)
	assert.Equal(t, "ai-platform", spec.Spec.Owners[0].Team)
	assert.Equal(t, "#ai-platform", spec.Spec.Owners[0].Contact)
	assert.Equal(t, "0.9.0", spec.Spec.Compatibility.CatalogServer.MinVersion)
	assert.Equal(t, "1.0.0", spec.Spec.Compatibility.CatalogServer.MaxVersion)
	assert.Equal(t, "v1alpha1", spec.Spec.Compatibility.FrameworkAPI)
	assert.Equal(t, []string{"yaml"}, spec.Spec.Providers)
	assert.Equal(t, "Apache-2.0", spec.Spec.License)
	assert.Equal(t, "https://github.com/kubeflow/model-registry", spec.Spec.Repository)
}

func TestLoadPluginMetadata_FileNotFound(t *testing.T) {
	_, err := LoadPluginMetadata("/nonexistent/plugin.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read plugin.yaml")
}

func TestLoadPluginMetadata_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid yaml"), 0644))

	_, err := LoadPluginMetadata(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse plugin.yaml")
}

func TestValidatePluginMetadata(t *testing.T) {
	tests := []struct {
		name     string
		spec     PluginMetadataSpec
		wantErrs []string
	}{
		{
			name: "valid spec",
			spec: PluginMetadataSpec{
				APIVersion: "catalog.kubeflow.org/v1alpha1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "A test plugin",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "team-a"}},
					Compatibility: CompatibilitySpec{
						CatalogServer: VersionRange{MinVersion: "0.9.0", MaxVersion: "1.0.0"},
						FrameworkAPI:  "v1alpha1",
					},
					Providers: []string{"yaml"},
				},
			},
			wantErrs: nil,
		},
		{
			name: "missing apiVersion",
			spec: PluginMetadataSpec{
				Kind:     "CatalogPlugin",
				Metadata: PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{"yaml"},
				},
			},
			wantErrs: []string{"apiVersion is required"},
		},
		{
			name: "wrong kind",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "WrongKind",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{"yaml"},
				},
			},
			wantErrs: []string{"kind must be CatalogPlugin"},
		},
		{
			name: "missing metadata.name",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{"yaml"},
				},
			},
			wantErrs: []string{"metadata.name is required"},
		},
		{
			name: "invalid semver version",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "not-semver",
					Owners:      []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{"yaml"},
				},
			},
			wantErrs: []string{"spec.version is not valid semver"},
		},
		{
			name: "empty owners",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName:   "Test",
					Description:   "desc",
					Version:       "1.0.0",
					Owners:        []OwnerRef{},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{"yaml"},
				},
			},
			wantErrs: []string{"spec.owners must have at least one entry"},
		},
		{
			name: "owner missing team",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName:   "Test",
					Description:   "desc",
					Version:       "1.0.0",
					Owners:        []OwnerRef{{Contact: "c"}},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{"yaml"},
				},
			},
			wantErrs: []string{"spec.owners[0].team is required"},
		},
		{
			name: "missing frameworkApi",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "t"}},
					Providers:   []string{"yaml"},
				},
			},
			wantErrs: []string{"spec.compatibility.frameworkApi is required"},
		},
		{
			name: "minVersion > maxVersion",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{
						CatalogServer: VersionRange{MinVersion: "2.0.0", MaxVersion: "1.0.0"},
						FrameworkAPI:  "v1alpha1",
					},
					Providers: []string{"yaml"},
				},
			},
			wantErrs: []string{"spec.compatibility.catalogServer.minVersion (2.0.0) must be <= maxVersion (1.0.0)"},
		},
		{
			name: "wildcard maxVersion is allowed",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName: "Test",
					Description: "desc",
					Version:     "1.0.0",
					Owners:      []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{
						CatalogServer: VersionRange{MinVersion: "0.9.0", MaxVersion: "1.x"},
						FrameworkAPI:  "v1alpha1",
					},
					Providers: []string{"yaml"},
				},
			},
			wantErrs: nil,
		},
		{
			name: "empty providers",
			spec: PluginMetadataSpec{
				APIVersion: "v1",
				Kind:       "CatalogPlugin",
				Metadata:   PluginMetadataName{Name: "test"},
				Spec: PluginMetadataBody{
					DisplayName:   "Test",
					Description:   "desc",
					Version:       "1.0.0",
					Owners:        []OwnerRef{{Team: "t"}},
					Compatibility: CompatibilitySpec{FrameworkAPI: "v1alpha1"},
					Providers:     []string{},
				},
			},
			wantErrs: []string{"spec.providers must have at least one entry"},
		},
		{
			name: "multiple errors",
			spec: PluginMetadataSpec{},
			wantErrs: []string{
				"apiVersion is required",
				"kind is required",
				"metadata.name is required",
				"spec.displayName is required",
				"spec.description is required",
				"spec.version is required",
				"spec.owners must have at least one entry",
				"spec.compatibility.frameworkApi is required",
				"spec.providers must have at least one entry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePluginMetadata(&tt.spec)
			if tt.wantErrs == nil {
				assert.Empty(t, errs)
			} else {
				require.Len(t, errs, len(tt.wantErrs), "got errors: %v", errs)
				for i, want := range tt.wantErrs {
					assert.Contains(t, errs[i], want)
				}
			}
		})
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input             string
		wantMajor         int
		wantMinor         int
		wantPatch         int
		wantErr           bool
		wantErrContains   string
	}{
		{input: "1.2.3", wantMajor: 1, wantMinor: 2, wantPatch: 3},
		{input: "0.1.0", wantMajor: 0, wantMinor: 1, wantPatch: 0},
		{input: "10.20.30", wantMajor: 10, wantMinor: 20, wantPatch: 30},
		{input: "v1.2.3", wantMajor: 1, wantMinor: 2, wantPatch: 3},
		{input: "0.0.0", wantMajor: 0, wantMinor: 0, wantPatch: 0},
		{input: "1.2", wantErr: true, wantErrContains: "expected 3 dot-separated components"},
		{input: "1.2.3.4", wantErr: true, wantErrContains: "expected 3 dot-separated components"},
		{input: "a.b.c", wantErr: true, wantErrContains: "invalid major version"},
		{input: "1.b.3", wantErr: true, wantErrContains: "invalid minor version"},
		{input: "1.2.c", wantErr: true, wantErrContains: "invalid patch version"},
		{input: "", wantErr: true, wantErrContains: "expected 3 dot-separated components"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, err := ParseSemver(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMajor, major)
				assert.Equal(t, tt.wantMinor, minor)
				assert.Equal(t, tt.wantPatch, patch)
			}
		})
	}
}

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		version string
		part    string
		want    string
		wantErr bool
	}{
		{version: "0.1.0", part: "patch", want: "0.1.1"},
		{version: "0.1.0", part: "minor", want: "0.2.0"},
		{version: "0.1.0", part: "major", want: "1.0.0"},
		{version: "1.2.3", part: "patch", want: "1.2.4"},
		{version: "1.2.3", part: "minor", want: "1.3.0"},
		{version: "1.2.3", part: "major", want: "2.0.0"},
		{version: "v1.0.0", part: "patch", want: "1.0.1"},
		{version: "0.0.0", part: "patch", want: "0.0.1"},
		{version: "0.0.0", part: "minor", want: "0.1.0"},
		{version: "0.0.0", part: "major", want: "1.0.0"},
		{version: "invalid", part: "patch", wantErr: true},
		{version: "1.0.0", part: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.part, func(t *testing.T) {
			result, err := BumpVersion(tt.version, tt.part)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}
