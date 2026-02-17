package policies

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*PolicyPlugin)(nil)
	_ plugin.RefreshProvider        = (*PolicyPlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*PolicyPlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*PolicyPlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*PolicyPlugin)(nil)
	_ plugin.UIHintsProvider        = (*PolicyPlugin)(nil)
	_ plugin.CLIHintsProvider       = (*PolicyPlugin)(nil)
	_ plugin.BasePathProvider       = (*PolicyPlugin)(nil)
)

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *PolicyPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/policies_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "Policy catalog for AI governance and access control",
			DisplayName: "Policies",
			Icon:        "gavel",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "Policy",
				Plural:      "policies",
				DisplayName: "Policy",
				Description: "AI governance and access control policies",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/policies",
					Get:  basePath + "/policies/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "policyType", DisplayName: "Policy Type", Path: "policyType", Type: "string", Sortable: true, Width: "md"},
						{Name: "language", DisplayName: "Language", Path: "language", Type: "string", Sortable: true, Width: "sm"},
						{Name: "enforcementScope", DisplayName: "Scope", Path: "enforcementScope", Type: "string", Sortable: true, Width: "md"},
						{Name: "enforcementMode", DisplayName: "Mode", Path: "enforcementMode", Type: "string", Sortable: true, Width: "sm"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "policyType", DisplayName: "Policy Type", Type: "select", Options: []string{"access_control", "data_governance", "safety", "tool_allowlist", "model_allowlist", "compliance"}, Operators: []string{"=", "!="}},
						{Name: "language", DisplayName: "Language", Type: "select", Options: []string{"rego", "yaml_rules", "json_rules", "cel"}, Operators: []string{"=", "!="}},
						{Name: "enforcementScope", DisplayName: "Scope", Type: "select", Options: []string{"agent", "organization", "namespace", "project"}, Operators: []string{"=", "!="}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "policyType", DisplayName: "Policy Type", Path: "policyType", Type: "string", Section: "Overview"},
						{Name: "language", DisplayName: "Language", Path: "language", Type: "string", Section: "Overview"},
						{Name: "enforcementScope", DisplayName: "Enforcement Scope", Path: "enforcementScope", Type: "string", Section: "Overview"},
						{Name: "enforcementMode", DisplayName: "Enforcement Mode", Path: "enforcementMode", Type: "string", Section: "Overview"},
						{Name: "version", DisplayName: "Version", Path: "version", Type: "string", Section: "Overview"},
						{Name: "author", DisplayName: "Author", Path: "author", Type: "string", Section: "Overview"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Section: "Overview"},
						{Name: "bundleRef", DisplayName: "Bundle Reference", Path: "bundleRef", Type: "string", Section: "Bundle"},
						{Name: "entrypoint", DisplayName: "Entrypoint", Path: "entrypoint", Type: "string", Section: "Bundle"},
						{Name: "inputSchema", DisplayName: "Input Schema", Path: "inputSchema", Type: "object", Section: "Input Schema"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "gavel",
					NameField:      "name",
					DetailSections: []string{"Overview", "Bundle", "Input Schema"},
				},
				Actions: []string{"tag", "annotate", "deprecate"},
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
func (p *PolicyPlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"Policy"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
	}
}

// ListSources returns information about all configured sources.
func (p *PolicyPlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
	var result []plugin.SourceInfo

	for _, src := range p.cfg.Section.Sources {
		props := make(map[string]any, len(src.Properties))
		for k, v := range src.Properties {
			props[k] = v
		}

		// For YAML sources with a file path, read the file content.
		if yamlPath, ok := props["yamlCatalogPath"].(string); ok && yamlPath != "" {
			resolved := resolveSourcePath(src, yamlPath)
			if data, err := os.ReadFile(resolved); err == nil {
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

		// Get entity count from in-memory data.
		if src.IsEnabled() {
			info.Status.State = plugin.SourceStateAvailable
			p.mu.RLock()
			if entries, ok := p.sources[src.ID]; ok {
				info.Status.EntityCount = len(entries)
			}
			p.mu.RUnlock()
		}

		result = append(result, info)
	}

	return result, nil
}

// ValidateSource validates a source configuration without applying it.
func (p *PolicyPlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
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
	} else if src.Type != "yaml" {
		result.Valid = false
		result.Errors = append(result.Errors, plugin.ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("unsupported source type %q; only yaml is supported", src.Type),
		})
	}

	return result, nil
}

// ApplySource adds or updates a source configuration.
func (p *PolicyPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
	// Reload from YAML if properties include a path.
	srcCfg := plugin.SourceConfig{
		ID:         src.ID,
		Name:       src.Name,
		Type:       src.Type,
		Labels:     src.Labels,
		Properties: src.Properties,
	}
	if src.Enabled != nil {
		srcCfg.Enabled = src.Enabled
	}

	if src.Type == "yaml" {
		entries, err := loadYAMLSource(srcCfg)
		if err != nil {
			return fmt.Errorf("failed to load policy source: %w", err)
		}
		for i := range entries {
			entries[i].SourceId = src.ID
		}
		p.mu.Lock()
		p.sources[src.ID] = entries
		p.mu.Unlock()
	}

	return nil
}

