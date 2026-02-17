package model

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	catalog "github.com/kubeflow/model-registry/catalog/internal/catalog"
	apimodels "github.com/kubeflow/model-registry/catalog/pkg/openapi"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertions.
var (
	_ plugin.SourceManager          = (*ModelCatalogPlugin)(nil)
	_ plugin.RefreshProvider        = (*ModelCatalogPlugin)(nil)
	_ plugin.DiagnosticsProvider    = (*ModelCatalogPlugin)(nil)
	_ plugin.CapabilitiesProvider   = (*ModelCatalogPlugin)(nil)
	_ plugin.CapabilitiesV2Provider = (*ModelCatalogPlugin)(nil)
	_ plugin.EntityGetter           = (*ModelCatalogPlugin)(nil)
	_ plugin.UIHintsProvider        = (*ModelCatalogPlugin)(nil)
	_ plugin.CLIHintsProvider       = (*ModelCatalogPlugin)(nil)
)

// Capabilities returns the plugin's advertised capabilities.
func (p *ModelCatalogPlugin) Capabilities() plugin.PluginCapabilities {
	return plugin.PluginCapabilities{
		EntityKinds:  []string{"CatalogModel"},
		ListEntities: true,
		GetEntity:    true,
		ListSources:  true,
		Artifacts:    true,
	}
}

// GetCapabilitiesV2 returns the full V2 capabilities discovery document.
func (p *ModelCatalogPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
	basePath := "/api/model_catalog/v1alpha1"
	return plugin.PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: plugin.PluginMeta{
			Name:        PluginName,
			Version:     PluginVersion,
			Description: "Model catalog for ML models",
			DisplayName: "Models",
			Icon:        "model",
		},
		Entities: []plugin.EntityCapabilities{
			{
				Kind:        "CatalogModel",
				Plural:      "models",
				DisplayName: "Model",
				Description: "Machine learning models",
				Endpoints: plugin.EntityEndpoints{
					List: basePath + "/models",
					Get:  basePath + "/sources/{source_id}/models/{name}",
				},
				Fields: plugin.EntityFields{
					Columns: []plugin.V2ColumnHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true, Width: "lg"},
						{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Sortable: true, Width: "md"},
						{Name: "task", DisplayName: "Task", Path: "tasks", Type: "string", Sortable: true, Width: "md"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Sortable: true, Width: "md"},
						{Name: "source_id", DisplayName: "Source", Path: "source_id", Type: "string", Sortable: true, Width: "md"},
					},
					FilterFields: []plugin.V2FilterField{
						{Name: "name", DisplayName: "Name", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "provider", DisplayName: "Provider", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "task", DisplayName: "Task", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "license", DisplayName: "License", Type: "text", Operators: []string{"=", "!=", "LIKE"}},
						{Name: "source_id", DisplayName: "Source", Type: "text", Operators: []string{"=", "!="}},
					},
					DetailFields: []plugin.V2FieldHint{
						{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
						{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
						{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Section: "Overview"},
						{Name: "tasks", DisplayName: "Tasks", Path: "tasks", Type: "string", Section: "Overview"},
						{Name: "license", DisplayName: "License", Path: "license", Type: "string", Section: "Overview"},
						{Name: "licenseLink", DisplayName: "License Link", Path: "licenseLink", Type: "string", Section: "Overview"},
						{Name: "maturity", DisplayName: "Maturity", Path: "maturity", Type: "string", Section: "Details"},
						{Name: "language", DisplayName: "Language", Path: "language", Type: "string", Section: "Details"},
						{Name: "libraryName", DisplayName: "Library", Path: "libraryName", Type: "string", Section: "Details"},
						{Name: "readme", DisplayName: "Readme", Path: "readme", Type: "string", Section: "Documentation"},
					},
				},
				UIHints: &plugin.EntityUIHints{
					Icon:           "model",
					NameField:      "name",
					DetailSections: []string{"Overview", "Details", "Documentation"},
				},
				Actions: []string{"tag", "annotate", "deprecate", "refresh"},
			},
		},
		Sources: &plugin.SourceCapabilities{
			Manageable:  true,
			Refreshable: true,
			Types:       []string{"yaml", "hf"},
		},
		Actions: []plugin.ActionDefinition{
			{ID: "tag", DisplayName: "Tag", Description: "Add or remove tags on an entity", Scope: "asset", SupportsDryRun: true, Idempotent: true},
			{ID: "annotate", DisplayName: "Annotate", Description: "Add or update annotations on an entity", Scope: "asset", SupportsDryRun: true, Idempotent: true},
			{ID: "deprecate", DisplayName: "Deprecate", Description: "Mark an entity as deprecated", Scope: "asset", SupportsDryRun: true, Idempotent: true},
			{ID: "refresh", DisplayName: "Refresh", Description: "Refresh entities from a source", Scope: "source", SupportsDryRun: false, Idempotent: true},
		},
	}
}

// ListSources returns information about all configured sources.
func (p *ModelCatalogPlugin) ListSources(ctx context.Context) ([]plugin.SourceInfo, error) {
	allSources := p.sources.AllSources()
	result := make([]plugin.SourceInfo, 0, len(allSources))

	for _, src := range allSources {
		enabled := src.Enabled != nil && *src.Enabled
		state := plugin.SourceStateAvailable
		if !enabled {
			state = plugin.SourceStateDisabled
		}

		info := plugin.SourceInfo{
			ID:         src.Id,
			Name:       src.Name,
			Type:       src.Type,
			Enabled:    enabled,
			Labels:     src.Labels,
			Properties: src.Properties,
			Status: plugin.SourceStatus{
				State: state,
			},
		}

		result = append(result, info)
	}

	return result, nil
}

