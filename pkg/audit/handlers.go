package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/model-registry/pkg/catalog/governance"
)

// ListEventsHandler handles GET /api/audit/v1alpha1/events
// Query params: namespace, actor, plugin, action, eventType, pageSize, pageToken
func ListEventsHandler(store *governance.AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := governance.AuditListFilter{
			Namespace: r.URL.Query().Get("namespace"),
			Actor:     r.URL.Query().Get("actor"),
			Plugin:    r.URL.Query().Get("plugin"),
			Action:    r.URL.Query().Get("action"),
			EventType: r.URL.Query().Get("eventType"),
		}

		pageSize := 20
		if ps := r.URL.Query().Get("pageSize"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil && v > 0 {
				pageSize = v
			}
		}
		pageToken := r.URL.Query().Get("pageToken")

		records, nextToken, total, err := store.ListFiltered(filter, pageSize, pageToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list audit events: %v", err))
			return
		}

		events := make([]auditEventResponse, len(records))
		for i, rec := range records {
			events[i] = recordToResponse(rec)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"events":        events,
			"nextPageToken": nextToken,
			"totalSize":     total,
		})
	}
}

// GetEventHandler handles GET /api/audit/v1alpha1/events/{eventId}
func GetEventHandler(store *governance.AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID := chi.URLParam(r, "eventId")
		if eventID == "" {
			writeError(w, http.StatusBadRequest, "missing event ID")
			return
		}

		record, err := store.GetByID(eventID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get audit event: %v", err))
			return
		}
		if record == nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("audit event %q not found", eventID))
			return
		}

		writeJSON(w, http.StatusOK, recordToResponse(*record))
	}
}

// auditEventResponse is the API response for an audit event, including V2 fields.
type auditEventResponse struct {
	ID            string         `json:"id"`
	Namespace     string         `json:"namespace"`
	CorrelationID string         `json:"correlationId,omitempty"`
	EventType     string         `json:"eventType"`
	Actor         string         `json:"actor"`
	RequestID     string         `json:"requestId,omitempty"`
	Plugin        string         `json:"plugin,omitempty"`
	ResourceType  string         `json:"resourceType,omitempty"`
	ResourceIDs   []string       `json:"resourceIds,omitempty"`
	AssetUID      string         `json:"assetUid,omitempty"`
	VersionID     string         `json:"versionId,omitempty"`
	Action        string         `json:"action,omitempty"`
	ActionVerb    string         `json:"actionVerb,omitempty"`
	Outcome       string         `json:"outcome"`
	StatusCode    int            `json:"statusCode,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	OldValue      map[string]any `json:"oldValue,omitempty"`
	NewValue      map[string]any `json:"newValue,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     string         `json:"createdAt"`
}

func recordToResponse(rec governance.AuditEventRecord) auditEventResponse {
	return auditEventResponse{
		ID:            rec.ID,
		Namespace:     rec.Namespace,
		CorrelationID: rec.CorrelationID,
		EventType:     rec.EventType,
		Actor:         rec.Actor,
		RequestID:     rec.RequestID,
		Plugin:        rec.Plugin,
		ResourceType:  rec.ResourceType,
		ResourceIDs:   []string(rec.ResourceIDs),
		AssetUID:      rec.AssetUID,
		VersionID:     rec.VersionID,
		Action:        rec.Action,
		ActionVerb:    rec.ActionVerb,
		Outcome:       rec.Outcome,
		StatusCode:    rec.StatusCode,
		Reason:        rec.Reason,
		OldValue:      map[string]any(rec.OldValue),
		NewValue:      map[string]any(rec.NewValue),
		Metadata:      map[string]any(rec.EventMetadata),
		CreatedAt:     rec.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
