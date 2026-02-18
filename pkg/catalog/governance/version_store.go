package governance

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// VersionStore provides CRUD operations for asset version records.
type VersionStore struct {
	db *gorm.DB
}

// NewVersionStore creates a new VersionStore.
func NewVersionStore(db *gorm.DB) *VersionStore {
	return &VersionStore{db: db}
}

// AutoMigrate creates or updates the asset_versions table.
func (s *VersionStore) AutoMigrate() error {
	if err := s.db.AutoMigrate(&AssetVersionRecord{}); err != nil {
		return fmt.Errorf("auto-migrate asset_versions: %w", err)
	}
	return nil
}

// CreateVersion inserts a new immutable version record.
func (s *VersionStore) CreateVersion(record *AssetVersionRecord) error {
	if record.Namespace == "" {
		record.Namespace = "default"
	}
	if err := s.db.Create(record).Error; err != nil {
		return fmt.Errorf("create version: %w", err)
	}
	return nil
}

// GetVersion retrieves a version record by its version ID.
// Returns nil, nil if no record exists.
func (s *VersionStore) GetVersion(versionID string) (*AssetVersionRecord, error) {
	var record AssetVersionRecord
	err := s.db.Where("version_id = ?", versionID).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get version: %w", err)
	}
	return &record, nil
}

// ListVersions returns paginated version records for an asset,
// ordered by created_at DESC (newest first).
// pageToken is an RFC3339Nano timestamp; versions with created_at < pageToken are returned.
func (s *VersionStore) ListVersions(assetUID string, pageSize int, pageToken string) ([]AssetVersionRecord, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Count total versions for this asset.
	var totalSize int64
	if err := s.db.Model(&AssetVersionRecord{}).Where("asset_uid = ?", assetUID).Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count versions: %w", err)
	}

	query := s.db.Where("asset_uid = ?", assetUID).Order("created_at DESC").Limit(pageSize + 1)
	if pageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page token: %w", err)
		}
		query = query.Where("created_at < ?", t)
	}

	var records []AssetVersionRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, "", 0, fmt.Errorf("list versions: %w", err)
	}

	var nextToken string
	if len(records) > pageSize {
		nextToken = records[pageSize-1].CreatedAt.Format(time.RFC3339Nano)
		records = records[:pageSize]
	}

	return records, nextToken, int(totalSize), nil
}
