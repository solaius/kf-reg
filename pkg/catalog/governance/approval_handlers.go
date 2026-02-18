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

// listApprovalsHandler returns a handler that lists approval requests with optional filtering.
// GET /api/governance/v1alpha1/approvals?status=pending&assetUid=...&pageSize=20&pageToken=...
func listApprovalsHandler(approvalStore *ApprovalStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns := tenancy.NamespaceFromContext(r.Context())
		status := ApprovalStatus(r.URL.Query().Get("status"))
		assetUID := r.URL.Query().Get("assetUid")

		pageSize := 20
		if ps := r.URL.Query().Get("pageSize"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil && v > 0 {
				pageSize = v
			}
		}
		pageToken := r.URL.Query().Get("pageToken")

		records, nextToken, total, err := approvalStore.List(ns, status, assetUID, pageSize, pageToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list approvals: %v", err))
			return
		}

		requests := make([]ApprovalRequest, len(records))
		for i, rec := range records {
			requests[i] = recordToApprovalRequest(&rec, nil)
		}

		writeJSON(w, http.StatusOK, ApprovalRequestList{
			Requests:      requests,
			NextPageToken: nextToken,
			TotalSize:     total,
		})
	}
}

// getApprovalHandler returns a handler that retrieves a single approval request with decisions.
// GET /api/governance/v1alpha1/approvals/{id}
func getApprovalHandler(approvalStore *ApprovalStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		req, decisions, err := approvalStore.Get(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get approval: %v", err))
			return
		}
		if req == nil {
			writeError(w, http.StatusNotFound, "approval request not found")
			return
		}

		writeJSON(w, http.StatusOK, recordToApprovalRequest(req, decisions))
	}
}

