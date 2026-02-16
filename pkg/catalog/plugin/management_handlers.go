package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// managementRouter registers management routes for a plugin on the given router.
// Management routes are sub-paths under the plugin's base path.
// The server parameter is used for config persistence; it may be nil if no
// ConfigStore is configured (mutations remain in-memory only).
func managementRouter(p CatalogPlugin, roleExtractor RoleExtractor, srv *Server) chi.Router {
	r := chi.NewRouter()

	configKey := pluginConfigKey(p)

	// Sources management (requires SourceManager)
	if sm, ok := p.(SourceManager); ok {
		r.Get("/sources", sourcesListHandler(sm))
		r.Post("/validate-source", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(validateHandler(sm))).ServeHTTP)
		r.Post("/apply-source", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(applyHandler(sm, srv, configKey, p))).ServeHTTP)
		r.Post("/sources/{sourceId}/enable", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(enableHandler(sm, srv, configKey))).ServeHTTP)
		r.Delete("/sources/{sourceId}", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(deleteSourceHandler(sm, srv, configKey))).ServeHTTP)

		// Detailed validation with multi-layer breakdown.
		r.Post("/sources/{sourceId}:validate", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(detailedValidateHandler(sm))).ServeHTTP)

		// Revision history and rollback (requires ConfigStore).
		r.Get("/sources/{sourceId}/revisions", revisionsHandler(srv))
		r.Post("/sources/{sourceId}:rollback", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(rollbackHandler(srv, configKey, p))).ServeHTTP)
	}

	// Refresh (requires RefreshProvider)
	if rp, ok := p.(RefreshProvider); ok {
		var rl *RefreshRateLimiter
		if srv != nil {
			rl = srv.rateLimiter
		}
		pluginName := p.Name()
		r.Post("/refresh", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(refreshAllHandler(rp, rl, pluginName))).ServeHTTP)
		r.Post("/refresh/{sourceId}", RequireRole(RoleOperator, roleExtractor)(http.HandlerFunc(refreshSourceHandler(rp, rl, pluginName))).ServeHTTP)
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
// It runs multi-layer validation before applying; invalid configs are rejected with 422.
// After the in-memory mutation succeeds, it persists the change to the ConfigStore.
func applyHandler(sm SourceManager, srv *Server, configKey string, p CatalogPlugin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input SourceConfigInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err)
			return
		}

		// Run validation before apply.
		validator := NewDefaultValidator(sm)
		valResult := validator.Validate(r.Context(), input)
		if !valResult.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(valResult)
			return
		}

		if err := sm.ApplySource(r.Context(), input); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to apply source", err)
			return
		}

		// Persist to ConfigStore if available.
		if srv != nil && srv.GetConfigStore() != nil {
			src := sourceConfigFromInput(input)
			srv.updateConfigSource(configKey, src)
			if _, err := srv.persistConfig(r.Context()); err != nil {
				if errors.Is(err, ErrVersionConflict) {
					writeError(w, http.StatusConflict, "config was modified externally, retry the operation", err)
					return
				}
				srv.logger.Error("failed to persist config after apply", "error", err)
				// Non-fatal: the in-memory state was already updated.
			}
		}

		result := ApplyResult{Status: "applied"}

		// Optionally trigger refresh after apply.
		if input.RefreshAfterApply != nil && *input.RefreshAfterApply {
			if rp, ok := p.(RefreshProvider); ok {
				start := time.Now()
				refreshResult, err := rp.Refresh(r.Context(), input.ID)
				elapsed := time.Since(start)
				if err != nil {
					result.RefreshResult = &RefreshResult{
						SourceID: input.ID,
						Duration: elapsed,
						Error:    err.Error(),
					}
				} else {
					if refreshResult != nil {
						refreshResult.Duration = elapsed
						result.RefreshResult = refreshResult
					} else {
						result.RefreshResult = &RefreshResult{
							SourceID: input.ID,
							Duration: elapsed,
						}
					}
				}
			}
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// enableHandler returns a handler that enables or disables a source.
// After the in-memory mutation succeeds, it persists the change to the ConfigStore.
func enableHandler(sm SourceManager, srv *Server, configKey string) http.HandlerFunc {
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

		// Persist to ConfigStore if available.
		if srv != nil && srv.GetConfigStore() != nil {
			srv.enableConfigSource(configKey, sourceID, body.Enabled)
			if _, err := srv.persistConfig(r.Context()); err != nil {
				if errors.Is(err, ErrVersionConflict) {
					writeError(w, http.StatusConflict, "config was modified externally, retry the operation", err)
					return
				}
				srv.logger.Error("failed to persist config after enable", "error", err)
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "updated",
			"enabled": body.Enabled,
		})
	}
}

