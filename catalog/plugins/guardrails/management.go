package guardrails

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*GuardrailPlugin)(nil)
	_ plugin.RefreshProvider        = (*GuardrailPlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*GuardrailPlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*GuardrailPlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*GuardrailPlugin)(nil)
	_ plugin.UIHintsProvider        = (*GuardrailPlugin)(nil)
	_ plugin.CLIHintsProvider       = (*GuardrailPlugin)(nil)
	_ plugin.BasePathProvider       = (*GuardrailPlugin)(nil)
)

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *GuardrailPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/guardrails_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "Guardrail catalog for AI safety and content moderation rules",
			DisplayName: "Guardrails",
			Icon:        "shield",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "Guardrail",
				Plural:      "guardrails",
				DisplayName: "Guardrail",
				Description: "AI safety and content moderation rules",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/guardrails",
					Get:  basePath + "/guardrails/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "guardrailType", DisplayName: "Type", Path: "guardrailType", Type: "string", Sortable: true, Width: "md"},
						{Name: "enforcementStage", DisplayName: "Stage", Path: "enforcementStage", Type: "string", Sortable: true, Width: "md"},
						{Name: "enforcementMode", DisplayName: "Mode", Path: "enforcementMode", Type: "string", Sortable: true, Width: "sm"},
						{Name: "riskCategories", DisplayName: "Risk Categories", Path: "riskCategories", Type: "array", Width: "md"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "guardrailType", DisplayName: "Guardrail Type", Type: "select", Options: []string{"nemo_guardrails", "guardrails_ai", "regex_rules", "content_filter", "moderation_profile"}, Operators: []string{"=", "!="}},
						{Name: "enforcementStage", DisplayName: "Enforcement Stage", Type: "select", Options: []string{"pre_prompt", "post_generation", "tool_use", "retrieval", "output_format"}, Operators: []string{"=", "!="}},
						{Name: "enforcementMode", DisplayName: "Enforcement Mode", Type: "select", Options: []string{"advisory", "required"}, Operators: []string{"=", "!="}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "guardrailType", DisplayName: "Guardrail Type", Path: "guardrailType", Type: "string", Section: "Overview"},
						{Name: "enforcementStage", DisplayName: "Enforcement Stage", Path: "enforcementStage", Type: "string", Section: "Overview"},
						{Name: "enforcementMode", DisplayName: "Enforcement Mode", Path: "enforcementMode", Type: "string", Section: "Overview"},
						{Name: "version", DisplayName: "Version", Path: "version", Type: "string", Section: "Overview"},
						{Name: "author", DisplayName: "Author", Path: "author", Type: "string", Section: "Overview"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Section: "Overview"},
						{Name: "riskCategories", DisplayName: "Risk Categories", Path: "riskCategories", Type: "array", Section: "Risk Categories"},
						{Name: "modalities", DisplayName: "Modalities", Path: "modalities", Type: "array", Section: "Risk Categories"},
						{Name: "configRef", DisplayName: "Configuration Reference", Path: "configRef", Type: "object", Section: "Configuration"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "shield",
					NameField:      "name",
					DetailSections: []string{"Overview", "Risk Categories", "Configuration"},
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
func (p *GuardrailPlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"Guardrail"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
	}
}

// ListSources returns information about all configured sources.
func (p *GuardrailPlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
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
func (p *GuardrailPlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
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
func (p *GuardrailPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
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
			return fmt.Errorf("failed to load guardrails source: %w", err)
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
func (p *GuardrailPlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	// Update the in-memory config so ListSources reflects the new state.
	found := false
	var srcCfg plugin.SourceConfig
	for i, src := range p.cfg.Section.Sources {
		if src.ID == id {
			p.cfg.Section.Sources[i].Enabled = &enabled
			srcCfg = p.cfg.Section.Sources[i]
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("source %q not found", id)
	}

	if !enabled {
		p.mu.Lock()
		delete(p.sources, id)
		p.mu.Unlock()
		return nil
	}

	// Re-enable: reload entities from the source.
	entries, err := loadYAMLSource(srcCfg)
	if err != nil {
		return fmt.Errorf("failed to reload source %q: %w", id, err)
	}
	for i := range entries {
		entries[i].SourceId = id
	}
	p.mu.Lock()
	p.sources[id] = entries
	p.mu.Unlock()
	return nil
}

// DeleteSource removes a source and its associated entities.
func (p *GuardrailPlugin) DeleteSource(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.sources, id)
	return nil
}

// Refresh triggers a reload of a specific source.
func (p *GuardrailPlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
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
func (p *GuardrailPlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
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
func (p *GuardrailPlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
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
func (p *GuardrailPlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "guardrailType", Label: "Type", Sortable: true, Filterable: true},
			{Field: "enforcementStage", Label: "Stage", Sortable: true, Filterable: true},
			{Field: "enforcementMode", Label: "Mode", Sortable: true, Filterable: true},
			{Field: "riskCategories", Label: "Risk Categories"},
			{Field: "version", Label: "Version", Sortable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "guardrailType", Label: "Guardrail Type"},
			{Field: "enforcementStage", Label: "Enforcement Stage"},
			{Field: "enforcementMode", Label: "Enforcement Mode"},
			{Field: "riskCategories", Label: "Risk Categories"},
			{Field: "modalities", Label: "Modalities"},
			{Field: "version", Label: "Version"},
			{Field: "author", Label: "Author"},
			{Field: "license", Label: "License"},
			{Field: "configRef", Label: "Configuration Reference"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *GuardrailPlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "guardrailType", "enforcementStage", "enforcementMode", "riskCategories"},
		SortField:        "name",
		FilterableFields: []string{"name", "guardrailType", "enforcementStage", "enforcementMode"},
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
