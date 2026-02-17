package guardrails

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*GuardrailPlugin)(nil)

// GetAssetMapper returns the asset mapper for the guardrails plugin.
func (p *GuardrailPlugin) GetAssetMapper() plugin.AssetMapper {
	return &guardrailAssetMapper{}
}

type guardrailAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *guardrailAssetMapper) SupportedKinds() []string {
	return []string{"Guardrail"}
}

// MapToAsset converts a single guardrail entity to an AssetResource.
func (m *guardrailAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case GuardrailEntry:
		return mapGuardrailToAsset(e), nil
	case *GuardrailEntry:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil GuardrailEntry pointer")
		}
		return mapGuardrailToAsset(*e), nil
	case map[string]any:
		return mapGuardrailMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for Guardrail mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *guardrailAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapGuardrailToAsset(e GuardrailEntry) plugin.AssetResource {
	spec := make(map[string]any)

	if e.GuardrailType != nil {
		spec["guardrailType"] = *e.GuardrailType
	}
	if e.EnforcementStage != nil {
		spec["enforcementStage"] = *e.EnforcementStage
	}
	if len(e.RiskCategories) > 0 {
		spec["riskCategories"] = e.RiskCategories
	}
	if e.EnforcementMode != nil {
		spec["enforcementMode"] = *e.EnforcementMode
	}
	if len(e.Modalities) > 0 {
		spec["modalities"] = e.Modalities
	}
	if len(e.ConfigRef) > 0 {
		spec["configRef"] = e.ConfigRef
	}

	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}

	asset := plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "Guardrail",
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

func mapGuardrailMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"guardrailType", "enforcementStage", "riskCategories",
		"enforcementMode", "modalities", "configRef",
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
		Kind:       "Guardrail",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
