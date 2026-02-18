package governance

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AuditStore provides append-only operations for audit event records.
type AuditStore struct {
	db *gorm.DB
}

// NewAuditStore creates a new AuditStore.
func NewAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Append creates a new immutable audit event record.
func (s *AuditStore) Append(event *AuditEventRecord) error {
	if err := s.db.Create(event).Error; err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

// ListByAsset returns paginated audit events for a specific asset,
// ordered by created_at DESC (newest first).
// pageToken is an RFC3339 timestamp; events with created_at < pageToken are returned.
func (s *AuditStore) ListByAsset(assetUID string, pageSize int, pageToken string) ([]AuditEventRecord, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Count total events for this asset.
	var totalSize int64
	if err := s.db.Model(&AuditEventRecord{}).Where("asset_uid = ?", assetUID).Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count audit events: %w", err)
	}

	query := s.db.Where("asset_uid = ?", assetUID).Order("created_at DESC").Limit(pageSize + 1)
	if pageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page token: %w", err)
		}
		query = query.Where("created_at < ?", t)
	}

	var records []AuditEventRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, "", 0, fmt.Errorf("list audit events by asset: %w", err)
	}

	var nextToken string
	if len(records) > pageSize {
		nextToken = records[pageSize-1].CreatedAt.Format(time.RFC3339Nano)
		records = records[:pageSize]
	}

	return records, nextToken, int(totalSize), nil
}

// DeleteOlderThan deletes audit events created before the given cutoff time.
// Returns the number of deleted records.
func (s *AuditStore) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result := s.db.Where("created_at < ?", cutoff).Delete(&AuditEventRecord{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete old audit events: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// ListAll returns paginated audit events across all assets,
// ordered by created_at DESC. Optionally filters by event type.
func (s *AuditStore) ListAll(pageSize int, pageToken string, filterEventType string) ([]AuditEventRecord, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	baseQuery := s.db.Model(&AuditEventRecord{})
	if filterEventType != "" {
		baseQuery = baseQuery.Where("event_type = ?", filterEventType)
	}

	var totalSize int64
	if err := baseQuery.Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count audit events: %w", err)
	}

	query := s.db.Order("created_at DESC").Limit(pageSize + 1)
	if filterEventType != "" {
		query = query.Where("event_type = ?", filterEventType)
	}
	if pageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page token: %w", err)
		}
		query = query.Where("created_at < ?", t)
	}

	var records []AuditEventRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, "", 0, fmt.Errorf("list all audit events: %w", err)
	}

	var nextToken string
	if len(records) > pageSize {
		nextToken = records[pageSize-1].CreatedAt.Format(time.RFC3339Nano)
		records = records[:pageSize]
	}

	return records, nextToken, int(totalSize), nil
}
