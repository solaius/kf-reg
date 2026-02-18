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
	"github.com/google/uuid"

	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/jobs"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// managementRouter registers management routes for a plugin on the given router.
// Management routes are sub-paths under the plugin's base path.
// The server parameter is used for config persistence; it may be nil if no
// ConfigStore is configured (mutations remain in-memory only).
// When authorizer is non-nil, fine-grained SAR-based permission checks are used
// instead of the simple RoleExtractor-based model.
func managementRouter(p CatalogPlugin, roleExtractor RoleExtractor, srv *Server, authorizer authz.Authorizer) chi.Router {
	r := chi.NewRouter()

	configKey := pluginConfigKey(p)

	pluginName := p.Name()

	// guard wraps a handler with the appropriate authorization middleware.
	// When an authz.Authorizer is configured, it uses fine-grained resource/verb
	// checks via RequirePermission. Otherwise, it falls back to the legacy
	// RoleExtractor-based RequireRole(RoleOperator) middleware.
	guard := func(resource, verb string, h http.HandlerFunc) http.HandlerFunc {
		if authorizer != nil {
			return authz.RequirePermission(authorizer, resource, verb)(h).ServeHTTP
		}
		return RequireRole(RoleOperator, roleExtractor)(h).ServeHTTP
	}

	// Sources management (requires SourceManager)
	if sm, ok := p.(SourceManager); ok {
		r.Get("/sources", sourcesListHandler(sm, srv, pluginName))
		r.Post("/validate-source", guard(authz.ResourceCatalogSources, authz.VerbUpdate, validateHandler(sm)))
		r.Post("/apply-source", guard(authz.ResourceCatalogSources, authz.VerbCreate, applyHandler(sm, srv, configKey, p)))
		r.Post("/sources/{sourceId}/enable", guard(authz.ResourceCatalogSources, authz.VerbUpdate, enableHandler(sm, srv, configKey)))
		r.Delete("/sources/{sourceId}", guard(authz.ResourceCatalogSources, authz.VerbDelete, deleteSourceHandler(sm, srv, configKey, pluginName)))

		// Detailed validation with multi-layer breakdown.
		r.Post("/sources/{sourceId}:validate", guard(authz.ResourceCatalogSources, authz.VerbUpdate, detailedValidateHandler(sm)))

		// Revision history and rollback (requires ConfigStore).
		r.Get("/sources/{sourceId}/revisions", revisionsHandler(srv))
		r.Post("/sources/{sourceId}:rollback", guard(authz.ResourceCatalogSources, authz.VerbUpdate, rollbackHandler(srv, configKey, p)))
	}

	// Refresh (requires RefreshProvider)
	if rp, ok := p.(RefreshProvider); ok {
		var rl *RefreshRateLimiter
		if srv != nil {
			rl = srv.rateLimiter
		}
		r.Post("/refresh", guard(authz.ResourceJobs, authz.VerbCreate, refreshAllHandler(rp, rl, pluginName, srv)))
		r.Post("/refresh/{sourceId}", guard(authz.ResourceJobs, authz.VerbCreate, refreshSourceHandler(rp, rl, pluginName, srv)))
	}

	// Diagnostics (read-only, available to viewers)
	if dp, ok := p.(DiagnosticsProvider); ok {
		r.Get("/diagnostics", diagnosticsHandler(dp))
	}

	// Action endpoints (requires ActionProvider)
	if _, ok := p.(ActionProvider); ok {
		// Source-scoped actions.
		r.Post("/sources/{sourceId}:action", guard(authz.ResourceActions, authz.VerbExecute, actionHandler(p, ActionScopeSource)))

		// Asset-scoped actions.
		r.Post("/entities/{entityName}:action", guard(authz.ResourceActions, authz.VerbExecute, actionHandler(p, ActionScopeAsset)))

		// Action discovery endpoints (read-only, available to all roles).
		r.Get("/actions/source", actionsListHandler(p, ActionScopeSource))
		r.Get("/actions/asset", actionsListHandler(p, ActionScopeAsset))
	}

	// Entity getter (requires EntityGetter) — provides GET /entities/{entityName}
	// for plugins whose native get endpoint has multiple path parameters.
	if eg, ok := p.(EntityGetter); ok {
		r.Get("/entities/{entityName}", entityGetterHandler(eg))
	}

	return r
}

