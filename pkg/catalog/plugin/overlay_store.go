package plugin

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OverlayRecord stores user-applied metadata overlays for entities.
// Tags, annotations, labels, and lifecycle changes are stored here
// rather than mutating the source data.
type OverlayRecord struct {
	Namespace  string      `gorm:"primaryKey;column:namespace;default:default"`
	PluginName string      `gorm:"primaryKey;column:plugin_name"`
	EntityKind string      `gorm:"primaryKey;column:entity_kind"`
	EntityUID  string      `gorm:"primaryKey;column:entity_uid"`
	Tags       StringSlice `gorm:"column:tags;type:text"`
	Annotations JSONMap    `gorm:"column:annotations;type:text"`
	Labels     JSONMap     `gorm:"column:labels;type:text"`
	Lifecycle  string      `gorm:"column:lifecycle_phase"`
	UpdatedAt  time.Time   `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the GORM table name.
func (OverlayRecord) TableName() string {
	return "catalog_overlays"
}

// StringSlice is a custom GORM type for []string stored as JSON.
type StringSlice []string

// Scan implements the sql.Scanner interface for StringSlice.
func (s *StringSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for StringSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for StringSlice.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// JSONMap is a custom GORM type for map[string]string stored as JSON.
type JSONMap map[string]string

// Scan implements the sql.Scanner interface for JSONMap.
func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONMap: %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Value implements the driver.Valuer interface for JSONMap.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// OverlayStore provides CRUD operations for overlay records.
type OverlayStore struct {
	db *gorm.DB
}

// NewOverlayStore creates a new OverlayStore.
func NewOverlayStore(db *gorm.DB) *OverlayStore {
	return &OverlayStore{db: db}
}

// AutoMigrate creates or updates the overlay table.
func (s *OverlayStore) AutoMigrate() error {
	return s.db.AutoMigrate(&OverlayRecord{})
}

// Get retrieves the overlay for an entity within a namespace.
// Returns nil, nil if no overlay exists (meaning no modifications).
func (s *OverlayStore) Get(namespace, pluginName, entityKind, entityUID string) (*OverlayRecord, error) {
	if namespace == "" {
		namespace = "default"
	}
	var record OverlayRecord
	err := s.db.Where(
		"namespace = ? AND plugin_name = ? AND entity_kind = ? AND entity_uid = ?",
		namespace, pluginName, entityKind, entityUID,
	).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// Upsert creates or updates an overlay record.
func (s *OverlayStore) Upsert(record *OverlayRecord) error {
	return s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(record).Error
}

// Delete removes an overlay by its composite primary key within a namespace.
func (s *OverlayStore) Delete(namespace, pluginName, entityKind, entityUID string) error {
	if namespace == "" {
		namespace = "default"
	}
	return s.db.Where(
		"namespace = ? AND plugin_name = ? AND entity_kind = ? AND entity_uid = ?",
		namespace, pluginName, entityKind, entityUID,
	).Delete(&OverlayRecord{}).Error
}

// ListByPlugin returns all overlays for a plugin within a namespace.
func (s *OverlayStore) ListByPlugin(namespace, pluginName string) ([]OverlayRecord, error) {
	if namespace == "" {
		namespace = "default"
	}
	var records []OverlayRecord
	if err := s.db.Where("namespace = ? AND plugin_name = ?", namespace, pluginName).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
