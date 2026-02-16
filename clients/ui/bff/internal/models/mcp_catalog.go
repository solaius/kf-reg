package models

// McpToolParameter represents a parameter for an MCP tool.
type McpToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// McpTool represents a tool provided by an MCP server.
type McpTool struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	AccessType  string             `json:"accessType"`
	Parameters  []McpToolParameter `json:"parameters,omitempty"`
}

// McpResource represents a resource provided by an MCP server.
type McpResource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri,omitempty"`
}

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
	Tools               []McpTool      `json:"tools,omitempty"`
	Resources           []McpResource  `json:"resources,omitempty"`
	Readme              string         `json:"readme,omitempty"`
	SourceUrl           string         `json:"sourceUrl,omitempty"`
	Version             string         `json:"version,omitempty"`
	LastModified        string         `json:"lastModified,omitempty"`
	Tags                []string       `json:"tags,omitempty"`
	SourceLabel         string         `json:"sourceLabel,omitempty"`
	CustomProperties    map[string]any `json:"customProperties,omitempty"`
}

// McpServerList represents a list of MCP servers.
type McpServerList struct {
	Items         []McpServer `json:"items"`
	Size          int         `json:"size"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
}
