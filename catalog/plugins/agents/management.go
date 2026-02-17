package agents

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*AgentPlugin)(nil)
	_ plugin.RefreshProvider        = (*AgentPlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*AgentPlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*AgentPlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*AgentPlugin)(nil)
	_ plugin.UIHintsProvider        = (*AgentPlugin)(nil)
	_ plugin.CLIHintsProvider       = (*AgentPlugin)(nil)
	_ plugin.BasePathProvider       = (*AgentPlugin)(nil)
)

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *AgentPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/agents_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "Agent catalog for AI agents and multi-agent orchestrations",
			DisplayName: "Agents",
			Icon:        "robot",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "Agent",
				Plural:      "agents",
				DisplayName: "Agent",
				Description: "AI agents and multi-agent orchestrations",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/agents",
					Get:  basePath + "/agents/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "agentType", DisplayName: "Type", Path: "agentType", Type: "string", Sortable: true, Width: "md"},
						{Name: "category", DisplayName: "Category", Path: "category", Type: "string", Sortable: true, Width: "md"},
						{Name: "version", DisplayName: "Version", Path: "version", Type: "string", Sortable: true, Width: "sm"},
						{Name: "author", DisplayName: "Author", Path: "author", Type: "string", Sortable: true, Width: "md"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "agentType", DisplayName: "Agent Type", Type: "select", Options: []string{"conversational", "task_oriented", "router", "planner", "executor", "evaluator"}, Operators: []string{"=", "!="}},
						{Name: "category", DisplayName: "Category", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "agentType", DisplayName: "Agent Type", Path: "agentType", Type: "string", Section: "Overview"},
						{Name: "category", DisplayName: "Category", Path: "category", Type: "string", Section: "Overview"},
						{Name: "version", DisplayName: "Version", Path: "version", Type: "string", Section: "Overview"},
						{Name: "author", DisplayName: "Author", Path: "author", Type: "string", Section: "Overview"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Section: "Overview"},
						{Name: "instructions", DisplayName: "Instructions", Path: "instructions", Type: "string", Section: "Instructions"},
						{Name: "modelConfig", DisplayName: "Model Configuration", Path: "modelConfig", Type: "object", Section: "Model Configuration"},
						{Name: "tools", DisplayName: "Tools", Path: "tools", Type: "array", Section: "Tools & Knowledge"},
						{Name: "knowledge", DisplayName: "Knowledge", Path: "knowledge", Type: "array", Section: "Tools & Knowledge"},
						{Name: "promptRefs", DisplayName: "Prompt References", Path: "promptRefs", Type: "array", Section: "Tools & Knowledge"},
						{Name: "guardrails", DisplayName: "Guardrails", Path: "guardrails", Type: "array", Section: "Guardrails & Policies"},
						{Name: "policies", DisplayName: "Policies", Path: "policies", Type: "array", Section: "Guardrails & Policies"},
						{Name: "dependencies", DisplayName: "Dependencies", Path: "dependencies", Type: "array", Section: "Dependencies"},
						{Name: "inputSchema", DisplayName: "Input Schema", Path: "inputSchema", Type: "object", Section: "Input/Output"},
						{Name: "outputSchema", DisplayName: "Output Schema", Path: "outputSchema", Type: "object", Section: "Input/Output"},
						{Name: "examples", DisplayName: "Examples", Path: "examples", Type: "array", Section: "Input/Output"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "robot",
					NameField:      "name",
					DetailSections: []string{"Overview", "Instructions", "Model Configuration", "Tools & Knowledge", "Guardrails & Policies", "Dependencies", "Input/Output"},
				},
				Actions: []string{"tag", "annotate", "deprecate"},
			},
		},
		Sources: &plugin.SourceCapabilities{
			Manageable:  true,
			Refreshable: true,
			Types:       []string{"yaml", "git"},
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
func (p *AgentPlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"Agent"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
	}
}

// ListSources returns information about all configured sources.
func (p *AgentPlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
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
func (p *AgentPlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
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
	} else if src.Type != "yaml" && src.Type != "git" {
		result.Valid = false
		result.Errors = append(result.Errors, plugin.ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("unsupported source type %q; supported types are yaml and git", src.Type),
		})
	}

	return result, nil
}

// ApplySource adds or updates a source configuration.
func (p *AgentPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
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
			return fmt.Errorf("failed to load agent source: %w", err)
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
func (p *AgentPlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !enabled {
		delete(p.sources, id)
	}
	return nil
}

// DeleteSource removes a source and its associated entities.
func (p *AgentPlugin) DeleteSource(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.sources, id)
	return nil
}

// Refresh triggers a reload of a specific source.
func (p *AgentPlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
	start := time.Now()

	// Find the source config.
	for _, src := range p.cfg.Section.Sources {
		if src.ID == sourceID {
			entries, err := p.refreshSource(ctx, src)
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
func (p *AgentPlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
	start := time.Now()
	totalLoaded := 0

	for _, src := range p.cfg.Section.Sources {
		if !src.IsEnabled() {
			continue
		}
		if src.Type != "yaml" && src.Type != "git" {
			continue
		}
		entries, err := p.refreshSource(ctx, src)
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

// refreshSource dispatches to the appropriate loader based on source type.
func (p *AgentPlugin) refreshSource(ctx context.Context, src plugin.SourceConfig) ([]AgentEntry, error) {
	switch src.Type {
	case "yaml":
		return loadYAMLSource(src)
	case "git":
		// Cancel the previous git provider goroutine for this source if one exists.
		if cancel, ok := p.gitCancels[src.ID]; ok {
			cancel()
			delete(p.gitCancels, src.ID)
		}
		entries, cancel, err := loadGitSource(ctx, src, p.logger)
		if err != nil {
			return nil, err
		}
		if cancel != nil {
			p.gitCancels[src.ID] = cancel
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("unsupported source type %q", src.Type)
	}
}

// Diagnostics returns diagnostic information about the plugin.
func (p *AgentPlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
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
func (p *AgentPlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "agentType", Label: "Type", Sortable: true, Filterable: true},
			{Field: "category", Label: "Category", Sortable: true, Filterable: true},
			{Field: "version", Label: "Version", Sortable: true},
			{Field: "author", Label: "Author", Sortable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "agentType", Label: "Agent Type"},
			{Field: "instructions", Label: "Instructions"},
			{Field: "version", Label: "Version"},
			{Field: "category", Label: "Category"},
			{Field: "author", Label: "Author"},
			{Field: "license", Label: "License"},
			{Field: "modelConfig", Label: "Model Configuration"},
			{Field: "tools", Label: "Tools"},
			{Field: "knowledge", Label: "Knowledge"},
			{Field: "guardrails", Label: "Guardrails"},
			{Field: "policies", Label: "Policies"},
			{Field: "promptRefs", Label: "Prompt References"},
			{Field: "dependencies", Label: "Dependencies"},
			{Field: "inputSchema", Label: "Input Schema"},
			{Field: "outputSchema", Label: "Output Schema"},
			{Field: "examples", Label: "Examples"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *AgentPlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "agentType", "category", "version", "author"},
		SortField:        "name",
		FilterableFields: []string{"name", "agentType", "category", "version", "author"},
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
