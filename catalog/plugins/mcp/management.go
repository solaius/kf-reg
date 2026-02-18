package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	pkgcatalog "github.com/kubeflow/model-registry/pkg/catalog"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*McpServerCatalogPlugin)(nil)
	_ plugin.RefreshProvider        = (*McpServerCatalogPlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*McpServerCatalogPlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*McpServerCatalogPlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*McpServerCatalogPlugin)(nil)
	_ plugin.UIHintsProvider        = (*McpServerCatalogPlugin)(nil)
	_ plugin.CLIHintsProvider       = (*McpServerCatalogPlugin)(nil)
)

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *McpServerCatalogPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/mcp_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "McpServer catalog",
			DisplayName: "MCP Servers",
			Icon:        "server",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "McpServer",
				Plural:      "mcpservers",
				DisplayName: "MCP Server",
				Description: "Model Context Protocol server entries",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/mcpservers",
					Get:  basePath + "/mcpservers/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "deploymentMode", DisplayName: "Deployment", Path: "deploymentMode", Type: "string", Sortable: true, Width: "md"},
						{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Sortable: true, Width: "md"},
						{Name: "transportType", DisplayName: "Transport", Path: "transportType", Type: "string", Sortable: true, Width: "sm"},
						{Name: "toolCount", DisplayName: "Tools", Path: "toolCount", Type: "integer", Sortable: true, Width: "sm"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Sortable: true, Width: "md"},
						{Name: "category", DisplayName: "Category", Path: "category", Type: "string", Sortable: true, Width: "md"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "deploymentMode", DisplayName: "Deployment Mode", Type: "select", Options: []string{"local", "remote", "hybrid"}, Operators: []string{"=", "!="}},
						{Name: "provider", DisplayName: "Provider", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "category", DisplayName: "Category", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "license", DisplayName: "License", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "transportType", DisplayName: "Transport Type", Type: "select", Options: []string{"stdio", "sse", "streamable-http"}, Operators: []string{"=", "!="}},
						{Name: "toolCount", DisplayName: "Tool Count", Type: "number", Operators: []string{"=", ">", "<", ">=", "<="}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "deploymentMode", DisplayName: "Deployment Mode", Path: "deploymentMode", Type: "string", Section: "Overview"},
						{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Section: "Overview"},
						{Name: "category", DisplayName: "Category", Path: "category", Type: "string", Section: "Overview"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Section: "Overview"},
						{Name: "serverUrl", DisplayName: "Server URL", Path: "serverUrl", Type: "string", Section: "Connection"},
						{Name: "image", DisplayName: "Container Image", Path: "image", Type: "string", Section: "Connection"},
						{Name: "endpoint", DisplayName: "Remote Endpoint", Path: "endpoint", Type: "string", Section: "Connection"},
						{Name: "supportedTransports", DisplayName: "Supported Transports", Path: "supportedTransports", Type: "string", Section: "Connection"},
						{Name: "transportType", DisplayName: "Transport Type", Path: "transportType", Type: "string", Section: "Connection"},
						{Name: "toolCount", DisplayName: "Tool Count", Path: "toolCount", Type: "integer", Section: "Statistics"},
						{Name: "resourceCount", DisplayName: "Resource Count", Path: "resourceCount", Type: "integer", Section: "Statistics"},
						{Name: "promptCount", DisplayName: "Prompt Count", Path: "promptCount", Type: "integer", Section: "Statistics"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "server",
					Color:          "#0066CC",
					NameField:      "name",
					DetailSections: []string{"Overview", "Connection", "Statistics"},
					ListView: &plugin.ListViewHints{
						TitleField: "name",
						Columns: []plugin.ColumnDisplay{
							{Field: "name", Label: "Name", Display: plugin.DisplayLink},
							{Field: "deploymentMode", Label: "Deployment", Display: plugin.DisplayBadge},
							{Field: "provider", Label: "Provider", Display: plugin.DisplayText},
							{Field: "transportType", Label: "Transport", Display: plugin.DisplayBadge, Width: "sm"},
							{Field: "toolCount", Label: "Tools", Display: plugin.DisplayText, Width: "sm"},
						},
						DefaultSort: &plugin.SortHint{Field: "name", Direction: "asc"},
					},
					DetailView: &plugin.DetailViewHints{
						Sections: []plugin.DetailSection{
							{
								Title: "Overview",
								Fields: []plugin.FieldDisplay{
									{Field: "name", Label: "Name", Display: plugin.DisplayText},
									{Field: "description", Label: "Description", Display: plugin.DisplayMarkdown},
									{Field: "deploymentMode", Label: "Deployment Mode", Display: plugin.DisplayBadge},
									{Field: "provider", Label: "Provider", Display: plugin.DisplayText},
									{Field: "category", Label: "Category", Display: plugin.DisplayBadge},
								},
							},
							{
								Title: "Connection",
								Fields: []plugin.FieldDisplay{
									{Field: "serverUrl", Label: "Server URL", Display: plugin.DisplayLink},
									{Field: "endpoint", Label: "Endpoint", Display: plugin.DisplayLink},
									{Field: "image", Label: "Container Image", Display: plugin.DisplayCode},
									{Field: "transportType", Label: "Transport", Display: plugin.DisplayBadge},
								},
							},
							{
								Title: "Statistics",
								Fields: []plugin.FieldDisplay{
									{Field: "toolCount", Label: "Tools", Display: plugin.DisplayText},
									{Field: "resourceCount", Label: "Resources", Display: plugin.DisplayText},
									{Field: "promptCount", Label: "Prompts", Display: plugin.DisplayText},
								},
							},
						},
					},
					Search: &plugin.SearchHints{
						SearchableFields: []string{"name", "description", "provider", "category"},
						Facets: []plugin.FacetHint{
							{Field: "deploymentMode", Display: plugin.DisplayBadge},
							{Field: "transportType", Display: plugin.DisplayBadge},
							{Field: "category", Display: plugin.DisplayBadge},
						},
					},
					ActionHints: &plugin.ActionDisplayHints{
						Primary:   []string{"refresh"},
						Secondary: []string{"tag", "annotate", "deprecate"},
					},
				},
				Actions: []string{"tag", "annotate", "deprecate", "refresh"},
			},
		},
		Sources: &plugin.SourceCapabilities{
			Manageable:  true,
			Refreshable: true,
			Types:       []string{"yaml"},
		},
		Actions: []plugin.ActionDefinition{
			{ID: "tag", DisplayName: "Tag", Description: "Add or remove tags on an entity", Scope: "asset", SupportsDryRun: true, Idempotent: true},
			{ID: "annotate", DisplayName: "Annotate", Description: "Add or update annotations on an entity", Scope: "asset", SupportsDryRun: true, Idempotent: true},
			{ID: "deprecate", DisplayName: "Deprecate", Description: "Mark an entity as deprecated", Scope: "asset", SupportsDryRun: true, Idempotent: true},
			{ID: "refresh", DisplayName: "Refresh", Description: "Refresh entities from a source", Scope: "source", SupportsDryRun: false, Idempotent: true},
		},
	}
}

