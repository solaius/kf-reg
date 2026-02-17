package policies

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*PolicyPlugin)(nil)

// GetAssetMapper returns the asset mapper for the policies plugin.
func (p *PolicyPlugin) GetAssetMapper() plugin.AssetMapper {
	return &policyAssetMapper{}
}

type policyAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *policyAssetMapper) SupportedKinds() []string {
	return []string{"Policy"}
}

// MapToAsset converts a single policy entity to an AssetResource.
func (m *policyAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case PolicyEntry:
		return mapPolicyToAsset(e), nil
	case *PolicyEntry:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil PolicyEntry pointer")
		}
		return mapPolicyToAsset(*e), nil
	case map[string]any:
		return mapPolicyMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for Policy mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *policyAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapPolicyToAsset(e PolicyEntry) plugin.AssetResource {
	spec := make(map[string]any)

	if e.PolicyType != nil {
		spec["policyType"] = *e.PolicyType
	}
	if e.Language != nil {
		spec["language"] = *e.Language
	}
	if e.BundleRef != nil {
		spec["bundleRef"] = *e.BundleRef
	}
	if e.Entrypoint != nil {
		spec["entrypoint"] = *e.Entrypoint
	}
	if e.EnforcementScope != nil {
		spec["enforcementScope"] = *e.EnforcementScope
	}
	if e.EnforcementMode != nil {
		spec["enforcementMode"] = *e.EnforcementMode
	}
	if e.InputSchema != nil {
		spec["inputSchema"] = e.InputSchema
	}

	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}

	asset := plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "Policy",
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

func mapPolicyMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"policyType", "language", "bundleRef", "entrypoint",
		"enforcementScope", "enforcementMode", "inputSchema",
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
		Kind:       "Policy",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
