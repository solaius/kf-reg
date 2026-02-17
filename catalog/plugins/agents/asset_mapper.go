package agents

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*AgentPlugin)(nil)

// GetAssetMapper returns the asset mapper for the agents plugin.
func (p *AgentPlugin) GetAssetMapper() plugin.AssetMapper {
	return &agentAssetMapper{}
}

type agentAssetMapper struct{}

// SupportedKinds returns the entity kinds this mapper handles.
func (m *agentAssetMapper) SupportedKinds() []string {
	return []string{"Agent"}
}

// MapToAsset converts a single agent entity to an AssetResource.
func (m *agentAssetMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
	switch e := entity.(type) {
	case AgentEntry:
		return mapAgentToAsset(e), nil
	case *AgentEntry:
		if e == nil {
			return plugin.AssetResource{}, fmt.Errorf("nil AgentEntry pointer")
		}
		return mapAgentToAsset(*e), nil
	case map[string]any:
		return mapAgentMapToAsset(e), nil
	default:
		return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T for Agent mapper", entity)
	}
}

// MapToAssets converts a slice of entities into AssetResource items.
func (m *agentAssetMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
	return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}

func mapAgentToAsset(e AgentEntry) plugin.AssetResource {
	spec := make(map[string]any)

	if e.AgentType != nil {
		spec["agentType"] = *e.AgentType
	}
	if e.Instructions != nil {
		spec["instructions"] = *e.Instructions
	}
	if e.ModelConfig != nil {
		spec["modelConfig"] = e.ModelConfig
	}
	if e.Tools != nil {
		spec["tools"] = e.Tools
	}
	if e.Knowledge != nil {
		spec["knowledge"] = e.Knowledge
	}
	if e.Guardrails != nil {
		spec["guardrails"] = e.Guardrails
	}
	if e.Policies != nil {
		spec["policies"] = e.Policies
	}
	if e.PromptRefs != nil {
		spec["promptRefs"] = e.PromptRefs
	}
	if e.Dependencies != nil {
		spec["dependencies"] = e.Dependencies
	}
	if e.InputSchema != nil {
		spec["inputSchema"] = e.InputSchema
	}
	if e.OutputSchema != nil {
		spec["outputSchema"] = e.OutputSchema
	}
	if e.Examples != nil {
		spec["examples"] = e.Examples
	}
	if e.Category != nil {
		spec["category"] = *e.Category
	}

	desc := ""
	if e.Description != nil {
		desc = *e.Description
	}

	asset := plugin.AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "Agent",
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

	// Populate cross-asset links.
	asset.Status.Links = extractCrossLinks(e)

	return asset
}

// extractCrossLinks extracts cross-asset references from an agent entry.
func extractCrossLinks(e AgentEntry) *plugin.AssetLinks {
	var links []plugin.LinkRef

	for _, t := range e.Tools {
		if ref, ok := t["skillRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "Skill", Name: ref})
		}
		if ref, ok := t["mcpToolRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "McpServer", Name: ref})
		}
	}
	for _, k := range e.Knowledge {
		if ref, ok := k["knowledgeSourceRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "KnowledgeSource", Name: ref})
		}
	}
	for _, g := range e.Guardrails {
		if ref, ok := g["guardrailRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "Guardrail", Name: ref})
		}
	}
	for _, pol := range e.Policies {
		if ref, ok := pol["policyRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "Policy", Name: ref})
		}
	}
	for _, pr := range e.PromptRefs {
		if ref, ok := pr["promptTemplateRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "PromptTemplate", Name: ref})
		}
	}
	for _, d := range e.Dependencies {
		if ref, ok := d["agentRef"].(string); ok && ref != "" {
			links = append(links, plugin.LinkRef{Kind: "Agent", Name: ref})
		}
	}

	if len(links) == 0 {
		return nil
	}
	return &plugin.AssetLinks{Related: links}
}

func mapAgentMapToAsset(m map[string]any) plugin.AssetResource {
	spec := make(map[string]any)
	for _, key := range []string{
		"agentType", "instructions", "modelConfig", "tools", "knowledge",
		"guardrails", "policies", "promptRefs", "dependencies",
		"inputSchema", "outputSchema", "examples", "category",
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
		Kind:       "Agent",
		Metadata: plugin.AssetMetadata{
			UID:         getString("id"),
			Name:        getString("name"),
			Description: getString("description"),
		},
		Spec:   spec,
		Status: plugin.DefaultAssetStatus(),
	}
}
