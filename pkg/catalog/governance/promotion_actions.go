package governance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PromotionActionHandler handles version and promotion governance actions.
type PromotionActionHandler struct {
	govStore             *GovernanceStore
	versionStore         *VersionStore
	bindingStore         *BindingStore
	auditStore           *AuditStore
	provenanceExtractor  ProvenanceExtractor
}

// NewPromotionActionHandler creates a promotion action handler.
func NewPromotionActionHandler(govStore *GovernanceStore, versionStore *VersionStore, bindingStore *BindingStore, auditStore *AuditStore) *PromotionActionHandler {
	return &PromotionActionHandler{
		govStore:     govStore,
		versionStore: versionStore,
		bindingStore: bindingStore,
		auditStore:   auditStore,
	}
}

// SetProvenanceExtractor sets an optional provenance extractor that will be
// called during version creation to populate provenance fields.
func (h *PromotionActionHandler) SetProvenanceExtractor(e ProvenanceExtractor) {
	h.provenanceExtractor = e
}

// HandleAction dispatches promotion actions.
// Supported: version.create, promotion.bind, promotion.promote, promotion.rollback
func (h *PromotionActionHandler) HandleAction(ctx context.Context, namespace, plugin, kind, name, actor, action string, params map[string]any, dryRun bool) (*ActionResult, error) {
	switch action {
	case "version.create":
		return h.handleVersionCreate(ctx, namespace, plugin, kind, name, actor, params, dryRun)
	case "promotion.bind":
		return h.handleBind(ctx, namespace, plugin, kind, name, actor, params, dryRun)
	case "promotion.promote":
		return h.handlePromote(ctx, namespace, plugin, kind, name, actor, params, dryRun)
	case "promotion.rollback":
		return h.handleRollback(ctx, namespace, plugin, kind, name, actor, params, dryRun)
	default:
		return nil, fmt.Errorf("unknown promotion action: %s", action)
	}
}

// handleVersionCreate creates an immutable version snapshot.
// params: { "versionLabel": "v1.0" }
func (h *PromotionActionHandler) handleVersionCreate(ctx context.Context, namespace, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	versionLabel, _ := params["versionLabel"].(string)
	if versionLabel == "" {
		return nil, fmt.Errorf("missing or invalid 'versionLabel' parameter")
	}
	reason, _ := params["reason"].(string)

	uid := fmt.Sprintf("%s:%s:%s", plugin, kind, name)
	govRecord, err := h.govStore.EnsureExists(namespace, plugin, kind, name, uid, actor)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance record: %w", err)
	}

	if dryRun {
		return &ActionResult{
			Action:  "version.create",
			Status:  "dry-run",
			Message: fmt.Sprintf("would create version %s for %s", versionLabel, name),
			Data: map[string]any{
				"versionLabel":   versionLabel,
				"lifecycleState": govRecord.LifecycleState,
			},
		}, nil
	}

	govSnapshot := overlayToMap(recordToOverlay(govRecord))
	versionRecord := &AssetVersionRecord{
		ID:                 uuid.New().String(),
		Namespace:          namespace,
		AssetUID:           govRecord.AssetUID,
		VersionID:          fmt.Sprintf("%s:%s", versionLabel, uuid.New().String()[:8]),
		VersionLabel:       versionLabel,
		CreatedBy:          actor,
		GovernanceSnapshot: govSnapshot,
		AssetSnapshot:      JSONAny{},
	}

	// Populate provenance fields if an extractor is configured.
	if h.provenanceExtractor != nil {
		applyProvenance(versionRecord, h.provenanceExtractor.ExtractProvenance(plugin, kind, name))
	}

	if err := h.versionStore.CreateVersion(versionRecord); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	_ = h.auditStore.Append(&AuditEventRecord{
		ID:            uuid.New().String(),
		Namespace:     namespace,
		CorrelationID: uuid.New().String(),
		EventType:     "governance.version.created",
		Actor:         actor,
		AssetUID:      govRecord.AssetUID,
		VersionID:     versionRecord.VersionID,
		Action:        "version.create",
		Outcome:       "success",
		Reason:        reason,
		NewValue: JSONAny{
			"versionId":    versionRecord.VersionID,
			"versionLabel": versionRecord.VersionLabel,
		},
	})

	return &ActionResult{
		Action:  "version.create",
		Status:  "completed",
		Message: fmt.Sprintf("created version %s for %s", versionLabel, name),
		Data: map[string]any{
			"versionId":    versionRecord.VersionID,
			"versionLabel": versionRecord.VersionLabel,
		},
	}, nil
}