// ValidateSource validates a source configuration without applying it.
func (p *ModelCatalogPlugin) ValidateSource(ctx context.Context, src plugin.SourceConfigInput) (*plugin.ValidationResult, error) {
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
	}

	return result, nil
}

// ApplySource adds or updates a source configuration.
func (p *ModelCatalogPlugin) ApplySource(ctx context.Context, src plugin.SourceConfigInput) error {
	enabled := true
	if src.Enabled != nil {
		enabled = *src.Enabled
	}

	catSource := apimodels.CatalogSource{
		Id:      src.ID,
		Name:    src.Name,
		Enabled: &enabled,
		Labels:  src.Labels,
	}

	source := catalog.Source{
		CatalogSource: catSource,
		Type:          src.Type,
		Properties:    src.Properties,
		Origin:        "api",
	}

	sources := map[string]catalog.Source{
		src.ID: source,
	}

	return p.sources.Merge("api", sources)
}

// EnableSource enables or disables a source.
func (p *ModelCatalogPlugin) EnableSource(ctx context.Context, id string, enabled bool) error {
	allSources := p.sources.AllSources()
	src, exists := allSources[id]
	if !exists {
		return fmt.Errorf("source %q not found", id)
	}

	src.Enabled = &enabled
	sources := map[string]catalog.Source{
		id: src,
	}

	return p.sources.Merge(src.Origin, sources)
}

// DeleteSource removes a source and its associated entities.
func (p *ModelCatalogPlugin) DeleteSource(ctx context.Context, id string) error {
	// Remove entities from the database
	if err := p.services.CatalogModelRepository.DeleteBySource(id); err != nil {
		return fmt.Errorf("failed to delete entities for source %q: %w", id, err)
	}

	// Disable the source so it won't be loaded again
	disabled := false
	allSources := p.sources.AllSources()
	if src, exists := allSources[id]; exists {
		src.Enabled = &disabled
		sources := map[string]catalog.Source{
			id: src,
		}
		return p.sources.Merge(src.Origin, sources)
	}

	return nil
}

// Refresh triggers a reload of a specific source.
func (p *ModelCatalogPlugin) Refresh(ctx context.Context, sourceID string) (*plugin.RefreshResult, error) {
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
func (p *ModelCatalogPlugin) RefreshAll(ctx context.Context) (*plugin.RefreshResult, error) {
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
func (p *ModelCatalogPlugin) Diagnostics(ctx context.Context) (*plugin.PluginDiagnostics, error) {
	diag := &plugin.PluginDiagnostics{
		PluginName: PluginName,
		Sources:    make([]plugin.SourceDiagnostic, 0),
	}

	allSources := p.sources.AllSources()
	for _, src := range allSources {
		enabled := src.Enabled != nil && *src.Enabled
		state := plugin.SourceStateAvailable
		if !enabled {
			state = plugin.SourceStateDisabled
		}

		sd := plugin.SourceDiagnostic{
			ID:    src.Id,
			Name:  src.Name,
			State: state,
		}
		diag.Sources = append(diag.Sources, sd)
	}

	return diag, nil
}

// UIHints returns display hints for the UI.
func (p *ModelCatalogPlugin) UIHints() plugin.UIHints {
	return plugin.UIHints{
		IdentityField:    "name",
		DisplayNameField: "name",
		DescriptionField: "description",
		ListColumns: []plugin.ColumnHint{
			{Field: "name", Label: "Name", Sortable: true, Filterable: true},
			{Field: "provider", Label: "Provider", Sortable: true, Filterable: true},
			{Field: "task", Label: "Task", Sortable: true, Filterable: true},
			{Field: "source_id", Label: "Source", Sortable: true, Filterable: true},
		},
		DetailFields: []plugin.FieldHint{
			{Field: "name", Label: "Name"},
			{Field: "description", Label: "Description"},
			{Field: "provider", Label: "Provider"},
			{Field: "task", Label: "Task"},
			{Field: "license", Label: "License"},
			{Field: "source_id", Label: "Source"},
		},
	}
}

// CLIHints returns display hints for the CLI.
func (p *ModelCatalogPlugin) CLIHints() plugin.CLIHints {
	return plugin.CLIHints{
		DefaultColumns:   []string{"name", "provider", "task", "source_id"},
		SortField:        "name",
		FilterableFields: []string{"name", "provider", "task", "license", "source_id"},
	}
}

// GetEntityByName retrieves a model by name, searching across all sources.
// This enables the standardized management GET /entities/{name} endpoint for
// the model plugin, whose native get endpoint requires both source_id and name.
func (p *ModelCatalogPlugin) GetEntityByName(ctx context.Context, entityKind string, name string) (map[string]any, error) {
	// Use the list endpoint with a name filter to find the model.
	models, err := p.dbCatalog.ListModels(ctx, catalog.ListModelsParams{
		FilterQuery: "name='" + name + "'",
		PageSize:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search for model %q: %w", name, err)
	}

	if len(models.Items) == 0 {
		return nil, nil
	}

	// Convert the first matching model to a generic map.
	model := models.Items[0]
	data, err := json.Marshal(model)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal model: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model: %w", err)
	}

	return result, nil
}