// submitDecisionHandler returns a handler that adds a reviewer decision to an approval request.
// POST /api/governance/v1alpha1/approvals/{id}/decisions
//
// When a decision meets the policy gate threshold, the original lifecycle action is
// auto-executed atomically.
func submitDecisionHandler(
	approvalStore *ApprovalStore,
	evaluator *ApprovalEvaluator,
	lifecycleHandler *LifecycleActionHandler,
	auditStore *AuditStore,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		actor := extractActor(r)

		var body struct {
			Verdict DecisionVerdict `json:"verdict"`
			Comment string          `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.Verdict != VerdictApprove && body.Verdict != VerdictDeny {
			writeError(w, http.StatusBadRequest, "verdict must be 'approve' or 'deny'")
			return
		}

		req, _, err := approvalStore.Get(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get approval: %v", err))
			return
		}
		if req == nil {
			writeError(w, http.StatusNotFound, "approval request not found")
			return
		}
		if req.Status != ApprovalStatusPending {
			writeError(w, http.StatusConflict, fmt.Sprintf("approval request is already %s", req.Status))
			return
		}

		// Don't allow the requester to approve their own request.
		if body.Verdict == VerdictApprove && actor == req.Requester {
			writeError(w, http.StatusForbidden, "requester cannot approve their own request")
			return
		}

		decision := &ApprovalDecisionRecord{
			ID:        uuid.New().String(),
			RequestID: id,
			Reviewer:  actor,
			Verdict:   body.Verdict,
			Comment:   body.Comment,
		}
		if err := approvalStore.AddDecision(decision); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add decision: %v", err))
			return
		}

		// Audit the decision.
		_ = auditStore.Append(&AuditEventRecord{
			ID:            uuid.New().String(),
			Namespace:     req.Namespace,
			CorrelationID: id,
			EventType:     "governance.approval.decision",
			Actor:         actor,
			AssetUID:      req.AssetUID,
			Action:        fmt.Sprintf("approval.%s", body.Verdict),
			Outcome:       "success",
			Reason:        body.Comment,
			NewValue:      JSONAny{"verdict": string(body.Verdict), "requestId": id},
		})

		// Check if the gate threshold is now met.
		approves, denies, err := approvalStore.CountDecisions(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to count decisions: %v", err))
			return
		}

		resolved := evaluator.EvaluateDecisions(req.PolicyID, approves, denies)

		if resolved == ApprovalStatusApproved {
			// Auto-execute the original action.
			if err := approvalStore.UpdateStatus(id, ApprovalStatusApproved, actor, "threshold met"); err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update approval status: %v", err))
				return
			}

			params := map[string]any(req.ActionParams)
			if params == nil {
				params = make(map[string]any)
			}

			result, execErr := lifecycleHandler.executeTransition(
				req.Namespace, req.Plugin, req.AssetKind, req.AssetName, req.Requester, params, req.Reason,
			)
			if execErr != nil {
				// The approval succeeded but execution failed; record this for debugging.
				_ = auditStore.Append(&AuditEventRecord{
					ID:            uuid.New().String(),
					Namespace:     req.Namespace,
					CorrelationID: id,
					EventType:     "governance.approval.execution_failed",
					Actor:         "system",
					AssetUID:      req.AssetUID,
					Action:        req.Action,
					Outcome:       "failure",
					Reason:        execErr.Error(),
				})
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("approved but execution failed: %v", execErr))
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"decision":    "approved",
				"requestId":   id,
				"autoExecuted": true,
				"result":      result,
			})
			return
		}

		if resolved == ApprovalStatusDenied {
			if err := approvalStore.UpdateStatus(id, ApprovalStatusDenied, actor, body.Comment); err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update approval status: %v", err))
				return
			}

			_ = auditStore.Append(&AuditEventRecord{
				ID:            uuid.New().String(),
				Namespace:     req.Namespace,
				CorrelationID: id,
				EventType:     "governance.approval.denied",
				Actor:         actor,
				AssetUID:      req.AssetUID,
				Action:        req.Action,
				Outcome:       "denied",
				Reason:        body.Comment,
			})

			writeJSON(w, http.StatusOK, map[string]any{
				"decision":  "denied",
				"requestId": id,
			})
			return
		}

		// Still pending.
		writeJSON(w, http.StatusOK, map[string]any{
			"decision":      string(body.Verdict),
			"requestId":     id,
			"status":        "pending",
			"approvesCount": approves,
			"deniesCount":   denies,
			"requiredCount": req.RequiredCount,
		})
	}
}

// cancelApprovalHandler returns a handler that cancels a pending approval request.
// POST /api/governance/v1alpha1/approvals/{id}/cancel
func cancelApprovalHandler(approvalStore *ApprovalStore, auditStore *AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		actor := extractActor(r)

		var body struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req, _, err := approvalStore.Get(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get approval: %v", err))
			return
		}
		if req == nil {
			writeError(w, http.StatusNotFound, "approval request not found")
			return
		}

		if err := approvalStore.Cancel(id, actor, body.Reason); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		_ = auditStore.Append(&AuditEventRecord{
			ID:            uuid.New().String(),
			Namespace:     req.Namespace,
			CorrelationID: id,
			EventType:     "governance.approval.canceled",
			Actor:         actor,
			AssetUID:      req.AssetUID,
			Action:        req.Action,
			Outcome:       "canceled",
			Reason:        body.Reason,
		})

		writeJSON(w, http.StatusOK, map[string]string{
			"status":    "canceled",
			"requestId": id,
		})
	}
}

// listPoliciesHandler returns a handler that lists all loaded approval policies.
// GET /api/governance/v1alpha1/policies
func listPoliciesHandler(evaluator *ApprovalEvaluator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"policies": evaluator.ListPolicies(),
		})
	}
}

// recordToApprovalRequest converts a record to the API type.
func recordToApprovalRequest(rec *ApprovalRequestRecord, decisions []ApprovalDecisionRecord) ApprovalRequest {
	ar := ApprovalRequest{
		ID: rec.ID,
		AssetRef: AssetRef{
			Plugin: rec.Plugin,
			Kind:   rec.AssetKind,
			Name:   rec.AssetName,
		},
		Action:         rec.Action,
		ActionParams:   map[string]any(rec.ActionParams),
		PolicyID:       rec.PolicyID,
		RequiredCount:  rec.RequiredCount,
		Status:         rec.Status,
		Requester:      rec.Requester,
		Reason:         rec.Reason,
		ResolvedBy:     rec.ResolvedBy,
		ResolutionNote: rec.ResolutionNote,
		CreatedAt:      rec.CreatedAt.Format(time.RFC3339),
	}
	if rec.ResolvedAt != nil {
		ar.ResolvedAt = rec.ResolvedAt.Format(time.RFC3339)
	}
	if rec.ExpiresAt != nil {
		ar.ExpiresAt = rec.ExpiresAt.Format(time.RFC3339)
	}

	if decisions != nil {
		ar.Decisions = make([]ApprovalDecision, len(decisions))
		for i, d := range decisions {
			ar.Decisions[i] = ApprovalDecision{
				ID:        d.ID,
				RequestID: d.RequestID,
				Reviewer:  d.Reviewer,
				Verdict:   d.Verdict,
				Comment:   d.Comment,
				CreatedAt: d.CreatedAt.Format(time.RFC3339),
			}
		}
	}

	return ar
}