// Capabilities returns the plugin's advertised capabilities.
func (p *McpServerCatalogPlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"McpServer"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
		Artifacts:    false,
	}
}

// ListSources returns information about all configured sources.
func (p *McpServerCatalogPlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
	allSources := p.loader.Sources.AllSources()
	result := make([]plugin.SourceInfo, 0, len(allSources))

	for _, src := range allSources {
		// Copy properties so we can enrich without mutating the original.
		props := make(map[string]any, len(src.Properties))
		for k, v := range src.Properties {
			props[k] = v
		}

		// For YAML sources with a file path, read the file content.
		if yamlPath, ok := props["yamlCatalogPath"].(string); ok && yamlPath != "" {
			resolvedPath := resolveSourcePath(src, yamlPath)
			if data, err := os.ReadFile(resolvedPath); err == nil {
				props["content"] = string(data)
			}
		}

		info := plugin.SourceInfo{
			ID:         src.ID,
			Name:       src.Name,
			Type:       src.Type,
			Enabled:    src.IsEnabled(),
			Labels:     src.Labels,
			Properties: props,
			Status: plugin.SourceStatus{
				State: sourceState(src),
			},
		}

		// Get entity count from the repository
		if src.IsEnabled() {
			info.Status.State = plugin.SourceStateAvailable
			count, err := p.services.McpServerRepository.CountBySource(src.ID)
			if err == nil {
				info.Status.EntityCount = count
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// mcpServerStrictEntry defines all known fields for an MCP server entry.
// Used with yaml.Decoder.KnownFields(true) to detect unknown fields in
// properties.content during validation.
type mcpServerStrictEntry struct {
	Name                string         `yaml:"name"`
	ExternalId          string         `yaml:"externalId"`
	Description         *string        `yaml:"description"`
	ServerUrl           string         `yaml:"serverUrl"`
	TransportType       *string        `yaml:"transportType"`
	ToolCount           *int32         `yaml:"toolCount"`
	ResourceCount       *int32         `yaml:"resourceCount"`
	PromptCount         *int32         `yaml:"promptCount"`
	DeploymentMode      *string        `yaml:"deploymentMode"`
	Image               *string        `yaml:"image"`
	Endpoint            *string        `yaml:"endpoint"`
	SupportedTransports *string        `yaml:"supportedTransports"`
	License             *string        `yaml:"license"`
	Verified            *bool          `yaml:"verified"`
	Certified           *bool          `yaml:"certified"`
	Provider            *string        `yaml:"provider"`
	Logo                *string        `yaml:"logo"`
	Category            *string        `yaml:"category"`
	CustomProperties    map[string]any `yaml:"customProperties"`
}

// mcpServerStrictCatalog is the top-level wrapper for strict YAML decoding.
type mcpServerStrictCatalog struct {
	McpServers []mcpServerStrictEntry `yaml:"mcpservers"`
}

// ValidateSource validates a source configuration without applying it.
func (p *McpServerCatalogPlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
	result := &plugin.ValidationResult{Valid: true}

	if src.ID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, plugin.ValidationError{
			Field:   "id",
			Message: "source ID is required",
		})
	}

	if src.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, plugin.ValidationError{
			Field:   "name",
			Message: "source name is required",
		})
	}

	if src.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, plugin.ValidationError{
			Field:   "type",
			Message: "source type is required",
		})
	} else {
		// Verify the provider type is registered
		registry := p.loader.Config().ProviderRegistry
		if registry != nil && !registry.Has(src.Type) {
			result.Valid = false
			result.Errors = append(result.Errors, plugin.ValidationError{
				Field:   "type",
				Message: fmt.Sprintf("unknown provider type %q", src.Type),
			})
		}
	}

	// Strict-decode properties.content to detect unknown fields.
	if errs := validateMcpContent(src); len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, errs...)
	}

	return result, nil
}

