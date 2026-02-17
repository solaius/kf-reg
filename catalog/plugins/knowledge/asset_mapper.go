package knowledge

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*KnowledgeSourcePlugin)(nil)

// GetAssetMapper returns the asset mapper for the knowledge plugin.
func (p *KnowledgeSourcePlugin) GetAssetMapper() plugin.AssetMapper {
	return &knowledgeAssetMapper{}
}

type knowledgeAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *knowledgeAssetMapper) SupportedKinds() []string {
	return []string{"KnowledgeSource"}
}

// MapToAsset converts a single knowledge source entity to an AssetResource.
func (m *knowledgeAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case KnowledgeSourceEntry:
		return mapKnowledgeSourceToAsset(e), nil
	case *KnowledgeSourceEntry:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil KnowledgeSourceEntry pointer")
		}
		return mapKnowledgeSourceToAsset(*e), nil
	case map[string]any:
		return mapKnowledgeSourceMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for KnowledgeSource mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *knowledgeAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapKnowledgeSourceToAsset(e KnowledgeSourceEntry) plugin.AssetResource {
	spec := make(map[string]any)

	if e.SourceType != nil {
		spec["sourceType"] = *e.SourceType
	}
	if e.Location != nil {
		spec["location"] = *e.Location
	}
	if e.ContentType != nil {
		spec["contentType"] = *e.ContentType
	}
	if e.Provider != nil {
		spec["provider"] = *e.Provider
	}
	if e.Status != nil {
		spec["status"] = *e.Status
	}
	if e.DocumentCount != nil {
		spec["documentCount"] = *e.DocumentCount
	}
	if e.VectorDimensions != nil {
		spec["vectorDimensions"] = *e.VectorDimensions
	}
	if e.IndexType != nil {
		spec["indexType"] = *e.IndexType
	}

	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}

	asset := plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "KnowledgeSource",
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

func mapKnowledgeSourceMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"sourceType", "location", "contentType", "provider",
		"status", "documentCount", "vectorDimensions", "indexType",
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
		Kind:       "KnowledgeSource",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
