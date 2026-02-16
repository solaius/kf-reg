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

// ValidationError describes a single validation problem.
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ValidationResult represents the response from a source config validation.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// SourceEnableRequest represents the request to enable or disable a source.
type SourceEnableRequest struct {
	Enabled bool `json:"enabled"`
}

// RefreshResult represents the response from a refresh operation.
type RefreshResult struct {
	SourceId        string `json:"sourceId,omitempty"`
	EntitiesLoaded  int    `json:"entitiesLoaded"`
	EntitiesRemoved int    `json:"entitiesRemoved"`
	Duration        int64  `json:"duration"`
	Error           string `json:"error,omitempty"`
}

// SourceDiagnostic provides diagnostic information for a single source.
type SourceDiagnostic struct {
	Id                  string `json:"id"`
	Name                string `json:"name"`
	State               string `json:"state"`
	EntityCount         int    `json:"entityCount"`
	LastRefreshTime     string `json:"lastRefreshTime,omitempty"`
	LastRefreshDuration int64  `json:"lastRefreshDuration,omitempty"`
	Error               string `json:"error,omitempty"`
}

// DiagnosticError represents a diagnostic-level error.
type DiagnosticError struct {
	Source  string `json:"source,omitempty"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

// PluginDiagnostics represents diagnostic information for a plugin.
type PluginDiagnostics struct {
	PluginName  string             `json:"pluginName"`
	Sources     []SourceDiagnostic `json:"sources"`
	LastRefresh string             `json:"lastRefresh,omitempty"`
	Errors      []DiagnosticError  `json:"errors,omitempty"`
}
