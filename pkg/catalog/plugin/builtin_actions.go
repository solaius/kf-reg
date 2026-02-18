package plugin

import (
	"context"
	"fmt"

	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// BuiltinActionHandler handles common asset actions (tag, annotate, deprecate)
// using the overlay store.
type BuiltinActionHandler struct {
	overlayStore *OverlayStore
	pluginName   string
	entityKind   string
}

// NewBuiltinActionHandler creates a handler for built-in actions.
func NewBuiltinActionHandler(overlayStore *OverlayStore, pluginName, entityKind string) *BuiltinActionHandler {
	return &BuiltinActionHandler{
		overlayStore: overlayStore,
		pluginName:   pluginName,
		entityKind:   entityKind,
	}
}

// BuiltinActionDefinitions returns the standard action definitions that
// the builtin handler supports. Plugins can include these in their
// ListActions response.
func BuiltinActionDefinitions() []ActionDefinition {
	return []ActionDefinition{
		{
			ID:             "tag",
			DisplayName:    "Tag",
			Description:    "Add or replace tags on an entity",
			Scope:          string(ActionScopeAsset),
			SupportsDryRun: true,
			Idempotent:     true,
		},
		{
			ID:             "annotate",
			DisplayName:    "Annotate",
			Description:    "Add or update annotations on an entity",
			Scope:          string(ActionScopeAsset),
			SupportsDryRun: true,
			Idempotent:     true,
		},
		{
			ID:             "deprecate",
			DisplayName:    "Deprecate",
			Description:    "Mark an entity as deprecated",
			Scope:          string(ActionScopeAsset),
			SupportsDryRun: true,
			Idempotent:     true,
		},
	}
}

// HandleTag adds or replaces tags on an entity.
func (h *BuiltinActionHandler) HandleTag(ctx context.Context, entityUID string, req ActionRequest) (*ActionResult, error) {
	tags, err := extractStringSlice(req.Params, "tags")
	if err != nil {
		return nil, fmt.Errorf("invalid tags parameter: %w", err)
	}

	if req.DryRun {
		return &ActionResult{
			Action:  "tag",
			Status:  "dry-run",
			Message: fmt.Sprintf("would set %d tags on %s", len(tags), entityUID),
			Data:    map[string]any{"tags": tags},
		}, nil
	}

	ns := tenancy.NamespaceFromContext(ctx)
	record, err := h.getOrCreateOverlay(ns, entityUID)
	if err != nil {
		return nil, err
	}

	record.Tags = tags
	if err := h.overlayStore.Upsert(record); err != nil {
		return nil, fmt.Errorf("failed to save overlay: %w", err)
	}

	return &ActionResult{
		Action:  "tag",
		Status:  "completed",
		Message: fmt.Sprintf("set %d tags on %s", len(tags), entityUID),
		Data:    map[string]any{"tags": tags},
	}, nil
}

// HandleAnnotate adds or updates annotations on an entity.
func (h *BuiltinActionHandler) HandleAnnotate(ctx context.Context, entityUID string, req ActionRequest) (*ActionResult, error) {
	annotations, err := extractStringMap(req.Params, "annotations")
	if err != nil {
		return nil, fmt.Errorf("invalid annotations parameter: %w", err)
	}

	if req.DryRun {
		return &ActionResult{
			Action:  "annotate",
			Status:  "dry-run",
			Message: fmt.Sprintf("would set %d annotations on %s", len(annotations), entityUID),
			Data:    map[string]any{"annotations": annotations},
		}, nil
	}

	ns := tenancy.NamespaceFromContext(ctx)
	record, err := h.getOrCreateOverlay(ns, entityUID)
	if err != nil {
		return nil, err
	}

	// Merge annotations into existing.
	if record.Annotations == nil {
		record.Annotations = make(JSONMap)
	}
	for k, v := range annotations {
		record.Annotations[k] = v
	}

	if err := h.overlayStore.Upsert(record); err != nil {
		return nil, fmt.Errorf("failed to save overlay: %w", err)
	}

	return &ActionResult{
		Action:  "annotate",
		Status:  "completed",
		Message: fmt.Sprintf("set %d annotations on %s", len(annotations), entityUID),
		Data:    map[string]any{"annotations": map[string]string(record.Annotations)},
	}, nil
}

// HandleDeprecate changes the lifecycle phase to deprecated.
func (h *BuiltinActionHandler) HandleDeprecate(ctx context.Context, entityUID string, req ActionRequest) (*ActionResult, error) {
	phase := "deprecated"
	if p, ok := req.Params["phase"]; ok {
		if s, ok := p.(string); ok && s != "" {
			phase = s
		}
	}

	if req.DryRun {
		return &ActionResult{
			Action:  "deprecate",
			Status:  "dry-run",
			Message: fmt.Sprintf("would set lifecycle of %s to %q", entityUID, phase),
			Data:    map[string]any{"lifecycle": phase},
		}, nil
	}

	ns := tenancy.NamespaceFromContext(ctx)
	record, err := h.getOrCreateOverlay(ns, entityUID)
	if err != nil {
		return nil, err
	}

	record.Lifecycle = phase
	if err := h.overlayStore.Upsert(record); err != nil {
		return nil, fmt.Errorf("failed to save overlay: %w", err)
	}

	return &ActionResult{
		Action:  "deprecate",
		Status:  "completed",
		Message: fmt.Sprintf("set lifecycle of %s to %q", entityUID, phase),
		Data:    map[string]any{"lifecycle": phase},
	}, nil
}

// getOrCreateOverlay retrieves an existing overlay or creates a new empty one.
func (h *BuiltinActionHandler) getOrCreateOverlay(namespace, entityUID string) (*OverlayRecord, error) {
	record, err := h.overlayStore.Get(namespace, h.pluginName, h.entityKind, entityUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overlay: %w", err)
	}
	if record == nil {
		record = &OverlayRecord{
			Namespace:  namespace,
			PluginName: h.pluginName,
			EntityKind: h.entityKind,
			EntityUID:  entityUID,
		}
	}
	return record, nil
}

// extractStringSlice extracts a []string from params by key.
func extractStringSlice(params map[string]any, key string) ([]string, error) {
	val, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("missing %q parameter", key)
	}

	switch v := val.(type) {
	case []string:
		return v, nil
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%q contains non-string element: %T", key, item)
			}
			result = append(result, s)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%q must be an array of strings, got %T", key, val)
	}
}

// extractStringMap extracts a map[string]string from params by key.
func extractStringMap(params map[string]any, key string) (map[string]string, error) {
	val, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("missing %q parameter", key)
	}

	switch v := val.(type) {
	case map[string]string:
		return v, nil
	case map[string]any:
		result := make(map[string]string, len(v))
		for k, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%q[%q] is not a string: %T", key, k, item)
			}
			result[k] = s
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%q must be an object with string values, got %T", key, val)
	}
}
