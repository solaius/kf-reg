package plugin

import (
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/governance"
)

// BuildCapabilitiesV2 assembles PluginCapabilitiesV2 from a plugin by checking
// which optional interfaces it implements. If the plugin implements
// CapabilitiesV2Provider directly, that result is returned as-is.
func BuildCapabilitiesV2(p CatalogPlugin, basePath string) PluginCapabilitiesV2 {
	// If the plugin provides V2 directly, start with that.
	if v2p, ok := p.(CapabilitiesV2Provider); ok {
		caps := v2p.GetCapabilitiesV2()
		applyGovernanceCaps(&caps)
		return caps
	}

	caps := PluginCapabilitiesV2{
		SchemaVersion: "v1",
		Plugin: PluginMeta{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
		},
		Entities: []EntityCapabilities{},
		Actions:  []ActionDefinition{},
	}

	// Build from V1 capabilities and other interfaces.
	if cp, ok := p.(CapabilitiesProvider); ok {
		v1 := cp.Capabilities()
		for _, kind := range v1.EntityKinds {
			entity := EntityCapabilities{
				Kind:   kind,
				Plural: pluralize(kind),
			}
			if v1.ListEntities {
				entity.Endpoints.List = fmt.Sprintf("%s/%s", basePath, entity.Plural)
			}
			if v1.GetEntity {
				entity.Endpoints.Get = fmt.Sprintf("%s/%s/{name}", basePath, entity.Plural)
			}
			caps.Entities = append(caps.Entities, entity)
		}
	}

	// Add source capabilities.
	_, hasSM := p.(SourceManager)
	_, hasRP := p.(RefreshProvider)
	if hasSM || hasRP {
		caps.Sources = &SourceCapabilities{
			Manageable:  hasSM,
			Refreshable: hasRP,
		}
	}

	applyGovernanceCaps(&caps)

	return caps
}

// applyGovernanceCaps adds governance capabilities to all entities in a PluginCapabilitiesV2.
func applyGovernanceCaps(caps *PluginCapabilitiesV2) {
	govCaps := &governance.GovernanceCapabilities{
		Supported: true,
		Lifecycle: &governance.LifecycleCapabilities{
			States:       []string{"draft", "approved", "deprecated", "archived"},
			DefaultState: "draft",
		},
		Versioning: &governance.VersionCapabilities{
			Enabled:      true,
			Environments: []string{"dev", "stage", "prod"},
		},
		Approvals: &governance.ApprovalCapabilities{
			Enabled: true,
		},
		Provenance: &governance.ProvenanceCapabilities{
			Enabled: true,
		},
	}
	for i := range caps.Entities {
		caps.Entities[i].Governance = govCaps
	}
}

// pluralize returns a simple lowercase plural form of a kind name.
// It lowercases the kind and appends "s".
func pluralize(kind string) string {
	if kind == "" {
		return ""
	}
	// Lowercase the whole kind for URL paths.
	lower := ""
	for i, r := range kind {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += kind[i : i+1]
		}
	}
	return lower + "s"
}