// validateMcpContent performs strict YAML decoding of properties.content
// against the known MCP server schema. Returns validation errors for any
// unknown fields found in the mcpservers entries.
func validateMcpContent(src plugin.SourceConfigInput) []plugin.ValidationError {
	if src.Properties == nil {
		return nil
	}
	raw, ok := src.Properties["content"]
	if !ok {
		return nil
	}
	content, ok := raw.(string)
	if !ok || content == "" {
		return nil
	}

	dec := yaml.NewDecoder(bytes.NewReader([]byte(content)))
	dec.KnownFields(true)

	var catalog mcpServerStrictCatalog
	if err := dec.Decode(&catalog); err != nil {
		return []plugin.ValidationError{
			{
				Field:   "properties.content",
				Message: fmt.Sprintf("unknown or invalid fields in content: %v", err),
			},
		}
	}
	return nil
}

// ApplySource adds or updates a source configuration.
func (p *McpServerCatalogPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
	enabled := true
	if src.Enabled != nil {
		enabled = *src.Enabled
	}

	// If both content and yamlCatalogPath are provided, write content to the file.
	if src.Properties != nil {
		if content, ok := src.Properties["content"].(string); ok && content != "" {
			if yamlPath, ok := src.Properties["yamlCatalogPath"].(string); ok && yamlPath != "" {
				// Resolve path using the existing source's origin if available.
				existingSource, exists := p.loader.Sources.AllSources()[src.ID]
				if exists {
					resolved := resolveSourcePath(existingSource, yamlPath)
					if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
						return fmt.Errorf("failed to write YAML file %s: %w", resolved, err)
					}
				}
			}
		}

		// Don't persist inline content in source properties — the file is the source of truth.
		delete(src.Properties, "content")
	}

	origin := "api"
	// Preserve the origin of an existing source so path resolution continues to work.
	if existingSource, exists := p.loader.Sources.AllSources()[src.ID]; exists && existingSource.Origin != "" {
		origin = existingSource.Origin
	}

	source := pkgcatalog.Source{
		ID:         src.ID,
		Name:       src.Name,
		Type:       src.Type,
		Enabled:    &enabled,
		Labels:     src.Labels,
		Properties: src.Properties,
		Origin:     origin,
	}

	sources := map[string]pkgcatalog.Source{
		src.ID: source,
	}

	return p.loader.Sources.Merge(origin, sources)
}

