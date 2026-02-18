package governance

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONStringSlice is a custom GORM type for []string stored as JSON.
type JSONStringSlice []string

// Scan implements the sql.Scanner interface for JSONStringSlice.
func (s *JSONStringSlice) Scan(value any) error {
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
		return fmt.Errorf("unsupported type for JSONStringSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for JSONStringSlice.
func (s JSONStringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// JSONAny is a custom GORM type for map[string]any stored as JSON.
type JSONAny map[string]any

// Scan implements the sql.Scanner interface for JSONAny.
func (m *JSONAny) Scan(value any) error {
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
		return fmt.Errorf("unsupported type for JSONAny: %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Value implements the driver.Valuer interface for JSONAny.
func (m JSONAny) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// AssetGovernanceRecord stores governance metadata for an asset.
type AssetGovernanceRecord struct {
	ID                      string          `gorm:"primaryKey;column:id;type:varchar(36)"`
	Namespace               string          `gorm:"column:namespace;index:idx_gov_asset,priority:1;uniqueIndex:idx_gov_ns_uid,priority:1;default:default;not null"`
	Plugin                  string          `gorm:"column:plugin;index:idx_gov_asset,priority:2;not null"`
	AssetKind               string          `gorm:"column:asset_kind;index:idx_gov_asset,priority:3;not null"`
	AssetName               string          `gorm:"column:asset_name;index:idx_gov_asset,priority:4;not null"`
	AssetUID                string          `gorm:"column:asset_uid;uniqueIndex:idx_gov_ns_uid,priority:2;not null"`
	OwnerPrincipal          string          `gorm:"column:owner_principal"`
	OwnerDisplayName        string          `gorm:"column:owner_display_name"`
	OwnerEmail              string          `gorm:"column:owner_email"`
	TeamName                string          `gorm:"column:team_name"`
	TeamID                  string          `gorm:"column:team_id"`
	SLATier                 string          `gorm:"column:sla_tier"`
	SLAResponseHours        int             `gorm:"column:sla_response_hours"`
	RiskLevel               string          `gorm:"column:risk_level;default:medium;not null"`
	RiskCategories          JSONStringSlice `gorm:"column:risk_categories;type:text"`
	IntendedUseSummary      string          `gorm:"column:intended_use_summary"`
	IntendedUseEnvs         JSONStringSlice `gorm:"column:intended_use_environments;type:text"`
	IntendedUseRestrictions JSONStringSlice `gorm:"column:intended_use_restrictions;type:text"`
	ComplianceTags          JSONStringSlice `gorm:"column:compliance_tags;type:text"`
	ComplianceControls      JSONStringSlice `gorm:"column:compliance_controls;type:text"`
	LifecycleState          string          `gorm:"column:lifecycle_state;default:draft;not null"`
	LifecycleReason         string          `gorm:"column:lifecycle_reason"`
	LifecycleChangedBy      string          `gorm:"column:lifecycle_changed_by;not null"`
	LifecycleChangedAt      *time.Time      `gorm:"column:lifecycle_changed_at"`
	AuditLastReviewedAt     *time.Time      `gorm:"column:audit_last_reviewed_at"`
	AuditReviewCadenceDays  int             `gorm:"column:audit_review_cadence_days"`
	CreatedAt               time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt               time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the GORM table name.
func (AssetGovernanceRecord) TableName() string { return "asset_governance" }

// AuditEventRecord is an immutable audit log entry.
type AuditEventRecord struct {
	ID            string          `gorm:"primaryKey;column:id;type:varchar(36)"`
	Namespace     string          `gorm:"column:namespace;index:idx_audit_ns_time,priority:1;default:default;not null"`
	CorrelationID string          `gorm:"column:correlation_id;index"`
	EventType     string          `gorm:"column:event_type;index:idx_audit_type_time,priority:1;not null"`
	Actor         string          `gorm:"column:actor;index:idx_audit_actor_time,priority:1;not null"`
	AssetUID      string          `gorm:"column:asset_uid;index:idx_audit_asset_time,priority:1"`
	VersionID     string          `gorm:"column:version_id"`
	Action        string          `gorm:"column:action"`
	Outcome       string          `gorm:"column:outcome;not null"` // success, failure, denied
	Reason        string          `gorm:"column:reason"`
	OldValue      JSONAny         `gorm:"column:old_value;type:text"`
	NewValue      JSONAny         `gorm:"column:new_value;type:text"`
	EventMetadata JSONAny         `gorm:"column:metadata;type:text"`
	RequestID     string          `gorm:"column:request_id;index"`
	Plugin        string          `gorm:"column:plugin;index:idx_audit_plugin_time,priority:1"`
	ResourceType  string          `gorm:"column:resource_type"`
	ResourceIDs   JSONStringSlice `gorm:"column:resource_ids;type:text"`
	ActionVerb    string          `gorm:"column:action_verb"`
	StatusCode    int             `gorm:"column:status_code"`
	CreatedAt     time.Time       `gorm:"column:created_at;index:idx_audit_type_time,priority:2;index:idx_audit_actor_time,priority:2;index:idx_audit_asset_time,priority:2;index:idx_audit_ns_time,priority:2;index:idx_audit_plugin_time,priority:2;autoCreateTime"`
}

// TableName returns the GORM table name.
func (AuditEventRecord) TableName() string { return "audit_events" }
