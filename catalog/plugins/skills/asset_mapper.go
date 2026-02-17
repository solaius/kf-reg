package skills

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*SkillPlugin)(nil)

// GetAssetMapper returns the asset mapper for the skills plugin.
func (p *SkillPlugin) GetAssetMapper() plugin.AssetMapper {
	return &skillAssetMapper{}
}

type skillAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *skillAssetMapper) SupportedKinds() []string {
	return []string{"Skill"}
}

// MapToAsset converts a single skill entity to an AssetResource.
func (m *skillAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case SkillEntry:
		return mapSkillToAsset(e), nil
	case *SkillEntry:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil SkillEntry pointer")
		}
		return mapSkillToAsset(*e), nil
	case map[string]any:
		return mapSkillMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for Skill mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *skillAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapSkillToAsset(e SkillEntry) plugin.AssetResource {
	spec := make(map[string]any)

	if e.SkillType != nil {
		spec["skillType"] = *e.SkillType
	}
	if e.InputSchema != nil {
		spec["inputSchema"] = e.InputSchema
	}
	if e.OutputSchema != nil {
		spec["outputSchema"] = e.OutputSchema
	}
	if e.Execution != nil {
		spec["execution"] = e.Execution
	}
	if e.Safety != nil {
		spec["safety"] = e.Safety
	}
	if e.RateLimit != nil {
		spec["rateLimit"] = e.RateLimit
	}
	if e.TimeoutSeconds != nil {
		spec["timeoutSeconds"] = *e.TimeoutSeconds
	}
	if e.RetryPolicy != nil {
		spec["retryPolicy"] = e.RetryPolicy
	}
	if e.Compatibility != nil {
		spec["compatibility"] = e.Compatibility
	}

	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}

	asset := plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "Skill",
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

func mapSkillMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"skillType", "inputSchema", "outputSchema", "execution",
		"safety", "rateLimit", "timeoutSeconds", "retryPolicy", "compatibility",
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
		Kind:       "Skill",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
