package plugin

import "context"

// ActionScope indicates whether an action targets a source or an asset.
type ActionScope string

const (
	// ActionScopeSource targets a data source.
	ActionScopeSource ActionScope = "source"

	// ActionScopeAsset targets an individual asset/entity.
	ActionScopeAsset ActionScope = "asset"
)

// ActionRequest is the body for :action endpoints.
type ActionRequest struct {
	Action string         `json:"action"`
	DryRun bool           `json:"dryRun,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

// ActionResult is the response from :action endpoints.
type ActionResult struct {
	Action  string         `json:"action"`
	Status  string         `json:"status"` // "completed", "dry-run", "error"
	Message string         `json:"message,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

// ActionProvider is an optional interface that plugins can implement
// to handle actions on their entities and sources.
type ActionProvider interface {
	// HandleAction executes an action. scope is "source" or "asset", targetID
	// is the source ID or entity name.
	HandleAction(ctx context.Context, scope ActionScope, targetID string, req ActionRequest) (*ActionResult, error)

	// ListActions returns the actions available for the given scope.
	ListActions(scope ActionScope) []ActionDefinition
}
