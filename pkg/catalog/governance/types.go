package governance

// LifecycleState represents asset lifecycle states.
type LifecycleState string

const (
	StateDraft      LifecycleState = "draft"
	StateApproved   LifecycleState = "approved"
	StateDeprecated LifecycleState = "deprecated"
	StateArchived   LifecycleState = "archived"
)

// RiskLevel represents asset risk classification.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// SLATier represents SLA classification.
type SLATier string

const (
	SLAGold   SLATier = "gold"
	SLASilver SLATier = "silver"
	SLABronze SLATier = "bronze"
	SLANone   SLATier = "none"
)

// GovernanceOverlay is the API-facing governance data for an asset.
type GovernanceOverlay struct {
	Owner       *OwnerInfo      `json:"owner,omitempty"`
	Team        *TeamInfo       `json:"team,omitempty"`
	SLA         *SLAInfo        `json:"sla,omitempty"`
	Risk        *RiskInfo       `json:"risk,omitempty"`
	IntendedUse *IntendedUse    `json:"intendedUse,omitempty"`
	Compliance  *ComplianceInfo `json:"compliance,omitempty"`
	Lifecycle   *LifecycleInfo  `json:"lifecycle,omitempty"`
	Audit       *AuditMetadata  `json:"audit,omitempty"`
}

// OwnerInfo describes the owner of an asset.
type OwnerInfo struct {
	Principal   string `json:"principal,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
}

// TeamInfo describes the team responsible for an asset.
type TeamInfo struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// SLAInfo describes the SLA for an asset.
type SLAInfo struct {
	Tier          SLATier `json:"tier,omitempty"`
	ResponseHours int     `json:"responseHours,omitempty"`
}

// RiskInfo describes the risk classification of an asset.
type RiskInfo struct {
	Level      RiskLevel `json:"level,omitempty"`
	Categories []string  `json:"categories,omitempty"`
}

// IntendedUse describes the intended use of an asset.
type IntendedUse struct {
	Summary      string   `json:"summary,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Restrictions []string `json:"restrictions,omitempty"`
}

// ComplianceInfo describes compliance metadata for an asset.
type ComplianceInfo struct {
	Tags     []string `json:"tags,omitempty"`
	Controls []string `json:"controls,omitempty"`
}

// LifecycleInfo describes the lifecycle state of an asset.
type LifecycleInfo struct {
	State     LifecycleState `json:"state"`
	Reason    string         `json:"reason,omitempty"`
	ChangedBy string         `json:"changedBy,omitempty"`
	ChangedAt string         `json:"changedAt,omitempty"` // RFC3339
}

// AuditMetadata describes audit review configuration for an asset.
type AuditMetadata struct {
	LastReviewedAt    string `json:"lastReviewedAt,omitempty"`
	ReviewCadenceDays int    `json:"reviewCadenceDays,omitempty"`
}

// AssetRef identifies an asset in the governance system.
type AssetRef struct {
	Plugin string `json:"plugin"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
}

// GovernanceResponse is the API response for GET governance.
type GovernanceResponse struct {
	AssetRef   AssetRef          `json:"assetRef"`
	Governance GovernanceOverlay `json:"governance"`
}

// GovernanceConfig holds runtime governance configuration.
type GovernanceConfig struct {
	Environments   []string             `yaml:"environments" json:"environments"`
	TrustedSources []string             `yaml:"trustedSources" json:"trustedSources"`
	AuditRetention AuditRetentionConfig `yaml:"auditRetention" json:"auditRetention"`
}

// AuditRetentionConfig holds audit retention configuration.
type AuditRetentionConfig struct {
	Days int `yaml:"days" json:"days"`
}

// AuditEvent is the API-facing audit event.
type AuditEvent struct {
	ID            string         `json:"id"`
	CorrelationID string         `json:"correlationId"`
	EventType     string         `json:"eventType"`
	Actor         string         `json:"actor"`
	AssetUID      string         `json:"assetUid"`
	VersionID     string         `json:"versionId,omitempty"`
	Action        string         `json:"action,omitempty"`
	Outcome       string         `json:"outcome"`
	Reason        string         `json:"reason,omitempty"`
	OldValue      map[string]any `json:"oldValue,omitempty"`
	NewValue      map[string]any `json:"newValue,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     string         `json:"createdAt"`
}

// AuditEventList is a paginated list of audit events.
type AuditEventList struct {
	Events        []AuditEvent `json:"events"`
	NextPageToken string       `json:"nextPageToken,omitempty"`
	TotalSize     int          `json:"totalSize"`
}
