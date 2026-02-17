package model

import (
	"fmt"

	apimodels "github.com/kubeflow/model-registry/catalog/pkg/openapi"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*ModelCatalogPlugin)(nil)

// GetAssetMapper returns the asset mapper for the model plugin.
func (p *ModelCatalogPlugin) GetAssetMapper() plugin.AssetMapper {
	return &modelAssetMapper{}
}

type modelAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *modelAssetMapper) SupportedKinds() []string {
	return []string{"CatalogModel"}
}

// MapToAsset converts a single catalog model entity to an AssetResource.
func (m *modelAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case apimodels.CatalogModel:
		return mapCatalogModelToAsset(e), nil
	case *apimodels.CatalogModel:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil CatalogModel pointer")
		}
		return mapCatalogModelToAsset(*e), nil
	case map[string]any:
		return mapCatalogModelMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for CatalogModel mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *modelAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapCatalogModelToAsset(cm apimodels.CatalogModel) plugin.AssetResource {
	spec := make(map[string]any)

	if cm.Provider != nil {
		spec["provider"] = *cm.Provider
	}
	if cm.License != nil {
		spec["license"] = *cm.License
	}
	if cm.LicenseLink != nil {
		spec["licenseLink"] = *cm.LicenseLink
	}
	if cm.Maturity != nil {
		spec["maturity"] = *cm.Maturity
	}
	if cm.LibraryName != nil {
		spec["libraryName"] = *cm.LibraryName
	}
	if cm.Logo != nil {
		spec["logo"] = *cm.Logo
	}
	if cm.Readme != nil {
		spec["readme"] = *cm.Readme
	}
	if len(cm.Tasks) > 0 {
		spec["tasks"] = cm.Tasks
	}
	if len(cm.Language) > 0 {
		spec["language"] = cm.Language
	}
	if cm.SourceId != nil {
		spec["source_id"] = *cm.SourceId
	}

	var description string
	if cm.Description != nil {
		description = *cm.Description
	}

	var uid string
	if cm.Id != nil {
		uid = *cm.Id
	}

	var createdAt string
	if cm.CreateTimeSinceEpoch != nil {
		createdAt = *cm.CreateTimeSinceEpoch
	}

	var updatedAt string
	if cm.LastUpdateTimeSinceEpoch != nil {
		updatedAt = *cm.LastUpdateTimeSinceEpoch
	}

	return plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "CatalogModel",
		Metadata: plugin.AssetMetadata{
			UID:         uid,
			Name:        cm.Name,
			Description: description,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}

func mapCatalogModelMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"provider", "license", "licenseLink", "maturity", "libraryName",
		"logo", "readme", "tasks", "language", "source_id",
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
		Kind:       "CatalogModel",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
			CreatedAt:   getString("createTimeSinceEpoch"),
			UpdatedAt:   getString("lastUpdateTimeSinceEpoch"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
