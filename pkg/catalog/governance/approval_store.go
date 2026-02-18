package governance

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ApprovalStore provides CRUD operations for approval requests and decisions.
type ApprovalStore struct {
	db *gorm.DB
}

// NewApprovalStore creates a new ApprovalStore.
func NewApprovalStore(db *gorm.DB) *ApprovalStore {
	return &ApprovalStore{db: db}
}

// AutoMigrate creates or updates the approval tables.
func (s *ApprovalStore) AutoMigrate() error {
	if err := s.db.AutoMigrate(&ApprovalRequestRecord{}); err != nil {
		return fmt.Errorf("auto-migrate approval_requests: %w", err)
	}
	if err := s.db.AutoMigrate(&ApprovalDecisionRecord{}); err != nil {
		return fmt.Errorf("auto-migrate approval_decisions: %w", err)
	}
	return nil
}

// Create inserts a new approval request.
// If Namespace is empty, it defaults to "default".
func (s *ApprovalStore) Create(req *ApprovalRequestRecord) error {
	if req.Namespace == "" {
		req.Namespace = "default"
	}
	if err := s.db.Create(req).Error; err != nil {
		return fmt.Errorf("create approval request: %w", err)
	}
	return nil
}

// Get retrieves an approval request by ID, including its decisions.
func (s *ApprovalStore) Get(id string) (*ApprovalRequestRecord, []ApprovalDecisionRecord, error) {
	var req ApprovalRequestRecord
	if err := s.db.Where("id = ?", id).First(&req).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get approval request: %w", err)
	}

	var decisions []ApprovalDecisionRecord
	if err := s.db.Where("request_id = ?", id).Order("created_at ASC").Find(&decisions).Error; err != nil {
		return nil, nil, fmt.Errorf("get approval decisions: %w", err)
	}

	return &req, decisions, nil
}

// List returns paginated approval requests within a namespace,
// optionally filtered by status and/or asset UID.
func (s *ApprovalStore) List(namespace string, status ApprovalStatus, assetUID string, pageSize int, pageToken string) ([]ApprovalRequestRecord, string, int, error) {
	if namespace == "" {
		namespace = "default"
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	baseQuery := s.db.Model(&ApprovalRequestRecord{}).Where("namespace = ?", namespace)
	if status != "" {
		baseQuery = baseQuery.Where("status = ?", string(status))
	}
	if assetUID != "" {
		baseQuery = baseQuery.Where("asset_uid = ?", assetUID)
	}

	var totalSize int64
	if err := baseQuery.Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count approval requests: %w", err)
	}

	query := s.db.Where("namespace = ?", namespace).Order("created_at DESC").Limit(pageSize + 1)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}
	if assetUID != "" {
		query = query.Where("asset_uid = ?", assetUID)
	}
	if pageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page token: %w", err)
		}
		query = query.Where("created_at < ?", t)
	}

	var records []ApprovalRequestRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, "", 0, fmt.Errorf("list approval requests: %w", err)
	}

	var nextToken string
	if len(records) > pageSize {
		nextToken = records[pageSize-1].CreatedAt.Format(time.RFC3339Nano)
		records = records[:pageSize]
	}

	return records, nextToken, int(totalSize), nil
}

// ListPendingForAsset returns all pending approval requests for a specific asset and action.
func (s *ApprovalStore) ListPendingForAsset(assetUID, action string) ([]ApprovalRequestRecord, error) {
	var records []ApprovalRequestRecord
	query := s.db.Where("asset_uid = ? AND action = ? AND status = ?", assetUID, action, string(ApprovalStatusPending))
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list pending approvals for asset: %w", err)
	}
	return records, nil
}

// AddDecision adds a reviewer decision to an existing approval request.
func (s *ApprovalStore) AddDecision(decision *ApprovalDecisionRecord) error {
	if err := s.db.Create(decision).Error; err != nil {
		return fmt.Errorf("add approval decision: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of an approval request and sets resolution fields.
func (s *ApprovalStore) UpdateStatus(id string, status ApprovalStatus, resolvedBy, resolutionNote string) error {
	now := time.Now()
	updates := map[string]any{
		"status":          string(status),
		"resolved_at":     &now,
		"resolved_by":     resolvedBy,
		"resolution_note": resolutionNote,
	}
	if err := s.db.Model(&ApprovalRequestRecord{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("update approval status: %w", err)
	}
	return nil
}

// Cancel cancels a pending approval request.
func (s *ApprovalStore) Cancel(id, actor, reason string) error {
	result := s.db.Model(&ApprovalRequestRecord{}).
		Where("id = ? AND status = ?", id, string(ApprovalStatusPending)).
		Updates(map[string]any{
			"status":          string(ApprovalStatusCanceled),
			"resolved_by":     actor,
			"resolution_note": reason,
			"resolved_at":     time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("cancel approval request: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("approval request %s not found or not in pending status", id)
	}
	return nil
}

// CountDecisions counts approve/deny decisions for a request.
func (s *ApprovalStore) CountDecisions(requestID string) (approves int, denies int, err error) {
	var decisions []ApprovalDecisionRecord
	if err := s.db.Where("request_id = ?", requestID).Find(&decisions).Error; err != nil {
		return 0, 0, fmt.Errorf("count decisions: %w", err)
	}
	for _, d := range decisions {
		switch d.Verdict {
		case VerdictApprove:
			approves++
		case VerdictDeny:
			denies++
		}
	}
	return approves, denies, nil
}
