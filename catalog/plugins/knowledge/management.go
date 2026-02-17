package knowledge

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*KnowledgeSourcePlugin)(nil)
	_ plugin.RefreshProvider        = (*KnowledgeSourcePlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*KnowledgeSourcePlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*KnowledgeSourcePlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*KnowledgeSourcePlugin)(nil)
	_ plugin.UIHintsProvider        = (*KnowledgeSourcePlugin)(nil)
	_ plugin.CLIHintsProvider       = (*KnowledgeSourcePlugin)(nil)
	_ plugin.BasePathProvider       = (*KnowledgeSourcePlugin)(nil)
)

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *KnowledgeSourcePlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/knowledge_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "Knowledge source catalog for documents, vector stores, and graph stores",
			DisplayName: "Knowledge Sources",
			Icon:        "database",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "KnowledgeSource",
				Plural:      "knowledgesources",
				DisplayName: "Knowledge Source",
				Description: "Documents, vector stores, and graph stores",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/knowledgesources",
					Get:  basePath + "/knowledgesources/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "sourceType", DisplayName: "Type", Path: "sourceType", Type: "string", Sortable: true, Width: "md"},
						{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Sortable: true, Width: "md"},
						{Name: "status", DisplayName: "Status", Path: "status", Type: "string", Sortable: true, Width: "sm"},
						{Name: "documentCount", DisplayName: "Documents", Path: "documentCount", Type: "integer", Sortable: true, Width: "sm"},
						{Name: "contentType", DisplayName: "Content Type", Path: "contentType", Type: "string", Width: "md"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "sourceType", DisplayName: "Source Type", Type: "select", Options: []string{"document", "url", "vector_store", "graph_store"}, Operators: []string{"=", "!="}},
						{Name: "provider", DisplayName: "Provider", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "status", DisplayName: "Status", Type: "select", Options: []string{"active", "indexing", "error", "archived"}, Operators: []string{"=", "!="}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "sourceType", DisplayName: "Source Type", Path: "sourceType", Type: "string", Section: "Overview"},
						{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Section: "Overview"},
						{Name: "status", DisplayName: "Status", Path: "status", Type: "string", Section: "Overview"},
						{Name: "location", DisplayName: "Location", Path: "location", Type: "string", Section: "Connection"},
						{Name: "contentType", DisplayName: "Content Type", Path: "contentType", Type: "string", Section: "Connection"},
						{Name: "indexType", DisplayName: "Index Type", Path: "indexType", Type: "string", Section: "Connection"},
						{Name: "documentCount", DisplayName: "Document Count", Path: "documentCount", Type: "integer", Section: "Statistics"},
						{Name: "vectorDimensions", DisplayName: "Vector Dimensions", Path: "vectorDimensions", Type: "integer", Section: "Statistics"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "database",
					NameField:      "name",
					DetailSections: []string{"Overview", "Connection", "Statistics"},
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
func (p *KnowledgeSourcePlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"KnowledgeSource"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
	}
}

// ListSources returns information about all configured sources.
func (p *KnowledgeSourcePlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
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
func (p *KnowledgeSourcePlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
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
func (p *KnowledgeSourcePlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
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
			return fmt.Errorf("failed to load knowledge source: %w", err)
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
func (p *KnowledgeSourcePlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !enabled {
		delete(p.sources, id)
	}
	return nil
}

// DeleteSource removes a source and its associated entities.
func (p *KnowledgeSourcePlugin) DeleteSource(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.sources, id)
	return nil
}

// Refresh triggers a reload of a specific source.
func (p *KnowledgeSourcePlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
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
func (p *KnowledgeSourcePlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
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
func (p *KnowledgeSourcePlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
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
func (p *KnowledgeSourcePlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "sourceType", Label: "Type", Sortable: true, Filterable: true},
			{Field: "provider", Label: "Provider", Sortable: true, Filterable: true},
			{Field: "status", Label: "Status", Sortable: true, Filterable: true},
			{Field: "documentCount", Label: "Documents", Sortable: true},
			{Field: "contentType", Label: "Content Type", Sortable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "sourceType", Label: "Source Type"},
			{Field: "location", Label: "Location"},
			{Field: "contentType", Label: "Content Type"},
			{Field: "provider", Label: "Provider"},
			{Field: "status", Label: "Status"},
			{Field: "documentCount", Label: "Document Count"},
			{Field: "vectorDimensions", Label: "Vector Dimensions"},
			{Field: "indexType", Label: "Index Type"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *KnowledgeSourcePlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "sourceType", "provider", "status", "documentCount"},
		SortField:        "name",
		FilterableFields: []string{"name", "sourceType", "provider", "status", "contentType"},
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
