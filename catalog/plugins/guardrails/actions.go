package guardrails

import (
	"context"
	"fmt"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.ActionProvider = (*GuardrailPlugin)(nil)

// HandleAction executes an action on a source or asset.
func (p *GuardrailPlugin) HandleAction(ctx context.Context, scope plugin.ActionScope, targetID string, req plugin.ActionRequest) (*plugin.ActionResult, error) {
	switch scope {
	case plugin.ActionScopeSource:
		return p.handleSourceAction(ctx, targetID, req)
	case plugin.ActionScopeAsset:
		return p.handleAssetAction(ctx, targetID, req)
	default:
		return nil, fmt.Errorf("unknown action scope %q", scope)
	}
}

// ListActions returns the actions available for the given scope.
func (p *GuardrailPlugin) ListActions(scope plugin.ActionScope) []plugin.ActionDefinition {
	switch scope {
	case plugin.ActionScopeSource:
		return []plugin.ActionDefinition{
			{
				ID:             "refresh",
				DisplayName:    "Refresh",
				Description:    "Refresh entities from source",
				Scope:          string(plugin.ActionScopeSource),
				SupportsDryRun: false,
				Idempotent:     true,
			},
		}
	case plugin.ActionScopeAsset:
		return plugin.BuiltinActionDefinitions()
	default:
		return nil
	}
}

func (p *GuardrailPlugin) handleSourceAction(ctx context.Context, sourceID string, req plugin.ActionRequest) (*plugin.ActionResult, error) {
	switch req.Action {
	case "refresh":
		result, err := p.Refresh(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("Refreshed source %s", sourceID)
		if result.Error != "" {
			msg = fmt.Sprintf("Refresh of source %s failed: %s", sourceID, result.Error)
		}
		return &plugin.ActionResult{
			Action:  "refresh",
			Status:  "completed",
			Message: msg,
			Data: map[string]any{
				"entitiesLoaded": result.EntitiesLoaded,
				"duration":       result.Duration.String(),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown source action %q", req.Action)
	}
}

func (p *GuardrailPlugin) handleAssetAction(ctx context.Context, entityName string, req plugin.ActionRequest) (*plugin.ActionResult, error) {
	handler := p.builtinActionHandler()
	if handler == nil {
		return nil, fmt.Errorf("overlay store not available")
	}

	switch req.Action {
	case "tag":
		return handler.HandleTag(ctx, entityName, req)
	case "annotate":
		return handler.HandleAnnotate(ctx, entityName, req)
	case "deprecate":
		return handler.HandleDeprecate(ctx, entityName, req)
	default:
		return nil, fmt.Errorf("unknown asset action %q", req.Action)
	}
}

// builtinActionHandler returns a BuiltinActionHandler for this plugin.
func (p *GuardrailPlugin) builtinActionHandler() *plugin.BuiltinActionHandler {
	if p.cfg.DB == nil {
		return nil
	}
	store := plugin.NewOverlayStore(p.cfg.DB)
	return plugin.NewBuiltinActionHandler(store, PluginName, "Guardrail")
}
