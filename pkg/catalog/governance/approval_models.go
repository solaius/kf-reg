package governance

import "time"

// ApprovalStatus represents the status of an approval request.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusDenied   ApprovalStatus = "denied"
	ApprovalStatusCanceled ApprovalStatus = "canceled"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// DecisionVerdict represents an individual reviewer's decision.
type DecisionVerdict string

const (
	VerdictApprove DecisionVerdict = "approve"
	VerdictDeny    DecisionVerdict = "deny"
)

// ApprovalRequestRecord is a GORM model for a pending or resolved approval request.
type ApprovalRequestRecord struct {
	ID             string         `gorm:"primaryKey;column:id;type:varchar(36)"`
	AssetUID       string         `gorm:"column:asset_uid;index:idx_approval_asset;not null"`
	Plugin         string         `gorm:"column:plugin;not null"`
	AssetKind      string         `gorm:"column:asset_kind;not null"`
	AssetName      string         `gorm:"column:asset_name;not null"`
	Action         string         `gorm:"column:action;not null"`
	ActionParams   JSONAny        `gorm:"column:action_params;type:text"`
	PolicyID       string         `gorm:"column:policy_id;not null"`
	RequiredCount  int            `gorm:"column:required_count;not null;default:1"`
	Status         ApprovalStatus `gorm:"column:status;index:idx_approval_status;not null;default:pending"`
	Requester      string         `gorm:"column:requester;not null"`
	Reason         string         `gorm:"column:reason"`
	ResolvedAt     *time.Time     `gorm:"column:resolved_at"`
	ResolvedBy     string         `gorm:"column:resolved_by"`
	ResolutionNote string         `gorm:"column:resolution_note"`
	ExpiresAt      *time.Time     `gorm:"column:expires_at"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the GORM table name.
func (ApprovalRequestRecord) TableName() string { return "approval_requests" }

// ApprovalDecisionRecord is a GORM model for an individual reviewer decision.
type ApprovalDecisionRecord struct {
	ID        string          `gorm:"primaryKey;column:id;type:varchar(36)"`
	RequestID string          `gorm:"column:request_id;index:idx_decision_request;not null"`
	Reviewer  string          `gorm:"column:reviewer;not null"`
	Verdict   DecisionVerdict `gorm:"column:verdict;not null"`
	Comment   string          `gorm:"column:comment"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime"`
}

// TableName returns the GORM table name.
func (ApprovalDecisionRecord) TableName() string { return "approval_decisions" }

// ApprovalRequest is the API-facing approval request type.
type ApprovalRequest struct {
	ID             string             `json:"id"`
	AssetRef       AssetRef           `json:"assetRef"`
	Action         string             `json:"action"`
	ActionParams   map[string]any     `json:"actionParams,omitempty"`
	PolicyID       string             `json:"policyId"`
	RequiredCount  int                `json:"requiredCount"`
	Status         ApprovalStatus     `json:"status"`
	Requester      string             `json:"requester"`
	Reason         string             `json:"reason,omitempty"`
	Decisions      []ApprovalDecision `json:"decisions,omitempty"`
	ResolvedAt     string             `json:"resolvedAt,omitempty"`
	ResolvedBy     string             `json:"resolvedBy,omitempty"`
	ResolutionNote string             `json:"resolutionNote,omitempty"`
	ExpiresAt      string             `json:"expiresAt,omitempty"`
	CreatedAt      string             `json:"createdAt"`
}

// ApprovalDecision is the API-facing approval decision type.
type ApprovalDecision struct {
	ID        string          `json:"id"`
	RequestID string          `json:"requestId"`
	Reviewer  string          `json:"reviewer"`
	Verdict   DecisionVerdict `json:"verdict"`
	Comment   string          `json:"comment,omitempty"`
	CreatedAt string          `json:"createdAt"`
}

// ApprovalRequestList is a paginated list of approval requests.
type ApprovalRequestList struct {
	Requests      []ApprovalRequest `json:"requests"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
	TotalSize     int               `json:"totalSize"`
}
