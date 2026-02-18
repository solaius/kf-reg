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
	if event.Namespace == "" {
		event.Namespace = "default"
	}
	if err := s.db.Create(event).Error; err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

// ListByAsset returns paginated audit events for a specific asset within a namespace,
// ordered by created_at DESC (newest first).
// pageToken is an RFC3339 timestamp; events with created_at < pageToken are returned.
func (s *AuditStore) ListByAsset(namespace, assetUID string, pageSize int, pageToken string) ([]AuditEventRecord, string, int, error) {
	if namespace == "" {
		namespace = "default"
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Count total events for this asset.
	var totalSize int64
	if err := s.db.Model(&AuditEventRecord{}).Where("namespace = ? AND asset_uid = ?", namespace, assetUID).Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count audit events: %w", err)
	}

	query := s.db.Where("namespace = ? AND asset_uid = ?", namespace, assetUID).Order("created_at DESC").Limit(pageSize + 1)
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

// AuditListFilter defines filters for listing audit events.
type AuditListFilter struct {
	Namespace string
	Actor     string
	Plugin    string
	Action    string
	EventType string
}

// GetByID retrieves a single audit event by its ID.
// Returns nil if not found.
func (s *AuditStore) GetByID(id string) (*AuditEventRecord, error) {
	var record AuditEventRecord
	result := s.db.Where("id = ?", id).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get audit event by ID: %w", result.Error)
	}
	return &record, nil
}

// ListFiltered returns paginated audit events matching the given filter,
// ordered by created_at DESC.
func (s *AuditStore) ListFiltered(filter AuditListFilter, pageSize int, pageToken string) ([]AuditEventRecord, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	buildQuery := func(base *gorm.DB) *gorm.DB {
		q := base.Model(&AuditEventRecord{})
		if filter.Namespace != "" {
			q = q.Where("namespace = ?", filter.Namespace)
		}
		if filter.Actor != "" {
			q = q.Where("actor = ?", filter.Actor)
		}
		if filter.Plugin != "" {
			q = q.Where("plugin = ?", filter.Plugin)
		}
		if filter.Action != "" {
			q = q.Where("action_verb = ?", filter.Action)
		}
		if filter.EventType != "" {
			q = q.Where("event_type = ?", filter.EventType)
		}
		return q
	}

	// Count total matching events.
	var totalSize int64
	if err := buildQuery(s.db).Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count filtered audit events: %w", err)
	}

	query := buildQuery(s.db).Order("created_at DESC").Limit(pageSize + 1)
	if pageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page token: %w", err)
		}
		query = query.Where("created_at < ?", t)
	}

	var records []AuditEventRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, "", 0, fmt.Errorf("list filtered audit events: %w", err)
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

// ListAll returns paginated audit events within a namespace,
// ordered by created_at DESC. Optionally filters by event type.
func (s *AuditStore) ListAll(namespace string, pageSize int, pageToken string, filterEventType string) ([]AuditEventRecord, string, int, error) {
	if namespace == "" {
		namespace = "default"
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	baseQuery := s.db.Model(&AuditEventRecord{}).Where("namespace = ?", namespace)
	if filterEventType != "" {
		baseQuery = baseQuery.Where("event_type = ?", filterEventType)
	}

	var totalSize int64
	if err := baseQuery.Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count audit events: %w", err)
	}

	query := s.db.Where("namespace = ?", namespace).Order("created_at DESC").Limit(pageSize + 1)
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
