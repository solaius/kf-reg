package mcp

import (
	"context"
	"fmt"
	"time"

	pkgcatalog "github.com/kubeflow/model-registry/pkg/catalog"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager       = (*McpServerCatalogPlugin)(nil)
	_ plugin.RefreshProvider     = (*McpServerCatalogPlugin)(nil)
	_ plugin.DiagnosticsProvider = (*McpServerCatalogPlugin)(nil)
	_ plugin.CapabilitiesProvider = (*McpServerCatalogPlugin)(nil)
	_ plugin.UIHintsProvider     = (*McpServerCatalogPlugin)(nil)
	_ plugin.CLIHintsProvider    = (*McpServerCatalogPlugin)(nil)
)

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
		info := plugin.SourceInfo{
			ID:         src.ID,
			Name:       src.Name,
			Type:       src.Type,
			Enabled:    src.IsEnabled(),
			Labels:     src.Labels,
			Properties: src.Properties,
			Status: plugin.SourceStatus{
				State: sourceState(src),
			},
		}

		// Get entity count from the repository
		if src.IsEnabled() {
			info.Status.State = plugin.SourceStateAvailable
		}

		result = append(result, info)
	}

	return result, nil
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

	return result, nil
}

// ApplySource adds or updates a source configuration.
func (p *McpServerCatalogPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
	enabled := true
	if src.Enabled != nil {
		enabled = *src.Enabled
	}

	source := pkgcatalog.Source{
		ID:         src.ID,
		Name:       src.Name,
		Type:       src.Type,
		Enabled:    &enabled,
		Labels:     src.Labels,
		Properties: src.Properties,
		Origin:     "api",
	}

	sources := map[string]pkgcatalog.Source{
		src.ID: source,
	}

	return p.loader.Sources.Merge("api", sources)
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

// sourceState returns the state string for a source.
func sourceState(src pkgcatalog.Source) string {
	if !src.IsEnabled() {
		return plugin.SourceStateDisabled
	}
	return plugin.SourceStateAvailable
}
