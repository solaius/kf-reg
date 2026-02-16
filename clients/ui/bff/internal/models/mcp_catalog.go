package models

// McpServer represents an MCP server entity from the catalog.
type McpServer struct {
	ID                  string         `json:"id,omitempty"`
	Name                string         `json:"name"`
	Description         string         `json:"description,omitempty"`
	ServerUrl           string         `json:"serverUrl"`
	TransportType       string         `json:"transportType,omitempty"`
	DeploymentMode      string         `json:"deploymentMode,omitempty"`
	Image               string         `json:"image,omitempty"`
	Endpoint            string         `json:"endpoint,omitempty"`
	SupportedTransports string         `json:"supportedTransports,omitempty"`
	License             string         `json:"license,omitempty"`
	Verified            bool           `json:"verified,omitempty"`
	Certified           bool           `json:"certified,omitempty"`
	Provider            string         `json:"provider,omitempty"`
	Logo                string         `json:"logo,omitempty"`
	Category            string         `json:"category,omitempty"`
	ToolCount           int            `json:"toolCount,omitempty"`
	ResourceCount       int            `json:"resourceCount,omitempty"`
	PromptCount         int            `json:"promptCount,omitempty"`
	CustomProperties    map[string]any `json:"customProperties,omitempty"`
}

// McpServerList represents a list of MCP servers.
type McpServerList struct {
	Items         []McpServer `json:"items"`
	Size          int         `json:"size"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
}
