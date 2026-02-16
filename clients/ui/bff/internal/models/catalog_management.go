package models

// SourceStatus represents the nested status of a source.
type SourceStatus struct {
	State       string `json:"state"`
	EntityCount int    `json:"entityCount"`
	LastRefresh string `json:"lastRefreshTime,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SourceInfo represents a source within a plugin catalog.
type SourceInfo struct {
	Id         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Enabled    bool                   `json:"enabled"`
	Status     SourceStatus           `json:"status"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// SourceInfoList represents the response from listing sources for a plugin.
type SourceInfoList struct {
	Sources []SourceInfo `json:"sources"`
	Count   int          `json:"count"`
}

// SourceConfigPayload represents the payload for validate/apply operations.
type SourceConfigPayload struct {
	Id         string                 `json:"id,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Enabled    *bool                  `json:"enabled,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ValidationResult represents the response from a source config validation.
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Message string   `json:"message,omitempty"`
}

// SourceEnableRequest represents the request to enable or disable a source.
type SourceEnableRequest struct {
	Enabled bool `json:"enabled"`
}

// RefreshResult represents the response from a refresh operation.
type RefreshResult struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	SourceId  string `json:"sourceId,omitempty"`
	ItemCount int    `json:"itemCount,omitempty"`
}

// PluginDiagnostics represents diagnostic information for a plugin.
type PluginDiagnostics struct {
	PluginName  string                 `json:"pluginName"`
	Healthy     bool                   `json:"healthy"`
	Uptime      string                 `json:"uptime,omitempty"`
	Version     string                 `json:"version,omitempty"`
	SourceCount int                    `json:"sourceCount,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}
