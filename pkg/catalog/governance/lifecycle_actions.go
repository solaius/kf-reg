package governance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ActionResult represents the outcome of a governance action.
type ActionResult struct {
	Action  string         `json:"action"`
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

// ActionDefinition describes a governance action available on assets.
type ActionDefinition struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	Destructive bool   `json:"destructive,omitempty"`
}

// LifecycleActionHandler handles lifecycle governance actions.
type LifecycleActionHandler struct {
	store         *GovernanceStore
	auditStore    *AuditStore
	approvalStore *ApprovalStore
	evaluator     *ApprovalEvaluator
	machine       *LifecycleMachine
}

// NewLifecycleActionHandler creates a lifecycle action handler.
func NewLifecycleActionHandler(store *GovernanceStore, auditStore *AuditStore) *LifecycleActionHandler {
	return &LifecycleActionHandler{
		store:      store,
		auditStore: auditStore,
		machine:    NewLifecycleMachine(),
	}
}

// SetApprovalEngine configures the approval store and evaluator for gated transitions.
func (h *LifecycleActionHandler) SetApprovalEngine(approvalStore *ApprovalStore, evaluator *ApprovalEvaluator) {
	h.approvalStore = approvalStore
	h.evaluator = evaluator
}

// HandleAction dispatches a lifecycle action.
// Supported actions: lifecycle.setState, lifecycle.deprecate, lifecycle.archive, lifecycle.restore
func (h *LifecycleActionHandler) HandleAction(ctx context.Context, plugin, kind, name, actor, action string, params map[string]any, dryRun bool) (*ActionResult, error) {
	switch action {
	case "lifecycle.setState":
		return h.handleSetState(ctx, plugin, kind, name, actor, params, dryRun)
	case "lifecycle.deprecate":
		return h.handleDeprecate(ctx, plugin, kind, name, actor, params, dryRun)
	case "lifecycle.archive":
		return h.handleArchive(ctx, plugin, kind, name, actor, params, dryRun)
	case "lifecycle.restore":
		return h.handleRestore(ctx, plugin, kind, name, actor, params, dryRun)
	default:
		return nil, fmt.Errorf("unknown lifecycle action: %s", action)
	}
}

func (h *LifecycleActionHandler) handleSetState(ctx context.Context, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	stateStr, ok := params["state"].(string)
	if !ok || stateStr == "" {
		return nil, fmt.Errorf("missing or invalid 'state' parameter")
	}
	toState := LifecycleState(stateStr)
	reason, _ := params["reason"].(string)

	// Get or create governance record.
	uid := fmt.Sprintf("%s:%s:%s", plugin, kind, name)
	record, err := h.store.EnsureExists(plugin, kind, name, uid, actor)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance record: %w", err)
	}

	fromState := LifecycleState(record.LifecycleState)

	// Validate transition.
	if err := h.machine.ValidateTransition(fromState, toState); err != nil {
		return nil, err
	}

	// Check if approval policy requires a gate for this transition.
	needsApprovalGate := false
	var evalResult EvaluationResult
	if h.evaluator != nil && h.approvalStore != nil && h.machine.RequiresApproval(fromState, toState) {
		evalResult = h.evaluator.Evaluate(plugin, kind, record.RiskLevel, fromState, toState)
		needsApprovalGate = evalResult.RequiresApproval
	}

	if dryRun {
		data := map[string]any{
			"from":             string(fromState),
			"to":               string(toState),
			"requiresApproval": needsApprovalGate,
		}
		if needsApprovalGate {
			data["policyId"] = evalResult.PolicyID
			data["requiredCount"] = evalResult.RequiredCount
		}
		return &ActionResult{
			Action:  "lifecycle.setState",
			Status:  "dry-run",
			Message: fmt.Sprintf("would transition %s from %s to %s", name, fromState, toState),
			Data:    data,
		}, nil
	}

	// If an approval gate applies, create a pending request instead of executing.
	if needsApprovalGate {
		return h.createApprovalRequest(plugin, kind, name, actor, reason, record.AssetUID, params, evalResult)
	}

	// No approval gate -- execute the transition directly.
	return h.executeTransition(plugin, kind, name, actor, params, reason)
}

