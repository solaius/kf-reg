package governance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// VersionResponse is the API response for a version.
type VersionResponse struct {
	VersionID     string          `json:"versionId"`
	VersionLabel  string          `json:"versionLabel"`
	CreatedAt     string          `json:"createdAt"`
	CreatedBy     string          `json:"createdBy"`
	ContentDigest string          `json:"contentDigest,omitempty"`
	Provenance    *ProvenanceInfo `json:"provenance,omitempty"`
}

// VersionListResponse is a paginated list of versions.
type VersionListResponse struct {
	Versions      []VersionResponse `json:"versions"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
	TotalSize     int               `json:"totalSize"`
}

// BindingResponse is the API response for a binding.
type BindingResponse struct {
	Environment       string `json:"environment"`
	VersionID         string `json:"versionId"`
	BoundAt           string `json:"boundAt"`
	BoundBy           string `json:"boundBy"`
	PreviousVersionID string `json:"previousVersionId,omitempty"`
}

// BindingsResponse is the API response for all bindings.
type BindingsResponse struct {
	Bindings []BindingResponse `json:"bindings"`
}

// SetBindingResponse extends BindingResponse with optional warnings.
type SetBindingResponse struct {
	BindingResponse
	Warnings []string `json:"warnings,omitempty"`
}

// ProvenanceInfo describes where an asset version came from.
type ProvenanceInfo struct {
	SourceType string         `json:"sourceType,omitempty"`
	SourceURI  string         `json:"sourceUri,omitempty"`
	SourceID   string         `json:"sourceId,omitempty"`
	RevisionID string         `json:"revisionId,omitempty"`
	ObservedAt string         `json:"observedAt,omitempty"`
	Integrity  *IntegrityInfo `json:"integrity,omitempty"`
}

// IntegrityInfo describes the integrity verification status of a version.
type IntegrityInfo struct {
	Verified bool   `json:"verified"`
	Method   string `json:"method,omitempty"`
	Details  string `json:"details,omitempty"`
}

// listVersionsHandler returns a handler that lists paginated versions for an asset.
// GET /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/versions
func listVersionsHandler(versionStore *VersionStore, govStore *GovernanceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")

		// Get governance record to find assetUID.
		record, err := govStore.Get(pluginName, kind, name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load governance record: %v", err))
			return
		}
		if record == nil {
			writeJSON(w, http.StatusOK, VersionListResponse{
				Versions:  []VersionResponse{},
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

		records, nextToken, total, err := versionStore.ListVersions(record.AssetUID, pageSize, pageToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list versions: %v", err))
			return
		}

		versions := make([]VersionResponse, len(records))
		for i, rec := range records {
			versions[i] = versionRecordToResponse(rec)
		}

		writeJSON(w, http.StatusOK, VersionListResponse{
			Versions:      versions,
			NextPageToken: nextToken,
			TotalSize:     total,
		})
	}
}

// createVersionHandler returns a handler that creates an immutable version snapshot.
// POST /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/versions
// The optional provenanceExtractor populates provenance fields on the version record.
func createVersionHandler(versionStore *VersionStore, govStore *GovernanceStore, auditStore *AuditStore, provenanceExtractor ...ProvenanceExtractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")
		actor := extractActor(r)

		var req struct {
			VersionLabel string `json:"versionLabel"`
			Reason       string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		if req.VersionLabel == "" {
			writeError(w, http.StatusBadRequest, "versionLabel is required")
			return
		}

		// Get or create governance record.
		uid := fmt.Sprintf("%s:%s:%s", pluginName, kind, name)
		govRecord, err := govStore.EnsureExists(pluginName, kind, name, uid, actor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get governance record: %v", err))
			return
		}

		// Serialize governance state as snapshot.
		govSnapshot := overlayToMap(recordToOverlay(govRecord))

		versionRecord := &AssetVersionRecord{
			ID:                 uuid.New().String(),
			AssetUID:           govRecord.AssetUID,
			VersionID:          fmt.Sprintf("%s:%s", req.VersionLabel, uuid.New().String()[:8]),
			VersionLabel:       req.VersionLabel,
			CreatedBy:          actor,
			GovernanceSnapshot: govSnapshot,
			AssetSnapshot:      JSONAny{},
		}

		// Populate provenance fields if an extractor is available.
		if len(provenanceExtractor) > 0 && provenanceExtractor[0] != nil {
			applyProvenance(versionRecord, provenanceExtractor[0].ExtractProvenance(pluginName, kind, name))
		}

		if err := versionStore.CreateVersion(versionRecord); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create version: %v", err))
			return
		}

		// Emit audit event.
		_ = auditStore.Append(&AuditEventRecord{
			ID:            uuid.New().String(),
			CorrelationID: uuid.New().String(),
			EventType:     "governance.version.created",
			Actor:         actor,
			AssetUID:      govRecord.AssetUID,
			VersionID:     versionRecord.VersionID,
			Action:        "version.create",
			Outcome:       "success",
			Reason:        req.Reason,
			NewValue: JSONAny{
				"versionId":    versionRecord.VersionID,
				"versionLabel": versionRecord.VersionLabel,
			},
		})

		writeJSON(w, http.StatusCreated, versionRecordToResponse(*versionRecord))
	}
}

// listBindingsHandler returns a handler that lists all environment bindings for an asset.
// GET /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/bindings
func listBindingsHandler(bindingStore *BindingStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")

		records, err := bindingStore.ListBindings(pluginName, kind, name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list bindings: %v", err))
			return
		}

		bindings := make([]BindingResponse, len(records))
		for i, rec := range records {
			bindings[i] = bindingRecordToResponse(rec)
		}

		writeJSON(w, http.StatusOK, BindingsResponse{
			Bindings: bindings,
		})
	}
}

// setBindingHandler returns a handler that sets a version binding for an environment.
// PUT /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/bindings/{environment}
func setBindingHandler(bindingStore *BindingStore, versionStore *VersionStore, govStore *GovernanceStore, auditStore *AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := chi.URLParam(r, "plugin")
		kind := chi.URLParam(r, "kind")
		name := chi.URLParam(r, "name")
		environment := chi.URLParam(r, "environment")
		actor := extractActor(r)

		var req struct {
			VersionID string `json:"versionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		if req.VersionID == "" {
			writeError(w, http.StatusBadRequest, "versionId is required")
			return
		}

		// Get governance record.
		govRecord, err := govStore.Get(pluginName, kind, name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load governance record: %v", err))
			return
		}
		if govRecord == nil {
			writeError(w, http.StatusNotFound, "governance record not found")
			return
		}

		// Check lifecycle state constraints.
		var warnings []string
		state := LifecycleState(govRecord.LifecycleState)
		switch state {
		case StateArchived:
			writeError(w, http.StatusBadRequest, "archived assets cannot be bound")
			return
		case StateDraft:
			if environment == "stage" || environment == "prod" {
				writeError(w, http.StatusBadRequest, "draft assets cannot be bound to stage/prod")
				return
			}
		case StateDeprecated:
			warnings = append(warnings, "asset is deprecated; binding is allowed but consider migrating")
		}

		// Verify version exists.
		version, err := versionStore.GetVersion(req.VersionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to verify version: %v", err))
			return
		}
		if version == nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("version %s not found", req.VersionID))
			return
		}

		// Get current binding for this env (if any) to record previous_version_id.
		var previousVersionID string
		existing, err := bindingStore.GetBinding(pluginName, kind, name, environment)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check existing binding: %v", err))
			return
		}
		if existing != nil {
			previousVersionID = existing.VersionID
		}

		now := time.Now()
		bindingRecord := &EnvBindingRecord{
			ID:                uuid.New().String(),
			Plugin:            pluginName,
			AssetKind:         kind,
			AssetName:         name,
			Environment:       environment,
			AssetUID:          govRecord.AssetUID,
			VersionID:         req.VersionID,
			BoundAt:           now,
			BoundBy:           actor,
			PreviousVersionID: previousVersionID,
		}

		if err := bindingStore.SetBinding(bindingRecord); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to set binding: %v", err))
			return
		}

		// Emit audit event.
		_ = auditStore.Append(&AuditEventRecord{
			ID:            uuid.New().String(),
			CorrelationID: uuid.New().String(),
			EventType:     "governance.promotion.bound",
			Actor:         actor,
			AssetUID:      govRecord.AssetUID,
			VersionID:     req.VersionID,
			Action:        "promotion.bind",
			Outcome:       "success",
			OldValue:      JSONAny{"versionId": previousVersionID, "environment": environment},
			NewValue:      JSONAny{"versionId": req.VersionID, "environment": environment},
		})

		resp := SetBindingResponse{
			BindingResponse: bindingRecordToResponse(*bindingRecord),
			Warnings:        warnings,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// versionRecordToResponse converts a version record to an API response.
func versionRecordToResponse(rec AssetVersionRecord) VersionResponse {
	resp := VersionResponse{
		VersionID:     rec.VersionID,
		VersionLabel:  rec.VersionLabel,
		CreatedAt:     rec.CreatedAt.Format(time.RFC3339),
		CreatedBy:     rec.CreatedBy,
		ContentDigest: rec.ContentDigest,
	}

	// Include provenance info if any fields are populated.
	if rec.ProvenanceSourceType != "" || rec.ProvenanceSourceURI != "" || rec.ProvenanceSourceID != "" {
		prov := &ProvenanceInfo{
			SourceType: rec.ProvenanceSourceType,
			SourceURI:  rec.ProvenanceSourceURI,
			SourceID:   rec.ProvenanceSourceID,
			RevisionID: rec.ProvenanceRevisionID,
		}
		if rec.ProvenanceRevObservedAt != nil {
			prov.ObservedAt = rec.ProvenanceRevObservedAt.Format(time.RFC3339)
		}
		if rec.ProvenanceVerified || rec.ProvenanceMethod != "" {
			prov.Integrity = &IntegrityInfo{
				Verified: rec.ProvenanceVerified,
				Method:   rec.ProvenanceMethod,
				Details:  rec.ProvenanceDetails,
			}
		}
		resp.Provenance = prov
	}

	return resp
}

// bindingRecordToResponse converts a binding record to an API response.
func bindingRecordToResponse(rec EnvBindingRecord) BindingResponse {
	return BindingResponse{
		Environment:       rec.Environment,
		VersionID:         rec.VersionID,
		BoundAt:           rec.BoundAt.Format(time.RFC3339),
		BoundBy:           rec.BoundBy,
		PreviousVersionID: rec.PreviousVersionID,
	}
}
