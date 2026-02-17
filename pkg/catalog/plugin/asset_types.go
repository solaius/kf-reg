package plugin

// AssetResource is the universal envelope that wraps any catalog entity.
// Plugins project their native entities into this shape via AssetMapper.
// This is an additive projection -- existing plugin-specific API responses
// remain unchanged.
type AssetResource struct {
	APIVersion string         `json:"apiVersion"` // e.g. "catalog/v1alpha1"
	Kind       string         `json:"kind"`       // e.g. "McpServer", "CatalogModel"
	Metadata   AssetMetadata  `json:"metadata"`
	Spec       map[string]any `json:"spec"`
	Status     AssetStatus    `json:"status"`
}

// AssetMetadata carries identity and discovery metadata for an asset.
type AssetMetadata struct {
	UID         string            `json:"uid"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   string            `json:"createdAt,omitempty"`
	UpdatedAt   string            `json:"updatedAt,omitempty"`
	Owner       *AssetOwner       `json:"owner,omitempty"`
	SourceRef   *SourceRef        `json:"sourceRef,omitempty"`
}

// AssetOwner identifies the owner of an asset.
type AssetOwner struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Team  string `json:"team,omitempty"`
}

// SourceRef links an asset back to the catalog source it was ingested from.
type SourceRef struct {
	SourceID   string `json:"sourceId"`
	SourceName string `json:"sourceName,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
}

// AssetStatus captures lifecycle, health, and condition information for an asset.
type AssetStatus struct {
	Lifecycle  LifecycleStatus   `json:"lifecycle"`
	Health     HealthStatus      `json:"health"`
	Conditions []StatusCondition `json:"conditions,omitempty"`
	Links      *AssetLinks       `json:"links,omitempty"`
}

// LifecycleStatus represents the lifecycle phase of an asset.
type LifecycleStatus string

const (
	LifecycleActive     LifecycleStatus = "active"
	LifecycleDeprecated LifecycleStatus = "deprecated"
	LifecycleRetired    LifecycleStatus = "retired"
	LifecycleDraft      LifecycleStatus = "draft"
)

// HealthStatus represents the health of an asset.
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// StatusCondition is a single condition entry in an asset's status,
// following the Kubernetes-style type/status/reason pattern.
type StatusCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"` // "True", "False", "Unknown"
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// AssetLinks holds cross-references to related assets.
type AssetLinks struct {
	Related []LinkRef `json:"related,omitempty"`
}

// LinkRef is a typed reference to another asset.
type LinkRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	UID  string `json:"uid,omitempty"`
}

// AssetList is a paginated list of AssetResource items.
type AssetList struct {
	APIVersion    string          `json:"apiVersion"`
	Kind          string          `json:"kind"` // always "AssetList"
	Items         []AssetResource `json:"items"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
	TotalSize     int             `json:"totalSize,omitempty"`
}
