package governance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// getGovernanceHandler returns a handler that retrieves the governance overlay
// for an asset. Creates a default record if none exists.
func getGovernanceHandler(store *GovernanceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")
		ns := tenancy.NamespaceFromContext(r.Context())

		actor := extractActor(r)
		record, err := store.EnsureExists(ns, pluginName, kind, name, "", actor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get governance record: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, GovernanceResponse{
			AssetRef: AssetRef{
				Plugin: pluginName,
				Kind:   kind,
				Name:   name,
			},
			Governance: recordToOverlay(record),
		})
	}
}

// patchGovernanceHandler returns a handler that updates governance metadata.
// Only fields present in the request body are updated. Emits an audit event.
func patchGovernanceHandler(store *GovernanceStore, auditStore *AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")
		ns := tenancy.NamespaceFromContext(r.Context())

		var overlay GovernanceOverlay
		if err := json.NewDecoder(r.Body).Decode(&overlay); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		actor := extractActor(r)

		// Load or create the existing record.
		record, err := store.EnsureExists(ns, pluginName, kind, name, "", actor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load governance record: %v", err))
			return
		}

		// Capture old state for audit.
		oldOverlay := recordToOverlay(record)

		// Apply non-nil fields from the request.
		applyOverlay(record, &overlay, actor)

		// Save the updated record.
		if err := store.Upsert(record); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to save governance record: %v", err))
			return
		}

		// Emit audit event.
		newOverlay := recordToOverlay(record)
		auditEvent := &AuditEventRecord{
			ID:        uuid.New().String(),
			Namespace: ns,
			EventType: "governance.metadata.changed",
			Actor:     actor,
			AssetUID:  record.AssetUID,
			Action:    "patch",
			Outcome:   "success",
			OldValue:  overlayToMap(oldOverlay),
			NewValue:  overlayToMap(newOverlay),
			CreatedAt: time.Now(),
		}
		// Best-effort audit; don't fail the request if audit write fails.
		_ = auditStore.Append(auditEvent)

		writeJSON(w, http.StatusOK, GovernanceResponse{
			AssetRef: AssetRef{
				Plugin: pluginName,
				Kind:   kind,
				Name:   name,
			},
			Governance: newOverlay,
		})
	}
}

// getHistoryHandler returns a handler that lists paginated audit events for an asset.
func getHistoryHandler(store *GovernanceStore, auditStore *AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")
		ns := tenancy.NamespaceFromContext(r.Context())

		// Resolve asset UID from the governance record.
		record, err := store.Get(ns, pluginName, kind, name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load governance record: %v", err))
			return
		}
		if record == nil {
			writeJSON(w, http.StatusOK, AuditEventList{
				Events:    []AuditEvent{},
				TotalSize: 0,
			})
			return
		}

		pageSize := 20
		if ps := r.URL.Query().Get("pageSize"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil && v > 0 {
				pageSize = v
			}
		}
		pageToken := r.URL.Query().Get("pageToken")

		records, nextToken, total, err := auditStore.ListByAsset(ns, record.AssetUID, pageSize, pageToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list audit events: %v", err))
			return
		}

		events := make([]AuditEvent, len(records))
		for i, rec := range records {
			events[i] = recordToAuditEvent(rec)
		}

		writeJSON(w, http.StatusOK, AuditEventList{
			Events:        events,
			NextPageToken: nextToken,
			TotalSize:     total,
		})
	}
}