// sourcesListHandler returns a handler that lists all sources for a plugin.
// Sensitive property values are redacted before returning.
// If a Server with a DB is available, persisted refresh status is merged into
// the returned SourceInfo objects.
func sourcesListHandler(sm SourceManager, srv *Server, pluginName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sources, err := sm.ListSources(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list sources", err)
			return
		}

		// Build a lookup of persisted refresh statuses.
		ns := tenancy.NamespaceFromContext(r.Context())
		var statusMap map[string]*RefreshStatusRecord
		if srv != nil {
			records := srv.listRefreshStatuses(ns, pluginName)
			if len(records) > 0 {
				statusMap = make(map[string]*RefreshStatusRecord, len(records))
				for i := range records {
					statusMap[records[i].SourceID] = &records[i]
				}
			}
		}

		// Redact sensitive values and enrich with persisted refresh status.
		for i := range sources {
			sources[i].Properties = RedactSensitiveProperties(sources[i].Properties)

			if rec, ok := statusMap[sources[i].ID]; ok {
				if sources[i].Status.LastRefreshTime == nil {
					sources[i].Status.LastRefreshTime = rec.LastRefreshTime
				}
				if sources[i].Status.LastRefreshStatus == "" {
					sources[i].Status.LastRefreshStatus = rec.LastRefreshStatus
				}
				if sources[i].Status.LastRefreshSummary == "" {
					sources[i].Status.LastRefreshSummary = rec.LastRefreshSummary
				}
				if sources[i].Status.Error == "" && rec.LastError != "" {
					sources[i].Status.Error = rec.LastError
				}
			}
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

		// Resolve SecretRef values before passing to the plugin.
		resolvedInput := input
		if srv != nil && srv.GetSecretResolver() != nil && input.Properties != nil {
			resolved, err := ResolveSecretRefs(r.Context(), input.Properties, srv.GetSecretResolver())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to resolve secret references", err)
				return
			}
			resolvedInput.Properties = resolved
		}

		if err := sm.ApplySource(r.Context(), resolvedInput); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to apply source", err)
			return
		}

		// Persist the original input (with SecretRefs intact) to ConfigStore.
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

		// Invalidate discovery/capabilities cache after successful apply.
		if srv != nil && srv.GetCacheManager() != nil {
			srv.GetCacheManager().InvalidatePlugin(p.Name())
		}

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
				// Persist refresh status after apply.
				if srv != nil && result.RefreshResult != nil {
					ns := tenancy.NamespaceFromContext(r.Context())
					srv.saveRefreshStatus(ns, p.Name(), input.ID, result.RefreshResult)
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
// After the in-memory mutation succeeds, it persists the change to the ConfigStore
// and cleans up the corresponding refresh status record.
func deleteSourceHandler(sm SourceManager, srv *Server, configKey string, pluginName string) http.HandlerFunc {
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

		// Clean up refresh status record for the deleted source.
		if srv != nil {
			ns := tenancy.NamespaceFromContext(r.Context())
			srv.deleteRefreshStatus(ns, pluginName, sourceID)
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
// When a job store is available, the refresh is enqueued as an async job and
// 202 Accepted is returned with the job ID. Otherwise, the refresh is executed
// synchronously. Rate limiting is checked before either path.
func refreshAllHandler(rp RefreshProvider, rl *RefreshRateLimiter, pluginName string, srv *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rl != nil {
			allowed, retryAfter := rl.Allow(RefreshAllKey(pluginName))
			if !allowed {
				writeRateLimited(w, retryAfter)
				return
			}
		}

		ns := tenancy.NamespaceFromContext(r.Context())
		actor := "system"
		if id, ok := authz.IdentityFromContext(r.Context()); ok {
			actor = id.User
		}

		// Async path: enqueue a job if the job store is available.
		if srv != nil && srv.jobStore != nil {
			idempKey := fmt.Sprintf("%s:%s:_all", ns, pluginName)
			job := &jobs.RefreshJob{
				ID:             uuid.New().String(),
				Namespace:      ns,
				Plugin:         pluginName,
				SourceID:       "_all",
				RequestedBy:    actor,
				RequestedAt:    time.Now(),
				State:          jobs.JobStateQueued,
				IdempotencyKey: idempKey,
			}

			enqueued, err := srv.jobStore.Enqueue(job)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to enqueue refresh job", err)
				return
			}

			writeJSON(w, http.StatusAccepted, map[string]any{
				"status": "queued",
				"jobId":  enqueued.ID,
			})
			return
		}

		// Synchronous fallback.
		result, err := rp.RefreshAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "refresh failed", err)
			return
		}

		if srv != nil && result != nil {
			srv.saveRefreshStatus(ns, pluginName, "_all", result)
		}

		// Invalidate cache after synchronous refresh completes.
		if srv != nil && srv.GetCacheManager() != nil {
			srv.GetCacheManager().InvalidatePlugin(pluginName)
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// refreshSourceHandler returns a handler that triggers a refresh of a single source.
// When a job store is available, the refresh is enqueued as an async job and
// 202 Accepted is returned with the job ID. Otherwise, the refresh is executed
// synchronously. Rate limiting is checked before either path.
func refreshSourceHandler(rp RefreshProvider, rl *RefreshRateLimiter, pluginName string, srv *Server) http.HandlerFunc {
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

		ns := tenancy.NamespaceFromContext(r.Context())
		actor := "system"
		if id, ok := authz.IdentityFromContext(r.Context()); ok {
			actor = id.User
		}

		// Async path: enqueue a job if the job store is available.
		if srv != nil && srv.jobStore != nil {
			idempKey := fmt.Sprintf("%s:%s:%s", ns, pluginName, sourceID)
			job := &jobs.RefreshJob{
				ID:             uuid.New().String(),
				Namespace:      ns,
				Plugin:         pluginName,
				SourceID:       sourceID,
				RequestedBy:    actor,
				RequestedAt:    time.Now(),
				State:          jobs.JobStateQueued,
				IdempotencyKey: idempKey,
			}

			enqueued, err := srv.jobStore.Enqueue(job)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to enqueue refresh job", err)
				return
			}

			writeJSON(w, http.StatusAccepted, map[string]any{
				"status": "queued",
				"jobId":  enqueued.ID,
			})
			return
		}

		// Synchronous fallback.
		result, err := rp.Refresh(r.Context(), sourceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "refresh failed", err)
			return
		}

		if srv != nil && result != nil {
			srv.saveRefreshStatus(ns, pluginName, sourceID, result)
		}

		// Invalidate cache after synchronous refresh completes.
		if srv != nil && srv.GetCacheManager() != nil {
			srv.GetCacheManager().InvalidatePlugin(pluginName)
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

// entityGetterHandler returns a handler that retrieves a single entity by name
// using the plugin's EntityGetter interface. This provides a standardized
// single-path-param get endpoint for plugins whose native get endpoint
// requires multiple path parameters.
func entityGetterHandler(eg EntityGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityName := chi.URLParam(r, "entityName")
		if entityName == "" {
			writeError(w, http.StatusBadRequest, "missing entity name", nil)
			return
		}

		// Use empty entityKind — the plugin can infer from context or handle all kinds.
		result, err := eg.GetEntityByName(r.Context(), "", entityName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get entity", err)
			return
		}

		if result == nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("entity %q not found", entityName), nil)
			return
		}

		writeJSON(w, http.StatusOK, result)
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
