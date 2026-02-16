package models

// CatalogPluginCapabilities describes what a plugin supports.
type CatalogPluginCapabilities struct {
	EntityKinds  []string `json:"entityKinds,omitempty"`
	ListEntities bool     `json:"listEntities"`
	GetEntity    bool     `json:"getEntity"`
	ListSources  bool     `json:"listSources"`
	Artifacts    bool     `json:"artifacts"`
}

// CatalogPluginStatus provides detailed status information about a plugin.
type CatalogPluginStatus struct {
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
	Serving     bool   `json:"serving"`
	LastError   string `json:"lastError,omitempty"`
}

// CatalogPluginManagement describes management capabilities of a plugin.
type CatalogPluginManagement struct {
	SourceManager bool `json:"sourceManager"`
	Refresh       bool `json:"refresh"`
	Diagnostics   bool `json:"diagnostics"`
}

// CatalogPlugin represents a single plugin from the catalog server.
type CatalogPlugin struct {
	Name         string                     `json:"name"`
	Version      string                     `json:"version"`
	Description  string                     `json:"description"`
	BasePath     string                     `json:"basePath"`
	Healthy      bool                       `json:"healthy"`
	EntityKinds  []string                   `json:"entityKinds,omitempty"`
	Capabilities *CatalogPluginCapabilities `json:"capabilities,omitempty"`
	Status       *CatalogPluginStatus       `json:"status,omitempty"`
	Management   *CatalogPluginManagement   `json:"management,omitempty"`
}

// CatalogPluginList represents the response from GET /api/plugins.
type CatalogPluginList struct {
	Plugins []CatalogPlugin `json:"plugins"`
	Count   int             `json:"count"`
}