// combinedActionHandler returns a handler that dispatches both lifecycle and promotion
// governance actions. Lifecycle actions (lifecycle.*) go to the lifecycle handler.
// Promotion actions (version.*, promotion.*) go to the promotion handler.
// POST /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/actions/{action}
func combinedActionHandler(lifecycleHandler *LifecycleActionHandler, promotionHandler *PromotionActionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")
		action := chi.URLParam(r, "action")
		ns := tenancy.NamespaceFromContext(r.Context())

		actor := extractActor(r)

		var req struct {
			DryRun bool           `json:"dryRun"`
			Params map[string]any `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Params == nil {
			req.Params = make(map[string]any)
		}

		var result *ActionResult
		var err error

		switch {
		case isLifecycleAction(action):
			result, err = lifecycleHandler.HandleAction(r.Context(), ns, pluginName, kind, name, actor, action, req.Params, req.DryRun)
		case isPromotionAction(action):
			if promotionHandler == nil {
				writeError(w, http.StatusNotImplemented, "versioning and promotion are not enabled")
				return
			}
			result, err = promotionHandler.HandleAction(r.Context(), ns, pluginName, kind, name, actor, action, req.Params, req.DryRun)
		default:
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown action: %s", action))
			return
		}

		if err != nil {
			if te, ok := err.(*TransitionError); ok {
				writeJSON(w, http.StatusBadRequest, te)
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		status := http.StatusOK
		if result.Status == "pending-approval" {
			status = http.StatusAccepted
		}
		writeJSON(w, status, result)
	}
}

// isLifecycleAction returns true for lifecycle-related actions.
func isLifecycleAction(action string) bool {
	switch action {
	case "lifecycle.setState", "lifecycle.deprecate", "lifecycle.archive", "lifecycle.restore":
		return true
	}
	return false
}

// isPromotionAction returns true for version/promotion-related actions.
func isPromotionAction(action string) bool {
	switch action {
	case "version.create", "promotion.bind", "promotion.promote", "promotion.rollback":
		return true
	}
	return false
}

// extractActor extracts the actor from the request headers.
// Prefers X-User-Principal over X-User-Role, falls back to "system".
func extractActor(r *http.Request) string {
	if principal := r.Header.Get("X-User-Principal"); principal != "" {
		return principal
	}
	if role := r.Header.Get("X-User-Role"); role != "" {
		return role
	}
	return "system"
}

// applyOverlay applies non-nil fields from the overlay to the record.
func applyOverlay(record *AssetGovernanceRecord, overlay *GovernanceOverlay, actor string) {
	if overlay.Owner != nil {
		record.OwnerPrincipal = overlay.Owner.Principal
		record.OwnerDisplayName = overlay.Owner.DisplayName
		record.OwnerEmail = overlay.Owner.Email
	}
	if overlay.Team != nil {
		record.TeamName = overlay.Team.Name
		record.TeamID = overlay.Team.ID
	}
	if overlay.SLA != nil {
		record.SLATier = string(overlay.SLA.Tier)
		record.SLAResponseHours = overlay.SLA.ResponseHours
	}
	if overlay.Risk != nil {
		record.RiskLevel = string(overlay.Risk.Level)
		record.RiskCategories = JSONStringSlice(overlay.Risk.Categories)
	}
	if overlay.IntendedUse != nil {
		record.IntendedUseSummary = overlay.IntendedUse.Summary
		record.IntendedUseEnvs = JSONStringSlice(overlay.IntendedUse.Environments)
		record.IntendedUseRestrictions = JSONStringSlice(overlay.IntendedUse.Restrictions)
	}
	if overlay.Compliance != nil {
		record.ComplianceTags = JSONStringSlice(overlay.Compliance.Tags)
		record.ComplianceControls = JSONStringSlice(overlay.Compliance.Controls)
	}
	if overlay.Lifecycle != nil {
		record.LifecycleState = string(overlay.Lifecycle.State)
		record.LifecycleReason = overlay.Lifecycle.Reason
		record.LifecycleChangedBy = actor
		now := time.Now()
		record.LifecycleChangedAt = &now
	}
	if overlay.Audit != nil {
		if overlay.Audit.LastReviewedAt != "" {
			if t, err := time.Parse(time.RFC3339, overlay.Audit.LastReviewedAt); err == nil {
				record.AuditLastReviewedAt = &t
			}
		}
		record.AuditReviewCadenceDays = overlay.Audit.ReviewCadenceDays
	}
}

// recordToOverlay converts a governance record to an API overlay.
func recordToOverlay(record *AssetGovernanceRecord) GovernanceOverlay {
	overlay := GovernanceOverlay{}

	if record.OwnerPrincipal != "" || record.OwnerDisplayName != "" || record.OwnerEmail != "" {
		overlay.Owner = &OwnerInfo{
			Principal:   record.OwnerPrincipal,
			DisplayName: record.OwnerDisplayName,
			Email:       record.OwnerEmail,
		}
	}

	if record.TeamName != "" || record.TeamID != "" {
		overlay.Team = &TeamInfo{
			Name: record.TeamName,
			ID:   record.TeamID,
		}
	}

	if record.SLATier != "" {
		overlay.SLA = &SLAInfo{
			Tier:          SLATier(record.SLATier),
			ResponseHours: record.SLAResponseHours,
		}
	}

	overlay.Risk = &RiskInfo{
		Level:      RiskLevel(record.RiskLevel),
		Categories: []string(record.RiskCategories),
	}

	if record.IntendedUseSummary != "" || len(record.IntendedUseEnvs) > 0 || len(record.IntendedUseRestrictions) > 0 {
		overlay.IntendedUse = &IntendedUse{
			Summary:      record.IntendedUseSummary,
			Environments: []string(record.IntendedUseEnvs),
			Restrictions: []string(record.IntendedUseRestrictions),
		}
	}

	if len(record.ComplianceTags) > 0 || len(record.ComplianceControls) > 0 {
		overlay.Compliance = &ComplianceInfo{
			Tags:     []string(record.ComplianceTags),
			Controls: []string(record.ComplianceControls),
		}
	}

	var changedAt string
	if record.LifecycleChangedAt != nil {
		changedAt = record.LifecycleChangedAt.Format(time.RFC3339)
	}
	overlay.Lifecycle = &LifecycleInfo{
		State:     LifecycleState(record.LifecycleState),
		Reason:    record.LifecycleReason,
		ChangedBy: record.LifecycleChangedBy,
		ChangedAt: changedAt,
	}

	if record.AuditLastReviewedAt != nil || record.AuditReviewCadenceDays > 0 {
		var lastReviewed string
		if record.AuditLastReviewedAt != nil {
			lastReviewed = record.AuditLastReviewedAt.Format(time.RFC3339)
		}
		overlay.Audit = &AuditMetadata{
			LastReviewedAt:    lastReviewed,
			ReviewCadenceDays: record.AuditReviewCadenceDays,
		}
	}

	return overlay
}

// recordToAuditEvent converts an audit event record to the API type.
func recordToAuditEvent(rec AuditEventRecord) AuditEvent {
	return AuditEvent{
		ID:            rec.ID,
		CorrelationID: rec.CorrelationID,
		EventType:     rec.EventType,
		Actor:         rec.Actor,
		AssetUID:      rec.AssetUID,
		VersionID:     rec.VersionID,
		Action:        rec.Action,
		Outcome:       rec.Outcome,
		Reason:        rec.Reason,
		OldValue:      map[string]any(rec.OldValue),
		NewValue:      map[string]any(rec.NewValue),
		Metadata:      map[string]any(rec.EventMetadata),
		CreatedAt:     rec.CreatedAt.Format(time.RFC3339),
	}
}

// overlayToMap converts a GovernanceOverlay to a map for audit event storage.
func overlayToMap(overlay GovernanceOverlay) JSONAny {
	data, err := json.Marshal(overlay)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return JSONAny(m)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
