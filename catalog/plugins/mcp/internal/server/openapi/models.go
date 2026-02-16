package openapi

// McpServer represents an MCP server entity.
type McpServer struct {
	Id                       string                 `json:"id,omitempty"`
	Name                     string                 `json:"name"`
	ExternalId               string                 `json:"externalId,omitempty"`
	Description              string                 `json:"description,omitempty"`
	CustomProperties         map[string]interface{} `json:"customProperties,omitempty"`
	CreateTimeSinceEpoch     string                 `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch string                 `json:"lastUpdateTimeSinceEpoch,omitempty"`
	ServerUrl                string                 `json:"serverUrl,omitempty"`
	TransportType            string                 `json:"transportType,omitempty"`
	ToolCount                *int32                 `json:"toolCount,omitempty"`
	ResourceCount            *int32                 `json:"resourceCount,omitempty"`
	PromptCount              *int32                 `json:"promptCount,omitempty"`
	DeploymentMode           string                 `json:"deploymentMode,omitempty"`
	Image                    string                 `json:"image,omitempty"`
	Endpoint                 string                 `json:"endpoint,omitempty"`
	SupportedTransports      string                 `json:"supportedTransports,omitempty"`
	License                  string                 `json:"license,omitempty"`
	Verified                 *bool                  `json:"verified,omitempty"`
	Certified                *bool                  `json:"certified,omitempty"`
	Provider                 string                 `json:"provider,omitempty"`
	Logo                     string                 `json:"logo,omitempty"`
	Category                 string                 `json:"category,omitempty"`
}

// McpServerList is a paginated list of McpServer entities.
type McpServerList struct {
	Items         []McpServer `json:"items"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
	Size          int32       `json:"size"`
}
