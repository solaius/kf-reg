package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// actionHandler is the generic HTTP handler for :action endpoints.
// It parses the action request, finds the plugin's ActionProvider,
// and dispatches the action.
func actionHandler(p CatalogPlugin, scope ActionScope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ap, ok := p.(ActionProvider)
		if !ok {
			writeError(w, http.StatusNotImplemented,
				fmt.Sprintf("plugin %q does not support actions", p.Name()), nil)
			return
		}

		// Extract target ID from the URL. For source scope, it's
		// the sourceId param; for asset scope, it's the entityName param.
		var targetID string
		if scope == ActionScopeSource {
			targetID = chi.URLParam(r, "sourceId")
		} else {
			targetID = chi.URLParam(r, "entityName")
		}

		var req ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("invalid action request: %v", err), nil)
			return
		}

		if req.Action == "" {
			writeError(w, http.StatusBadRequest, "action field is required", nil)
			return
		}

		// Verify the action is declared.
		actions := ap.ListActions(scope)
		var found *ActionDefinition
		for i := range actions {
			if actions[i].ID == req.Action {
				found = &actions[i]
				break
			}
		}
		if found == nil {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("unknown action %q for scope %q", req.Action, scope), nil)
			return
		}

		// Check dry-run support.
		if req.DryRun && !found.SupportsDryRun {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("action %q does not support dry-run", req.Action), nil)
			return
		}

		result, err := ap.HandleAction(r.Context(), scope, targetID, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "action execution failed", err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// actionsListHandler returns a handler that lists available actions for a scope.
func actionsListHandler(p CatalogPlugin, scope ActionScope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ap, ok := p.(ActionProvider)
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"actions": []ActionDefinition{},
				"count":   0,
			})
			return
		}

		actions := ap.ListActions(scope)
		if actions == nil {
			actions = []ActionDefinition{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"actions": actions,
			"count":   len(actions),
		})
	}
}