// createApprovalRequest creates a pending approval request and returns a 202-style result.
func (h *LifecycleActionHandler) createApprovalRequest(
	plugin, kind, name, actor, reason, assetUID string,
	params map[string]any,
	evalResult EvaluationResult,
) (*ActionResult, error) {
	reqID := uuid.New().String()

	var expiresAt *time.Time
	if evalResult.ExpiryHours > 0 {
		t := time.Now().Add(time.Duration(evalResult.ExpiryHours) * time.Hour)
		expiresAt = &t
	}

	rec := &ApprovalRequestRecord{
		ID:            reqID,
		AssetUID:      assetUID,
		Plugin:        plugin,
		AssetKind:     kind,
		AssetName:     name,
		Action:        "lifecycle.setState",
		ActionParams:  JSONAny(params),
		PolicyID:      evalResult.PolicyID,
		RequiredCount: evalResult.RequiredCount,
		Status:        ApprovalStatusPending,
		Requester:     actor,
		Reason:        reason,
		ExpiresAt:     expiresAt,
	}

	if err := h.approvalStore.Create(rec); err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	_ = h.auditStore.Append(&AuditEventRecord{
		ID:            uuid.New().String(),
		CorrelationID: reqID,
		EventType:     "governance.approval.requested",
		Actor:         actor,
		AssetUID:      assetUID,
		Action:        "lifecycle.setState",
		Outcome:       "pending",
		Reason:        reason,
		NewValue: JSONAny{
			"policyId":      evalResult.PolicyID,
			"requiredCount": evalResult.RequiredCount,
			"state":         params["state"],
		},
	})

	return &ActionResult{
		Action:  "lifecycle.setState",
		Status:  "pending-approval",
		Message: fmt.Sprintf("transition of %s requires approval (policy: %s)", name, evalResult.PolicyName),
		Data: map[string]any{
			"requestId":     reqID,
			"policyId":      evalResult.PolicyID,
			"requiredCount": evalResult.RequiredCount,
		},
	}, nil
}

// executeTransition performs the actual lifecycle state transition.
// Called directly when no approval gate applies, or by the approval handler
// after the approval threshold is met.
func (h *LifecycleActionHandler) executeTransition(plugin, kind, name, actor string, params map[string]any, reason string) (*ActionResult, error) {
	stateStr, _ := params["state"].(string)
	toState := LifecycleState(stateStr)

	uid := fmt.Sprintf("%s:%s:%s", plugin, kind, name)
	record, err := h.store.EnsureExists(plugin, kind, name, uid, actor)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance record: %w", err)
	}

	oldState := record.LifecycleState
	now := time.Now()
	record.LifecycleState = string(toState)
	record.LifecycleReason = reason
	record.LifecycleChangedBy = actor
	record.LifecycleChangedAt = &now

	if err := h.store.Upsert(record); err != nil {
		return nil, fmt.Errorf("failed to update lifecycle: %w", err)
	}

	_ = h.auditStore.Append(&AuditEventRecord{
		ID:            uuid.New().String(),
		CorrelationID: uuid.New().String(),
		EventType:     "governance.lifecycle.changed",
		Actor:         actor,
		AssetUID:      record.AssetUID,
		Action:        "lifecycle.setState",
		Outcome:       "success",
		Reason:        reason,
		OldValue:      JSONAny{"lifecycleState": oldState},
		NewValue:      JSONAny{"lifecycleState": string(toState)},
	})

	return &ActionResult{
		Action:  "lifecycle.setState",
		Status:  "completed",
		Message: fmt.Sprintf("transitioned %s from %s to %s", name, LifecycleState(oldState), toState),
		Data: map[string]any{
			"from":   oldState,
			"to":     string(toState),
			"reason": reason,
		},
	}, nil
}

// handleDeprecate is a convenience for lifecycle.setState with state=deprecated.
func (h *LifecycleActionHandler) handleDeprecate(ctx context.Context, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	if params == nil {
		params = make(map[string]any)
	}
	params["state"] = "deprecated"
	result, err := h.handleSetState(ctx, plugin, kind, name, actor, params, dryRun)
	if result != nil {
		result.Action = "lifecycle.deprecate"
	}
	return result, err
}

// handleArchive is a convenience for lifecycle.setState with state=archived.
func (h *LifecycleActionHandler) handleArchive(ctx context.Context, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	if params == nil {
		params = make(map[string]any)
	}
	params["state"] = "archived"
	result, err := h.handleSetState(ctx, plugin, kind, name, actor, params, dryRun)
	if result != nil {
		result.Action = "lifecycle.archive"
	}
	return result, err
}

// handleRestore transitions archived assets back to deprecated or draft.
func (h *LifecycleActionHandler) handleRestore(ctx context.Context, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	if params == nil {
		params = make(map[string]any)
	}
	targetState, _ := params["targetState"].(string)
	if targetState == "" {
		targetState = "deprecated" // default restore target
	}
	params["state"] = targetState
	result, err := h.handleSetState(ctx, plugin, kind, name, actor, params, dryRun)
	if result != nil {
		result.Action = "lifecycle.restore"
	}
	return result, err
}

// LifecycleActionDefinitions returns action definitions for lifecycle actions.
func LifecycleActionDefinitions() []ActionDefinition {
	return []ActionDefinition{
		{
			ID:          "lifecycle.setState",
			DisplayName: "Set Lifecycle State",
			Description: "Transition asset to a new lifecycle state",
			Scope:       "governance",
		},
		{
			ID:          "lifecycle.deprecate",
			DisplayName: "Deprecate",
			Description: "Mark asset as deprecated",
			Scope:       "governance",
		},
		{
			ID:          "lifecycle.archive",
			DisplayName: "Archive",
			Description: "Archive asset, hiding from default views",
			Scope:       "governance",
			Destructive: true,
		},
		{
			ID:          "lifecycle.restore",
			DisplayName: "Restore",
			Description: "Restore archived asset to deprecated or draft",
			Scope:       "governance",
		},
	}
}
