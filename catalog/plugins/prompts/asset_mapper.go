package prompts

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*PromptTemplatePlugin)(nil)

// GetAssetMapper returns the asset mapper for the prompts plugin.
func (p *PromptTemplatePlugin) GetAssetMapper() plugin.AssetMapper {
	return &promptTemplateAssetMapper{}
}

type promptTemplateAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *promptTemplateAssetMapper) SupportedKinds() []string {
	return []string{"PromptTemplate"}
}

// MapToAsset converts a single prompt template entity to an AssetResource.
func (m *promptTemplateAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case PromptTemplateEntry:
		return mapPromptTemplateToAsset(e), nil
	case *PromptTemplateEntry:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil PromptTemplateEntry pointer")
		}
		return mapPromptTemplateToAsset(*e), nil
	case map[string]any:
		return mapPromptTemplateMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for PromptTemplate mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *promptTemplateAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapPromptTemplateToAsset(e PromptTemplateEntry) plugin.AssetResource {
	spec := make(map[string]any)

	if e.Format != nil {
		spec["format"] = *e.Format
	}
	if e.Template != nil {
		spec["template"] = *e.Template
	}
	if e.ParametersSchema != nil {
		spec["parametersSchema"] = e.ParametersSchema
	}
	if e.OutputSchema != nil {
		spec["outputSchema"] = e.OutputSchema
	}
	if e.ModelConstraints != nil {
		spec["modelConstraints"] = e.ModelConstraints
	}
	if e.Examples != nil {
		spec["examples"] = e.Examples
	}
	if e.TaskTags != nil {
		spec["taskTags"] = e.TaskTags
	}
	if e.Version != nil {
		spec["version"] = *e.Version
	}
	if e.Author != nil {
		spec["author"] = *e.Author
	}
	if e.License != nil {
		spec["license"] = *e.License
	}

	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}

	asset := plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "PromptTemplate",
		Metadata: plugin.AssetMetadata{
			Name:        e.Name,
			Description: desc,
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}

	if e.SourceId != "" {
		asset.Metadata.SourceRef = &plugin.SourceRef{
			SourceID: e.SourceId,
		}
	}

	return asset
}

func mapPromptTemplateMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"format", "template", "parametersSchema", "outputSchema",
		"modelConstraints", "examples", "taskTags", "version",
		"author", "license",
	} {
		if v, ok := m[key]; ok {
			spec[key] = v
		}
	}

	getString := func(key string) string {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	return plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "PromptTemplate",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