// deleteSourceHandler returns a handler that removes a source.
// After the in-memory mutation succeeds, it persists the change to the ConfigStore.
func deleteSourceHandler(sm SourceManager, srv *Server, configKey string) http.HandlerFunc {
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

		// Persist to ConfigStore if available.
		if srv != nil && srv.GetConfigStore() != nil {
			srv.deleteConfigSource(configKey, sourceID)
			if _, err := srv.persistConfig(r.Context()); err != nil {
				if errors.Is(err, ErrVersionConflict) {
					writeError(w, http.StatusConflict, "config was modified externally, retry the operation", err)
					return
				}
				srv.logger.Error("failed to persist config after delete", "error", err)
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// refreshAllHandler returns a handler that triggers a refresh of all sources.
// It checks the rate limiter before proceeding. If rate limited, it returns
// 429 with a Retry-After header.
func refreshAllHandler(rp RefreshProvider, rl *RefreshRateLimiter, pluginName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rl != nil {
			allowed, retryAfter := rl.Allow(RefreshAllKey(pluginName))
			if !allowed {
				writeRateLimited(w, retryAfter)
				return
			}
		}

		result, err := rp.RefreshAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "refresh failed", err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// refreshSourceHandler returns a handler that triggers a refresh of a single source.
// It checks the rate limiter before proceeding. If rate limited, it returns
// 429 with a Retry-After header.
func refreshSourceHandler(rp RefreshProvider, rl *RefreshRateLimiter, pluginName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceID := chi.URLParam(r, "sourceId")
		if sourceID == "" {
			writeError(w, http.StatusBadRequest, "missing source ID", nil)
			return
		}

		if rl != nil {
			allowed, retryAfter := rl.Allow(RefreshKey(pluginName, sourceID))
			if !allowed {
				writeRateLimited(w, retryAfter)
				return
			}
		}

		result, err := rp.Refresh(r.Context(), sourceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "refresh failed", err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// detailedValidateHandler returns a handler that runs the multi-layer validator
// and returns a DetailedValidationResult with per-layer breakdown.
func detailedValidateHandler(sm SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input SourceConfigInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err)
			return
		}

		validator := NewDefaultValidator(sm)
		result := validator.Validate(r.Context(), input)

		writeJSON(w, http.StatusOK, result)
	}
}

// revisionsHandler returns a handler that lists revision history from the ConfigStore.
func revisionsHandler(srv *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if srv == nil || srv.GetConfigStore() == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"revisions": []ConfigRevision{},
				"count":     0,
			})
			return
		}

		revisions, err := srv.GetConfigStore().ListRevisions(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list revisions", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"revisions": revisions,
			"count":     len(revisions),
		})
	}
}

// rollbackHandler returns a handler that restores a previous config revision.
// After rollback, it re-initializes the affected plugin.
func rollbackHandler(srv *Server, configKey string, p CatalogPlugin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if srv == nil || srv.GetConfigStore() == nil {
			writeError(w, http.StatusBadRequest, "no config store configured, rollback not available", nil)
			return
		}

		var body struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err)
			return
		}
		if body.Version == "" {
			writeError(w, http.StatusBadRequest, "version is required", nil)
			return
		}

		cfg, newVersion, err := srv.GetConfigStore().Rollback(r.Context(), body.Version)
		if err != nil {
			if errors.Is(err, ErrRevisionNotFound) {
				writeError(w, http.StatusNotFound, "revision not found", err)
				return
			}
			if errors.Is(err, ErrVersionConflict) {
				writeError(w, http.StatusConflict, "config was modified externally, retry the operation", err)
				return
			}
			writeError(w, http.StatusInternalServerError, "rollback failed", err)
			return
		}

		// Update in-memory config and version.
		srv.mu.Lock()
		srv.config = cfg
		srv.configVersion = newVersion
		srv.mu.Unlock()

		// Re-initialize the plugin with the restored config.
		section, ok := cfg.Catalogs[configKey]
		if !ok {
			section = CatalogSection{}
		}

		var basePath string
		if bp, ok := p.(BasePathProvider); ok {
			basePath = bp.BasePath()
		} else {
			basePath = fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
		}

		pluginCfg := Config{
			Section:  section,
			DB:       srv.db,
			Logger:   srv.logger.With("plugin", p.Name()),
			BasePath: basePath,
		}

		if err := p.Init(r.Context(), pluginCfg); err != nil {
			srv.logger.Error("rollback: plugin re-init failed", "plugin", p.Name(), "error", err)
			writeJSON(w, http.StatusOK, map[string]any{
				"status":     "rolled_back",
				"version":    newVersion,
				"reinitError": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "rolled_back",
			"version": newVersion,
		})
	}
}

// writeRateLimited writes a 429 Too Many Requests response with a Retry-After header.
func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, http.StatusTooManyRequests, fmt.Sprintf("rate limited, retry after %d seconds", seconds), nil)
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

// sourceConfigFromInput converts a SourceConfigInput (API type) to a
// SourceConfig (file/config type) for persistence.
func sourceConfigFromInput(input SourceConfigInput) SourceConfig {
	return SourceConfig{
		ID:         input.ID,
		Name:       input.Name,
		Type:       input.Type,
		Enabled:    input.Enabled,
		Labels:     input.Labels,
		Properties: input.Properties,
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
