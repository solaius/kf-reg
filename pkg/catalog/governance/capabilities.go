package governance

// GovernanceCapabilities describes what governance features are enabled for a plugin's assets.
type GovernanceCapabilities struct {
	Supported  bool                    `json:"supported"`
	Lifecycle  *LifecycleCapabilities  `json:"lifecycle,omitempty"`
	Versioning *VersionCapabilities    `json:"versioning,omitempty"`
	Approvals  *ApprovalCapabilities   `json:"approvals,omitempty"`
	Provenance *ProvenanceCapabilities `json:"provenance,omitempty"`
}

// LifecycleCapabilities describes available lifecycle states and defaults.
type LifecycleCapabilities struct {
	States       []string `json:"states"`
	DefaultState string   `json:"defaultState"`
}

// VersionCapabilities describes versioning support.
type VersionCapabilities struct {
	Enabled      bool     `json:"enabled"`
	Environments []string `json:"environments"`
}

// ApprovalCapabilities describes approval workflow support.
type ApprovalCapabilities struct {
	Enabled bool `json:"enabled"`
}

// ProvenanceCapabilities describes provenance tracking support.
type ProvenanceCapabilities struct {
	Enabled bool `json:"enabled"`
}
