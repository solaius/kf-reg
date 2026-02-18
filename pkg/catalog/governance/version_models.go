package governance

import "time"

// AssetVersionRecord stores an immutable version snapshot.
type AssetVersionRecord struct {
	ID                      string     `gorm:"primaryKey;column:id;type:varchar(36)"`
	AssetUID                string     `gorm:"column:asset_uid;index:idx_version_asset;not null"`
	VersionID               string     `gorm:"column:version_id;uniqueIndex;not null"`
	VersionLabel            string     `gorm:"column:version_label;not null"`
	CreatedAt               time.Time  `gorm:"column:created_at;autoCreateTime"`
	CreatedBy               string     `gorm:"column:created_by;not null"`
	ContentDigest           string     `gorm:"column:content_digest"`
	SourceRevisionRef       string     `gorm:"column:source_revision_ref"`
	ProvenanceSourceType    string     `gorm:"column:provenance_source_type"`
	ProvenanceSourceURI     string     `gorm:"column:provenance_source_uri"`
	ProvenanceSourceID      string     `gorm:"column:provenance_source_id"`
	ProvenanceRevisionID    string     `gorm:"column:provenance_revision_id"`
	ProvenanceRevObservedAt *time.Time `gorm:"column:provenance_revision_observed_at"`
	ProvenanceVerified      bool       `gorm:"column:provenance_integrity_verified"`
	ProvenanceMethod        string     `gorm:"column:provenance_integrity_method"`
	ProvenanceDetails       string     `gorm:"column:provenance_integrity_details"`
	GovernanceSnapshot      JSONAny    `gorm:"column:governance_snapshot;type:text;not null"`
	AssetSnapshot           JSONAny    `gorm:"column:asset_snapshot;type:text;not null"`
}

// TableName returns the GORM table name.
func (AssetVersionRecord) TableName() string { return "asset_versions" }

// EnvBindingRecord maps (plugin, kind, name, environment) to a version.
type EnvBindingRecord struct {
	ID                string    `gorm:"primaryKey;column:id;type:varchar(36)"`
	Plugin            string    `gorm:"column:plugin;uniqueIndex:idx_binding_unique,priority:1;not null"`
	AssetKind         string    `gorm:"column:asset_kind;uniqueIndex:idx_binding_unique,priority:2;not null"`
	AssetName         string    `gorm:"column:asset_name;uniqueIndex:idx_binding_unique,priority:3;not null"`
	Environment       string    `gorm:"column:environment;uniqueIndex:idx_binding_unique,priority:4;not null"`
	AssetUID          string    `gorm:"column:asset_uid;index;not null"`
	VersionID         string    `gorm:"column:version_id;not null"`
	BoundAt           time.Time `gorm:"column:bound_at;not null"`
	BoundBy           string    `gorm:"column:bound_by;not null"`
	PreviousVersionID string    `gorm:"column:previous_version_id"`
}

// TableName returns the GORM table name.
func (EnvBindingRecord) TableName() string { return "env_bindings" }
