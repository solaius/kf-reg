package governance

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BindingStore provides CRUD operations for environment binding records.
type BindingStore struct {
	db *gorm.DB
}

// NewBindingStore creates a new BindingStore.
func NewBindingStore(db *gorm.DB) *BindingStore {
	return &BindingStore{db: db}
}

// AutoMigrate creates or updates the env_bindings table.
func (s *BindingStore) AutoMigrate() error {
	if err := s.db.AutoMigrate(&EnvBindingRecord{}); err != nil {
		return fmt.Errorf("auto-migrate env_bindings: %w", err)
	}
	return nil
}

// GetBinding retrieves the binding for a specific asset in a specific environment within a namespace.
// Returns nil, nil if no binding exists.
func (s *BindingStore) GetBinding(namespace, plugin, kind, name, environment string) (*EnvBindingRecord, error) {
	if namespace == "" {
		namespace = "default"
	}
	var record EnvBindingRecord
	err := s.db.Where(
		"namespace = ? AND plugin = ? AND asset_kind = ? AND asset_name = ? AND environment = ?",
		namespace, plugin, kind, name, environment,
	).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get binding: %w", err)
	}
	return &record, nil
}

// SetBinding creates or updates a binding using an upsert on the unique index.
func (s *BindingStore) SetBinding(record *EnvBindingRecord) error {
	if record.Namespace == "" {
		record.Namespace = "default"
	}
	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "namespace"},
			{Name: "plugin"},
			{Name: "asset_kind"},
			{Name: "asset_name"},
			{Name: "environment"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"asset_uid", "version_id", "bound_at", "bound_by", "previous_version_id",
		}),
	}).Create(record).Error
}

// ListBindings returns all environment bindings for a specific asset within a namespace.
func (s *BindingStore) ListBindings(namespace, plugin, kind, name string) ([]EnvBindingRecord, error) {
	if namespace == "" {
		namespace = "default"
	}
	var records []EnvBindingRecord
	err := s.db.Where(
		"namespace = ? AND plugin = ? AND asset_kind = ? AND asset_name = ?",
		namespace, plugin, kind, name,
	).Order("environment ASC").Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("list bindings: %w", err)
	}
	return records, nil
}