// EnableSource enables or disables a source.
func (p *McpServerCatalogPlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	allSources := p.loader.Sources.AllSources()
	src, exists := allSources[id]
	if !exists {
		return fmt.Errorf("source %q not found", id)
	}

	src.Enabled = &enabled
	sources := map[string]pkgcatalog.Source{
		id: src,
	}

	return p.loader.Sources.Merge(src.Origin, sources)
}

// DeleteSource removes a source and its associated entities.
func (p *McpServerCatalogPlugin) DeleteSource(ctx context.Context, id string) error {
	// Remove entities from the database
	if err := p.services.McpServerRepository.DeleteBySource(id); err != nil {
		return fmt.Errorf("failed to delete entities for source %q: %w", id, err)
	}

	// Disable the source so it won't be loaded again
	disabled := false
	allSources := p.loader.Sources.AllSources()
	if src, exists := allSources[id]; exists {
		src.Enabled = &disabled
		sources := map[string]pkgcatalog.Source{
			id: src,
		}
		return p.loader.Sources.Merge(src.Origin, sources)
	}

	return nil
}

// Refresh triggers a reload of a specific source.
func (p *McpServerCatalogPlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
	start := time.Now()

	if err := p.loader.Reload(ctx); err != nil {
		return &plugin.RefreshResult{
			SourceID: sourceID,
			Duration: time.Since(start),
			Error:    err.Error(),
		}, nil
	}

	return &plugin.RefreshResult{
		SourceID: sourceID,
		Duration: time.Since(start),
	}, nil
}

// RefreshAll triggers a reload of all sources.
func (p *McpServerCatalogPlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
	start := time.Now()

	if err := p.loader.Reload(ctx); err != nil {
		return &plugin.RefreshResult{
			Duration: time.Since(start),
			Error:    err.Error(),
		}, nil
	}

	return &plugin.RefreshResult{
		Duration: time.Since(start),
	}, nil
}

// Diagnostics returns diagnostic information about the plugin.
func (p *McpServerCatalogPlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
	diag := &plugin.PluginDiagnostics{
		PluginName: PluginName,
		Sources:    make([]plugin.SourceDiagnostic, 0),
	}

	allSources := p.loader.Sources.AllSources()
	for _, src := range allSources {
		sd := plugin.SourceDiagnostic{
			ID:    src.ID,
			Name:  src.Name,
			State: sourceState(src),
		}
		if src.IsEnabled() {
			count, err := p.services.McpServerRepository.CountBySource(src.ID)
			if err == nil {
				sd.EntityCount = count
			}
		}
		diag.Sources = append(diag.Sources, sd)
	}

	return diag, nil
}

// UIHints returns display hints for the UI.
func (p *McpServerCatalogPlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "deploymentMode", Label: "Deployment", Sortable: true, Filterable: true},
			{Field: "provider", Label: "Provider", Sortable: true, Filterable: true},
			{Field: "transportType", Label: "Transport", Sortable: true, Filterable: true},
			{Field: "toolCount", Label: "Tools", Sortable: true},
			{Field: "license", Label: "License", Sortable: true, Filterable: true},
			{Field: "category", Label: "Category", Sortable: true, Filterable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "deploymentMode", Label: "Deployment Mode"},
			{Field: "serverUrl", Label: "Server URL"},
			{Field: "image", Label: "Container Image"},
			{Field: "endpoint", Label: "Remote Endpoint"},
			{Field: "supportedTransports", Label: "Supported Transports"},
			{Field: "license", Label: "License"},
			{Field: "provider", Label: "Provider"},
			{Field: "category", Label: "Category"},
			{Field: "toolCount", Label: "Tool Count"},
			{Field: "resourceCount", Label: "Resource Count"},
			{Field: "promptCount", Label: "Prompt Count"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *McpServerCatalogPlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "deploymentMode", "provider", "toolCount", "category"},
		SortField:        "name",
		FilterableFields: []string{"name", "deploymentMode", "provider", "category", "license", "transportType"},
	}
}

// resolveSourcePath resolves a relative path against the source's origin directory.
func resolveSourcePath(src pkgcatalog.Source, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if src.Origin != "" {
		return filepath.Join(filepath.Dir(src.Origin), path)
	}
	return path
}

// sourceState returns the state string for a source.
func sourceState(src pkgcatalog.Source) string {
	if !src.IsEnabled() {
		return plugin.SourceStateDisabled
	}
	return plugin.SourceStateAvailable
}
