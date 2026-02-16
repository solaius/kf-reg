package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// managementRouter registers management routes for a plugin on the given router.
// Management routes are sub-paths under the plugin's base path.
func managementRouter(p CatalogPlugin, roleExtractor RoleExtractor) chi.Router {
	r := chi.NewRouter()

	// Sources management (requires SourceManager)
	if sm, ok := p.(SourceManager); ok {
		r.Get("/sources", sourcesListHandler(sm))
		r.Post("/validate-source", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(validateHandler(sm))).ServeHTTP)
		r.Post("/apply-source", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(applyHandler(sm))).ServeHTTP)
		r.Post("/sources/{sourceId}/enable", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(enableHandler(sm))).ServeHTTP)
		r.Delete("/sources/{sourceId}", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(deleteSourceHandler(sm))).ServeHTTP)
	}

	// Refresh (requires RefreshProvider)
	if rp, ok := p.(RefreshProvider); ok {
		r.Post("/refresh", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(refreshAllHandler(rp))).ServeHTTP)
		r.Post("/refresh/{sourceId}", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(refreshSourceHandler(rp))).ServeHTTP)
	}

	// Diagnostics (read-only, available to viewers)
	if dp, ok := p.(DiagnosticsProvider); ok {
		r.Get("/diagnostics", diagnosticsHandler(dp))
	}

	return r
}

// sourcesListHandler returns a handler that lists all sources for a plugin.
func sourcesListHandler(sm SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sources, err := sm.ListSources(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list sources", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sources": sources,
			"count":   len(sources),
		})
	}
}

// validateHandler returns a handler that validates a source configuration.
func validateHandler(sm SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input SourceConfigInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err)
			return
		}

		result, err := sm.ValidateSource(r.Context(), input)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "validation failed", err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// applyHandler returns a handler that applies a source configuration.
func applyHandler(sm SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input SourceConfigInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err)
			return
		}

		if err := sm.ApplySource(r.Context(), input); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to apply source", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "applied"})
	}
}

// enableHandler returns a handler that enables or disables a source.
func enableHandler(sm SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceID := chi.URLParam(r, "sourceId")
		if sourceID == "" {
			writeError(w, http.StatusBadRequest, "missing source ID", nil)
			return
		}

		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err)
			return
		}

		if err := sm.EnableSource(r.Context(), sourceID, body.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to enable/disable source", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "updated",
			"enabled": body.Enabled,
		})
	}
}

// deleteSourceHandler returns a handler that removes a source.
func deleteSourceHandler(sm SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceID := chi.URLParam(r, "sourceId")
		if sourceID == "" {
			writeError(w, http.StatusBadRequest, "missing source ID", nil)
			return
		}

		if err := sm.DeleteSource(r.Context(), sourceID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete source", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// refreshAllHandler returns a handler that triggers a refresh of all sources.
func refreshAllHandler(rp RefreshProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := rp.RefreshAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "refresh failed", err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// refreshSourceHandler returns a handler that triggers a refresh of a single source.
func refreshSourceHandler(rp RefreshProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceID := chi.URLParam(r, "sourceId")
		if sourceID == "" {
			writeError(w, http.StatusBadRequest, "missing source ID", nil)
			return
		}

		result, err := rp.Refresh(r.Context(), sourceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "refresh failed", err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// diagnosticsHandler returns a handler that returns diagnostic information.
func diagnosticsHandler(dp DiagnosticsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diag, err := dp.Diagnostics(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get diagnostics", err)
			return
		}
		writeJSON(w, http.StatusOK, diag)
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   http.StatusText(status),
		"message": errMsg,
	})
}
