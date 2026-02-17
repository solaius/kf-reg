package skills

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*SkillPlugin)(nil)
	_ plugin.RefreshProvider        = (*SkillPlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*SkillPlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*SkillPlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*SkillPlugin)(nil)
	_ plugin.UIHintsProvider        = (*SkillPlugin)(nil)
	_ plugin.CLIHintsProvider       = (*SkillPlugin)(nil)
	_ plugin.BasePathProvider       = (*SkillPlugin)(nil)
)

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *SkillPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/skills_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "Skill catalog for tools, operations, and executable actions",
			DisplayName: "Skills",
			Icon:        "wrench",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "Skill",
				Plural:      "skills",
				DisplayName: "Skill",
				Description: "Tools, operations, and executable actions",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/skills",
					Get:  basePath + "/skills/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "skillType", DisplayName: "Type", Path: "skillType", Type: "string", Sortable: true, Width: "md"},
						{Name: "riskLevel", DisplayName: "Risk Level", Path: "safety.riskLevel", Type: "string", Sortable: true, Width: "sm"},
						{Name: "version", DisplayName: "Version", Path: "version", Type: "string", Sortable: true, Width: "sm"},
						{Name: "author", DisplayName: "Author", Path: "author", Type: "string", Sortable: true, Width: "md"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "skillType", DisplayName: "Skill Type", Type: "select", Options: []string{"mcp_tool", "openapi_operation", "k8s_action", "shell_command"}, Operators: []string{"=", "!="}},
						{Name: "riskLevel", DisplayName: "Risk Level", Type: "select", Options: []string{"low", "medium", "high"}, Operators: []string{"=", "!="}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "skillType", DisplayName: "Skill Type", Path: "skillType", Type: "string", Section: "Overview"},
						{Name: "version", DisplayName: "Version", Path: "version", Type: "string", Section: "Overview"},
						{Name: "author", DisplayName: "Author", Path: "author", Type: "string", Section: "Overview"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Section: "Overview"},
						{Name: "inputSchema", DisplayName: "Input Schema", Path: "inputSchema", Type: "object", Section: "Input/Output Schema"},
						{Name: "outputSchema", DisplayName: "Output Schema", Path: "outputSchema", Type: "object", Section: "Input/Output Schema"},
						{Name: "execution", DisplayName: "Execution", Path: "execution", Type: "object", Section: "Execution"},
						{Name: "safety", DisplayName: "Safety", Path: "safety", Type: "object", Section: "Safety & Constraints"},
						{Name: "rateLimit", DisplayName: "Rate Limit", Path: "rateLimit", Type: "object", Section: "Safety & Constraints"},
						{Name: "timeoutSeconds", DisplayName: "Timeout (seconds)", Path: "timeoutSeconds", Type: "integer", Section: "Safety & Constraints"},
						{Name: "retryPolicy", DisplayName: "Retry Policy", Path: "retryPolicy", Type: "object", Section: "Safety & Constraints"},
						{Name: "compatibility", DisplayName: "Compatibility", Path: "compatibility", Type: "object", Section: "Safety & Constraints"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "wrench",
					NameField:      "name",
					DetailSections: []string{"Overview", "Input/Output Schema", "Execution", "Safety & Constraints"},
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
func (p *SkillPlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"Skill"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
	}
}

// ListSources returns information about all configured sources.
func (p *SkillPlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
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
func (p *SkillPlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
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
func (p *SkillPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
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
			return fmt.Errorf("failed to load skill source: %w", err)
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
func (p *SkillPlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !enabled {
		delete(p.sources, id)
	}
	return nil
}

// DeleteSource removes a source and its associated entities.
func (p *SkillPlugin) DeleteSource(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.sources, id)
	return nil
}

// Refresh triggers a reload of a specific source.
func (p *SkillPlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
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
func (p *SkillPlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
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
func (p *SkillPlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
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
func (p *SkillPlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "skillType", Label: "Type", Sortable: true, Filterable: true},
			{Field: "version", Label: "Version", Sortable: true},
			{Field: "author", Label: "Author", Sortable: true, Filterable: true},
			{Field: "license", Label: "License", Sortable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "skillType", Label: "Skill Type"},
			{Field: "version", Label: "Version"},
			{Field: "author", Label: "Author"},
			{Field: "license", Label: "License"},
			{Field: "inputSchema", Label: "Input Schema"},
			{Field: "outputSchema", Label: "Output Schema"},
			{Field: "execution", Label: "Execution"},
			{Field: "safety", Label: "Safety"},
			{Field: "rateLimit", Label: "Rate Limit"},
			{Field: "timeoutSeconds", Label: "Timeout (seconds)"},
			{Field: "retryPolicy", Label: "Retry Policy"},
			{Field: "compatibility", Label: "Compatibility"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *SkillPlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "skillType", "version", "author", "license"},
		SortField:        "name",
		FilterableFields: []string{"name", "skillType", "version", "author", "license"},
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
