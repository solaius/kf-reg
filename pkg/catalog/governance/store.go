package governance

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GovernanceStore provides CRUD operations for asset governance records.
type GovernanceStore struct {
	db *gorm.DB
}

// NewGovernanceStore creates a new GovernanceStore.
func NewGovernanceStore(db *gorm.DB) *GovernanceStore {
	return &GovernanceStore{db: db}
}

// AutoMigrate creates or updates the governance tables, including approval tables.
func (s *GovernanceStore) AutoMigrate() error {
	if err := s.db.AutoMigrate(&AssetGovernanceRecord{}); err != nil {
		return fmt.Errorf("auto-migrate asset_governance: %w", err)
	}
	if err := s.db.AutoMigrate(&AuditEventRecord{}); err != nil {
		return fmt.Errorf("auto-migrate audit_events: %w", err)
	}
	if err := s.db.AutoMigrate(&ApprovalRequestRecord{}); err != nil {
		return fmt.Errorf("auto-migrate approval_requests: %w", err)
	}
	if err := s.db.AutoMigrate(&ApprovalDecisionRecord{}); err != nil {
		return fmt.Errorf("auto-migrate approval_decisions: %w", err)
	}
	return nil
}

// Get retrieves the governance record for an asset by namespace, plugin, kind, and name.
// Returns nil, nil if no record exists.
func (s *GovernanceStore) Get(namespace, plugin, kind, name string) (*AssetGovernanceRecord, error) {
	if namespace == "" {
		namespace = "default"
	}
	var record AssetGovernanceRecord
	err := s.db.Where(
		"namespace = ? AND plugin = ? AND asset_kind = ? AND asset_name = ?",
		namespace, plugin, kind, name,
	).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get governance record: %w", err)
	}
	return &record, nil
}

// GetByUID retrieves the governance record for an asset by its unique ID.
// Returns nil, nil if no record exists.
func (s *GovernanceStore) GetByUID(assetUID string) (*AssetGovernanceRecord, error) {
	var record AssetGovernanceRecord
	err := s.db.Where("asset_uid = ?", assetUID).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get governance record by uid: %w", err)
	}
	return &record, nil
}

// Upsert creates or updates a governance record.
// The conflict is resolved on the asset_uid unique index.
func (s *GovernanceStore) Upsert(record *AssetGovernanceRecord) error {
	if record.Namespace == "" {
		record.Namespace = "default"
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "namespace"}, {Name: "asset_uid"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"plugin", "asset_kind", "asset_name",
			"owner_principal", "owner_display_name", "owner_email",
			"team_name", "team_id",
			"sla_tier", "sla_response_hours",
			"risk_level", "risk_categories",
			"intended_use_summary", "intended_use_environments", "intended_use_restrictions",
			"compliance_tags", "compliance_controls",
			"lifecycle_state", "lifecycle_reason", "lifecycle_changed_by", "lifecycle_changed_at",
			"audit_last_reviewed_at", "audit_review_cadence_days",
			"updated_at",
		}),
	}).Create(record).Error
}

// Delete removes a governance record by namespace, plugin, kind, and name.
func (s *GovernanceStore) Delete(namespace, plugin, kind, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	return s.db.Where(
		"namespace = ? AND plugin = ? AND asset_kind = ? AND asset_name = ?",
		namespace, plugin, kind, name,
	).Delete(&AssetGovernanceRecord{}).Error
}

// List returns paginated governance records for a plugin within a namespace.
// pageToken is the ID of the last record from the previous page; pass "" for the first page.
func (s *GovernanceStore) List(namespace, plugin string, pageSize int, pageToken string) ([]AssetGovernanceRecord, string, error) {
	if namespace == "" {
		namespace = "default"
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.Where("namespace = ? AND plugin = ?", namespace, plugin).Order("id ASC").Limit(pageSize + 1)
	if pageToken != "" {
		query = query.Where("id > ?", pageToken)
	}

	var records []AssetGovernanceRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, "", fmt.Errorf("list governance records: %w", err)
	}

	var nextToken string
	if len(records) > pageSize {
		nextToken = records[pageSize-1].ID
		records = records[:pageSize]
	}

	return records, nextToken, nil
}

// EnsureExists returns the existing governance record for an asset, or creates
// a new one with default values if none exists.
func (s *GovernanceStore) EnsureExists(namespace, plugin, kind, name, uid, changedBy string) (*AssetGovernanceRecord, error) {
	if namespace == "" {
		namespace = "default"
	}
	existing, err := s.Get(namespace, plugin, kind, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	if uid == "" {
		uid = fmt.Sprintf("%s:%s:%s", plugin, kind, name)
	}

	record := &AssetGovernanceRecord{
		ID:                 uuid.New().String(),
		Namespace:          namespace,
		Plugin:             plugin,
		AssetKind:          kind,
		AssetName:          name,
		AssetUID:           uid,
		RiskLevel:          string(RiskMedium),
		LifecycleState:     string(StateDraft),
		LifecycleChangedBy: changedBy,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("create default governance record: %w", err)
	}

	return record, nil
}
