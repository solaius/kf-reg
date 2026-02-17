package openapi

// KnowledgeSource represents a knowledge source entity.
type KnowledgeSource struct {
	Id                       string                 `json:"id,omitempty"`
	Name                     string                 `json:"name"`
	ExternalId               string                 `json:"externalId,omitempty"`
	Description              string                 `json:"description,omitempty"`
	CustomProperties         map[string]interface{} `json:"customProperties,omitempty"`
	CreateTimeSinceEpoch     string                 `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch string                 `json:"lastUpdateTimeSinceEpoch,omitempty"`
	SourceType               string                 `json:"sourceType,omitempty"`
	Location                 string                 `json:"location,omitempty"`
	ContentType              string                 `json:"contentType,omitempty"`
	Provider                 string                 `json:"provider,omitempty"`
	Status                   string                 `json:"status,omitempty"`
	DocumentCount            *int32                 `json:"documentCount,omitempty"`
	VectorDimensions         *int32                 `json:"vectorDimensions,omitempty"`
	IndexType                string                 `json:"indexType,omitempty"`
}

// KnowledgeSourceList is a paginated list of KnowledgeSource entities.
type KnowledgeSourceList struct {
	Items         []KnowledgeSource `json:"items"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
	Size          int32             `json:"size"`
}