// EnableSource enables or disables a source.
func (p *PolicyPlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !enabled {
		delete(p.sources, id)
	}
	return nil
}

// DeleteSource removes a source and its associated entities.
func (p *PolicyPlugin) DeleteSource(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.sources, id)
	return nil
}

// Refresh triggers a reload of a specific source.
func (p *PolicyPlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
	start := time.Now()

	// Find the source config.
	for _, src := range p.cfg.Section.Sources {
		if src.ID == sourceID {
			entries, err := loadYAMLSource(src)
			if err != nil {
				return &plugin.RefreshResult{
					SourceID: sourceID,
					Duration: time.Since(start),
					Error:    err.Error(),
				}, nil
			}
			for i := range entries {
				entries[i].SourceId = sourceID
			}
			p.mu.Lock()
			p.sources[sourceID] = entries
			p.mu.Unlock()

			return &plugin.RefreshResult{
				SourceID:       sourceID,
				EntitiesLoaded: len(entries),
				Duration:       time.Since(start),
			}, nil
		}
	}

	return &plugin.RefreshResult{
		SourceID: sourceID,
		Duration: time.Since(start),
		Error:    fmt.Sprintf("source %q not found", sourceID),
	}, nil
}

// RefreshAll triggers a reload of all sources.
func (p *PolicyPlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
	start := time.Now()
	totalLoaded := 0

	for _, src := range p.cfg.Section.Sources {
		if !src.IsEnabled() || src.Type != "yaml" {
			continue
		}
		entries, err := loadYAMLSource(src)
		if err != nil {
			p.logger.Error("failed to refresh source", "source", src.ID, "error", err)
			continue
		}
		for i := range entries {
			entries[i].SourceId = src.ID
		}
		p.mu.Lock()
		p.sources[src.ID] = entries
		p.mu.Unlock()
		totalLoaded += len(entries)
	}

	return &plugin.RefreshResult{
		EntitiesLoaded: totalLoaded,
		Duration:       time.Since(start),
	}, nil
}

// Diagnostics returns diagnostic information about the plugin.
func (p *PolicyPlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
	diag := &plugin.PluginDiagnostics{
		PluginName: PluginName,
		Sources:    make([]plugin.SourceDiagnostic, 0),
	}

	for _, src := range p.cfg.Section.Sources {
		sd := plugin.SourceDiagnostic{
			ID:    src.ID,
			Name:  src.Name,
			State: sourceState(src),
		}
		if src.IsEnabled() {
			p.mu.RLock()
			if entries, ok := p.sources[src.ID]; ok {
				sd.EntityCount = len(entries)
			}
			p.mu.RUnlock()
		}
		diag.Sources = append(diag.Sources, sd)
	}

	return diag, nil
}

// UIHints returns display hints for the UI.
func (p *PolicyPlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "policyType", Label: "Policy Type", Sortable: true, Filterable: true},
			{Field: "language", Label: "Language", Sortable: true, Filterable: true},
			{Field: "enforcementScope", Label: "Scope", Sortable: true, Filterable: true},
			{Field: "enforcementMode", Label: "Mode", Sortable: true, Filterable: true},
			{Field: "version", Label: "Version", Sortable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "policyType", Label: "Policy Type"},
			{Field: "language", Label: "Language"},
			{Field: "bundleRef", Label: "Bundle Reference"},
			{Field: "entrypoint", Label: "Entrypoint"},
			{Field: "enforcementScope", Label: "Enforcement Scope"},
			{Field: "enforcementMode", Label: "Enforcement Mode"},
			{Field: "version", Label: "Version"},
			{Field: "author", Label: "Author"},
			{Field: "license", Label: "License"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *PolicyPlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "policyType", "language", "enforcementScope", "enforcementMode"},
		SortField:        "name",
		FilterableFields: []string{"name", "policyType", "language", "enforcementScope", "enforcementMode"},
	}
}

// resolveSourcePath resolves a relative path against the source's origin directory.
func resolveSourcePath(src plugin.SourceConfig, path string) string {
	if path == "" {
		return path
	}
	if len(path) > 0 && (path[0] == '/' || (len(path) > 1 && path[1] == ':')) {
		return path
	}
	if src.Origin != "" {
		return fmt.Sprintf("%s/%s", dirOfPath(src.Origin), path)
	}
	return path
}

// dirOfPath returns the directory portion of a path.
func dirOfPath(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}

// sourceState returns the state string for a source.
func sourceState(src plugin.SourceConfig) string {
	if !src.IsEnabled() {
		return plugin.SourceStateDisabled
	}
	return plugin.SourceStateAvailable
}
