package plugin

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// RefreshStatusRecord persists refresh metadata so it survives server restarts.
type RefreshStatusRecord struct {
	Namespace          string     `gorm:"primaryKey;column:namespace;default:default"`
	SourceID           string     `gorm:"primaryKey;column:source_id"`
	PluginName         string     `gorm:"column:plugin_name;index"`
	LastRefreshTime    *time.Time `gorm:"column:last_refresh_time"`
	LastRefreshStatus  string     `gorm:"column:last_refresh_status"`  // "success", "error"
	LastRefreshSummary string     `gorm:"column:last_refresh_summary"` // e.g. "Loaded 6 entities"
	LastError          string     `gorm:"column:last_error"`
	EntitiesLoaded     int        `gorm:"column:entities_loaded"`
	EntitiesRemoved    int        `gorm:"column:entities_removed"`
	DurationMs         int64      `gorm:"column:duration_ms"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName overrides the default table name.
func (RefreshStatusRecord) TableName() string {
	return "catalog_refresh_status"
}

// saveRefreshStatus persists refresh results to the database.
// If the DB is nil this is a no-op.
func (s *Server) saveRefreshStatus(namespace, pluginName, sourceID string, result *RefreshResult) {
	if s.db == nil || result == nil {
		return
	}

	if namespace == "" {
		namespace = "default"
	}

	now := time.Now()
	status := "success"
	if result.Error != "" {
		status = "error"
	}

	summary := formatRefreshSummary(result)

	record := RefreshStatusRecord{
		Namespace:          namespace,
		SourceID:           sourceID,
		PluginName:         pluginName,
		LastRefreshTime:    &now,
		LastRefreshStatus:  status,
		LastRefreshSummary: summary,
		LastError:          result.Error,
		EntitiesLoaded:     result.EntitiesLoaded,
		EntitiesRemoved:    result.EntitiesRemoved,
		DurationMs:         result.Duration.Milliseconds(),
	}

	// Upsert: create or update.
	if err := s.db.Save(&record).Error; err != nil {
		s.logger.Error("failed to save refresh status", "namespace", namespace, "plugin", pluginName, "source", sourceID, "error", err)
	}
}

// getRefreshStatus loads a single refresh status record from the database.
func (s *Server) getRefreshStatus(namespace, pluginName, sourceID string) *RefreshStatusRecord {
	if s.db == nil {
		return nil
	}

	if namespace == "" {
		namespace = "default"
	}

	var record RefreshStatusRecord
	err := s.db.Where("namespace = ? AND plugin_name = ? AND source_id = ?", namespace, pluginName, sourceID).First(&record).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			s.logger.Error("failed to load refresh status", "namespace", namespace, "plugin", pluginName, "source", sourceID, "error", err)
		}
		return nil
	}
	return &record
}

// listRefreshStatuses loads all refresh status records for a plugin within a namespace.
func (s *Server) listRefreshStatuses(namespace, pluginName string) []RefreshStatusRecord {
	if s.db == nil {
		return nil
	}

	if namespace == "" {
		namespace = "default"
	}

	var records []RefreshStatusRecord
	if err := s.db.Where("namespace = ? AND plugin_name = ?", namespace, pluginName).Find(&records).Error; err != nil {
		s.logger.Error("failed to list refresh statuses", "namespace", namespace, "plugin", pluginName, "error", err)
		return nil
	}
	return records
}

// deleteRefreshStatus removes the refresh status record for a specific source.
// If the DB is nil this is a no-op.
func (s *Server) deleteRefreshStatus(namespace, pluginName, sourceID string) {
	if s.db == nil {
		return
	}
	if namespace == "" {
		namespace = "default"
	}
	if err := s.db.Where("namespace = ? AND plugin_name = ? AND source_id = ?", namespace, pluginName, sourceID).Delete(&RefreshStatusRecord{}).Error; err != nil {
		s.logger.Error("failed to delete refresh status", "namespace", namespace, "plugin", pluginName, "source", sourceID, "error", err)
	}
}

// deleteAllRefreshStatuses removes all refresh status records for a plugin.
// If the DB is nil this is a no-op. Deletes across all namespaces.
func (s *Server) deleteAllRefreshStatuses(pluginName string) {
	if s.db == nil {
		return
	}
	if err := s.db.Where("plugin_name = ?", pluginName).Delete(&RefreshStatusRecord{}).Error; err != nil {
		s.logger.Error("failed to delete all refresh statuses", "plugin", pluginName, "error", err)
	}
}

// formatRefreshSummary creates a human-readable summary from a RefreshResult.
func formatRefreshSummary(result *RefreshResult) string {
	if result.Error != "" {
		return "Refresh failed"
	}
	if result.EntitiesRemoved > 0 {
		return fmt.Sprintf("Loaded %d entities, removed %d", result.EntitiesLoaded, result.EntitiesRemoved)
	}
	return fmt.Sprintf("Loaded %d entities", result.EntitiesLoaded)
}
