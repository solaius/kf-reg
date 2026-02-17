package mcp

import (
	"fmt"

	"github.com/kubeflow/model-registry/catalog/plugins/mcp/internal/server/openapi"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*McpServerCatalogPlugin)(nil)

// GetAssetMapper returns the asset mapper for the MCP plugin.
func (p *McpServerCatalogPlugin) GetAssetMapper() plugin.AssetMapper {
	return &mcpAssetMapper{}
}

type mcpAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *mcpAssetMapper) SupportedKinds() []string {
	return []string{"McpServer"}
}

// MapToAsset converts a single MCP server entity to an AssetResource.
func (m *mcpAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case openapi.McpServer:
		return mapMcpServerToAsset(e), nil
	case *openapi.McpServer:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil McpServer pointer")
		}
		return mapMcpServerToAsset(*e), nil
	case map[string]any:
		return mapMcpServerMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for McpServer mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *mcpAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapMcpServerToAsset(s openapi.McpServer) plugin.AssetResource {
	spec := map[string]any{
		"serverUrl":      s.ServerUrl,
		"transportType":  s.TransportType,
		"deploymentMode": s.DeploymentMode,
		"provider":       s.Provider,
		"license":        s.License,
		"category":       s.Category,
		"image":          s.Image,
		"endpoint":       s.Endpoint,
	}
	if s.ToolCount != nil {
		spec["toolCount"] = *s.ToolCount
	}
	if s.ResourceCount != nil {
		spec["resourceCount"] = *s.ResourceCount
	}
	if s.PromptCount != nil {
		spec["promptCount"] = *s.PromptCount
	}
	if s.SupportedTransports != "" {
		spec["supportedTransports"] = s.SupportedTransports
	}
	if s.Verified != nil {
		spec["verified"] = *s.Verified
	}
	if s.Certified != nil {
		spec["certified"] = *s.Certified
	}
	if s.Logo != "" {
		spec["logo"] = s.Logo
	}

	return plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "McpServer",
		Metadata: plugin.AssetMetadata{
			UID:         s.Id,
			Name:        s.Name,
			Description: s.Description,
			CreatedAt:   s.CreateTimeSinceEpoch,
			UpdatedAt:   s.LastUpdateTimeSinceEpoch,
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}

func mapMcpServerMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"serverUrl", "transportType", "deploymentMode", "provider",
		"license", "category", "image", "endpoint", "toolCount",
		"resourceCount", "promptCount", "supportedTransports",
		"verified", "certified", "logo",
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
		Kind:       "McpServer",
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