// handleBind sets a version binding for an environment.
// params: { "environment": "dev", "versionId": "v1.0:abc12345" }
func (h *PromotionActionHandler) handleBind(ctx context.Context, namespace, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	environment, _ := params["environment"].(string)
	if environment == "" {
		return nil, fmt.Errorf("missing or invalid 'environment' parameter")
	}
	versionID, _ := params["versionId"].(string)
	if versionID == "" {
		return nil, fmt.Errorf("missing or invalid 'versionId' parameter")
	}

	govRecord, err := h.govStore.Get(namespace, plugin, kind, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance record: %w", err)
	}
	if govRecord == nil {
		return nil, fmt.Errorf("governance record not found for %s/%s/%s", plugin, kind, name)
	}

	// Check lifecycle state constraints.
	state := LifecycleState(govRecord.LifecycleState)
	switch state {
	case StateArchived:
		return nil, fmt.Errorf("archived assets cannot be bound")
	case StateDraft:
		if environment == "stage" || environment == "prod" {
			return nil, fmt.Errorf("draft assets cannot be bound to stage/prod")
		}
	}

	// Verify version exists.
	version, err := h.versionStore.GetVersion(versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify version: %w", err)
	}
	if version == nil {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	if dryRun {
		msg := fmt.Sprintf("would bind %s to version %s in %s", name, versionID, environment)
		data := map[string]any{
			"environment": environment,
			"versionId":   versionID,
		}
		if state == StateDeprecated {
			data["warning"] = "asset is deprecated; binding is allowed but consider migrating"
		}
		return &ActionResult{
			Action:  "promotion.bind",
			Status:  "dry-run",
			Message: msg,
			Data:    data,
		}, nil
	}

	// Get current binding for this env (if any) to record previous_version_id.
	var previousVersionID string
	existing, err := h.bindingStore.GetBinding(namespace, plugin, kind, name, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing binding: %w", err)
	}
	if existing != nil {
		previousVersionID = existing.VersionID
	}

	now := time.Now()
	bindingRecord := &EnvBindingRecord{
		ID:                uuid.New().String(),
		Namespace:         namespace,
		Plugin:            plugin,
		AssetKind:         kind,
		AssetName:         name,
		Environment:       environment,
		AssetUID:          govRecord.AssetUID,
		VersionID:         versionID,
		BoundAt:           now,
		BoundBy:           actor,
		PreviousVersionID: previousVersionID,
	}

	if err := h.bindingStore.SetBinding(bindingRecord); err != nil {
		return nil, fmt.Errorf("failed to set binding: %w", err)
	}

	_ = h.auditStore.Append(&AuditEventRecord{
		ID:            uuid.New().String(),
		Namespace:     namespace,
		CorrelationID: uuid.New().String(),
		EventType:     "governance.promotion.bound",
		Actor:         actor,
		AssetUID:      govRecord.AssetUID,
		VersionID:     versionID,
		Action:        "promotion.bind",
		Outcome:       "success",
		OldValue:      JSONAny{"versionId": previousVersionID, "environment": environment},
		NewValue:      JSONAny{"versionId": versionID, "environment": environment},
	})

	data := map[string]any{
		"environment":       environment,
		"versionId":         versionID,
		"previousVersionId": previousVersionID,
	}
	if state == StateDeprecated {
		data["warning"] = "asset is deprecated; binding is allowed but consider migrating"
	}

	return &ActionResult{
		Action:  "promotion.bind",
		Status:  "completed",
		Message: fmt.Sprintf("bound %s to version %s in %s", name, versionID, environment),
		Data:    data,
	}, nil
}

// handlePromote copies a binding from one environment to another.
// params: { "fromEnv": "dev", "toEnv": "stage" }
func (h *PromotionActionHandler) handlePromote(ctx context.Context, namespace, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	fromEnv, _ := params["fromEnv"].(string)
	if fromEnv == "" {
		return nil, fmt.Errorf("missing or invalid 'fromEnv' parameter")
	}
	toEnv, _ := params["toEnv"].(string)
	if toEnv == "" {
		return nil, fmt.Errorf("missing or invalid 'toEnv' parameter")
	}
	if fromEnv == toEnv {
		return nil, fmt.Errorf("fromEnv and toEnv must be different")
	}

	// Get the source binding.
	sourceBinding, err := h.bindingStore.GetBinding(namespace, plugin, kind, name, fromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to get source binding: %w", err)
	}
	if sourceBinding == nil {
		return nil, fmt.Errorf("no binding found in %s for %s/%s/%s", fromEnv, plugin, kind, name)
	}

	// Check lifecycle state constraints for the target environment.
	govRecord, err := h.govStore.Get(namespace, plugin, kind, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance record: %w", err)
	}
	if govRecord == nil {
		return nil, fmt.Errorf("governance record not found for %s/%s/%s", plugin, kind, name)
	}

	state := LifecycleState(govRecord.LifecycleState)
	switch state {
	case StateArchived:
		return nil, fmt.Errorf("archived assets cannot be promoted")
	case StateDraft:
		if toEnv == "stage" || toEnv == "prod" {
			return nil, fmt.Errorf("draft assets cannot be promoted to stage/prod")
		}
	}

	if dryRun {
		return &ActionResult{
			Action:  "promotion.promote",
			Status:  "dry-run",
			Message: fmt.Sprintf("would promote %s from %s to %s (version %s)", name, fromEnv, toEnv, sourceBinding.VersionID),
			Data: map[string]any{
				"fromEnv":   fromEnv,
				"toEnv":     toEnv,
				"versionId": sourceBinding.VersionID,
			},
		}, nil
	}

	// Get existing target binding for previous_version_id tracking.
	var previousVersionID string
	existingTarget, err := h.bindingStore.GetBinding(namespace, plugin, kind, name, toEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing target binding: %w", err)
	}
	if existingTarget != nil {
		previousVersionID = existingTarget.VersionID
	}

	now := time.Now()
	targetBinding := &EnvBindingRecord{
		ID:                uuid.New().String(),
		Namespace:         namespace,
		Plugin:            plugin,
		AssetKind:         kind,
		AssetName:         name,
		Environment:       toEnv,
		AssetUID:          govRecord.AssetUID,
		VersionID:         sourceBinding.VersionID,
		BoundAt:           now,
		BoundBy:           actor,
		PreviousVersionID: previousVersionID,
	}

	if err := h.bindingStore.SetBinding(targetBinding); err != nil {
		return nil, fmt.Errorf("failed to set target binding: %w", err)
	}

	_ = h.auditStore.Append(&AuditEventRecord{
		ID:            uuid.New().String(),
		Namespace:     namespace,
		CorrelationID: uuid.New().String(),
		EventType:     "governance.promotion.promoted",
		Actor:         actor,
		AssetUID:      govRecord.AssetUID,
		VersionID:     sourceBinding.VersionID,
		Action:        "promotion.promote",
		Outcome:       "success",
		OldValue:      JSONAny{"versionId": previousVersionID, "environment": toEnv},
		NewValue:      JSONAny{"versionId": sourceBinding.VersionID, "fromEnv": fromEnv, "toEnv": toEnv},
	})

	return &ActionResult{
		Action:  "promotion.promote",
		Status:  "completed",
		Message: fmt.Sprintf("promoted %s from %s to %s (version %s)", name, fromEnv, toEnv, sourceBinding.VersionID),
		Data: map[string]any{
			"fromEnv":           fromEnv,
			"toEnv":             toEnv,
			"versionId":         sourceBinding.VersionID,
			"previousVersionId": previousVersionID,
		},
	}, nil
}

// handleRollback updates a binding to a specific (previous) version.
// params: { "environment": "prod", "targetVersionId": "v1.0:abc12345" }
func (h *PromotionActionHandler) handleRollback(ctx context.Context, namespace, plugin, kind, name, actor string, params map[string]any, dryRun bool) (*ActionResult, error) {
	environment, _ := params["environment"].(string)
	if environment == "" {
		return nil, fmt.Errorf("missing or invalid 'environment' parameter")
	}
	targetVersionID, _ := params["targetVersionId"].(string)
	if targetVersionID == "" {
		return nil, fmt.Errorf("missing or invalid 'targetVersionId' parameter")
	}

	// Verify the target version exists.
	version, err := h.versionStore.GetVersion(targetVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify target version: %w", err)
	}
	if version == nil {
		return nil, fmt.Errorf("version %s not found", targetVersionID)
	}

	govRecord, err := h.govStore.Get(namespace, plugin, kind, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance record: %w", err)
	}
	if govRecord == nil {
		return nil, fmt.Errorf("governance record not found for %s/%s/%s", plugin, kind, name)
	}

	// Get current binding.
	currentBinding, err := h.bindingStore.GetBinding(namespace, plugin, kind, name, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to get current binding: %w", err)
	}

	var currentVersionID string
	if currentBinding != nil {
		currentVersionID = currentBinding.VersionID
	}

	if dryRun {
		return &ActionResult{
			Action:  "promotion.rollback",
			Status:  "dry-run",
			Message: fmt.Sprintf("would rollback %s in %s from %s to %s", name, environment, currentVersionID, targetVersionID),
			Data: map[string]any{
				"environment":      environment,
				"currentVersionId": currentVersionID,
				"targetVersionId":  targetVersionID,
			},
		}, nil
	}

	now := time.Now()
	bindingRecord := &EnvBindingRecord{
		ID:                uuid.New().String(),
		Namespace:         namespace,
		Plugin:            plugin,
		AssetKind:         kind,
		AssetName:         name,
		Environment:       environment,
		AssetUID:          govRecord.AssetUID,
		VersionID:         targetVersionID,
		BoundAt:           now,
		BoundBy:           actor,
		PreviousVersionID: currentVersionID,
	}

	if err := h.bindingStore.SetBinding(bindingRecord); err != nil {
		return nil, fmt.Errorf("failed to set rollback binding: %w", err)
	}

	_ = h.auditStore.Append(&AuditEventRecord{
		ID:            uuid.New().String(),
		Namespace:     namespace,
		CorrelationID: uuid.New().String(),
		EventType:     "governance.promotion.rollback",
		Actor:         actor,
		AssetUID:      govRecord.AssetUID,
		VersionID:     targetVersionID,
		Action:        "promotion.rollback",
		Outcome:       "success",
		OldValue:      JSONAny{"versionId": currentVersionID, "environment": environment},
		NewValue:      JSONAny{"versionId": targetVersionID, "environment": environment},
	})

	return &ActionResult{
		Action:  "promotion.rollback",
		Status:  "completed",
		Message: fmt.Sprintf("rolled back %s in %s to version %s", name, environment, targetVersionID),
		Data: map[string]any{
			"environment":       environment,
			"versionId":         targetVersionID,
			"previousVersionId": currentVersionID,
		},
	}, nil
}

// PromotionActionDefinitions returns action definitions for version and promotion actions.
func PromotionActionDefinitions() []ActionDefinition {
	return []ActionDefinition{
		{
			ID:          "version.create",
			DisplayName: "Create Version",
			Description: "Create an immutable version snapshot of the asset",
			Scope:       "governance",
		},
		{
			ID:          "promotion.bind",
			DisplayName: "Bind to Environment",
			Description: "Bind a specific version to an environment",
			Scope:       "governance",
		},
		{
			ID:          "promotion.promote",
			DisplayName: "Promote",
			Description: "Promote a version from one environment to another",
			Scope:       "governance",
		},
		{
			ID:          "promotion.rollback",
			DisplayName: "Rollback",
			Description: "Roll back an environment to a previous version",
			Scope:       "governance",
			Destructive: true,
		},
	}
}
