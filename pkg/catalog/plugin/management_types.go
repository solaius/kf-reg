package plugin

import "time"

// SourceInfo provides detailed information about a configured data source.
type SourceInfo struct {
	// ID is the unique identifier for this source.
	ID string `json:"id"`

	// Name is the human-readable display name.
	Name string `json:"name"`

	// Type identifies the provider type (e.g., "yaml", "http", "hf").
	Type string `json:"type"`

	// Enabled indicates whether this source is active.
	Enabled bool `json:"enabled"`

	// Labels are tags for filtering and categorization.
	Labels []string `json:"labels,omitempty"`

	// Properties contains provider-specific configuration.
	Properties map[string]any `json:"properties,omitempty"`

	// Status provides runtime status of this source.
	Status SourceStatus `json:"status"`
}

// SourceStatus provides runtime status for a source.
type SourceStatus struct {
	// State is the current state (available, error, disabled, loading).
	State string `json:"state"`

	// LastRefreshTime is when the source was last refreshed.
	LastRefreshTime *time.Time `json:"lastRefreshTime,omitempty"`

	// LastRefreshStatus is the result of the last refresh (success, error).
	LastRefreshStatus string `json:"lastRefreshStatus,omitempty"`

	// LastRefreshSummary is a human-readable summary of the last refresh.
	LastRefreshSummary string `json:"lastRefreshSummary,omitempty"`

	// EntityCount is the number of entities loaded from this source.
	EntityCount int `json:"entityCount"`

	// Error is the last error message, if any.
	Error string `json:"error,omitempty"`
}

// Source status state constants.
const (
	SourceStateAvailable = "available"
	SourceStateError     = "error"
	SourceStateDisabled  = "disabled"
	SourceStateLoading   = "loading"
)

// SourceConfig represents source configuration for API-based mutations.
// This mirrors the file-based SourceConfig but is used for runtime changes.
type SourceConfigInput struct {
	// ID is the unique identifier for this source.
	ID string `json:"id"`

	// Name is the human-readable display name.
	Name string `json:"name"`

	// Type identifies the provider type (e.g., "yaml", "http", "hf").
	Type string `json:"type"`

	// Enabled indicates whether this source should be loaded.
	Enabled *bool `json:"enabled,omitempty"`

	// Labels are tags for filtering and categorization.
	Labels []string `json:"labels,omitempty"`

	// Properties contains provider-specific configuration.
	Properties map[string]any `json:"properties,omitempty"`

	// RefreshAfterApply indicates whether to trigger a refresh after applying
	// the source configuration. When true, the apply response includes the
	// refresh result with entity counts and timing.
	RefreshAfterApply *bool `json:"refreshAfterApply,omitempty"`
}

// ApplyResult is the result of applying a source configuration, optionally
// including refresh results when RefreshAfterApply was set.
type ApplyResult struct {
	// Status is "applied" on success.
	Status string `json:"status"`

	// RefreshResult is populated when refreshAfterApply was true.
	RefreshResult *RefreshResult `json:"refreshResult,omitempty"`
}

// ValidationResult is the result of validating a source configuration.
type ValidationResult struct {
	// Valid is true if the configuration is valid.
	Valid bool `json:"valid"`

	// Errors lists any validation errors found.
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidationError describes a single validation problem.
type ValidationError struct {
	// Field is the configuration field that has the error (empty for general errors).
	Field string `json:"field,omitempty"`

	// Message describes the error.
	Message string `json:"message"`
}

// RefreshResult is the result of a refresh operation.
type RefreshResult struct {
	// SourceID is the source that was refreshed (empty for refresh-all).
	SourceID string `json:"sourceId,omitempty"`

	// EntitiesLoaded is the number of entities loaded during refresh.
	EntitiesLoaded int `json:"entitiesLoaded"`

	// EntitiesRemoved is the number of entities removed during refresh.
	EntitiesRemoved int `json:"entitiesRemoved"`

	// Duration is how long the refresh took.
	Duration time.Duration `json:"duration"`

	// Error is the error message, if the refresh failed.
	Error string `json:"error,omitempty"`
}

// PluginDiagnostics provides diagnostic information about a plugin.
type PluginDiagnostics struct {
	// PluginName identifies the plugin.
	PluginName string `json:"pluginName"`

	// Sources provides per-source diagnostic information.
	Sources []SourceDiagnostic `json:"sources"`

	// LastRefresh is when any source was last refreshed.
	LastRefresh *time.Time `json:"lastRefresh,omitempty"`

	// Errors lists active diagnostic errors.
	Errors []DiagnosticError `json:"errors,omitempty"`
}

// SourceDiagnostic provides diagnostic information for a single source.
type SourceDiagnostic struct {
	// ID is the source identifier.
	ID string `json:"id"`

	// Name is the source display name.
	Name string `json:"name"`

	// State is the current state.
	State string `json:"state"`

	// EntityCount is the number of entities from this source.
	EntityCount int `json:"entityCount"`

	// LastRefreshTime is when this source was last refreshed.
	LastRefreshTime *time.Time `json:"lastRefreshTime,omitempty"`

	// LastRefreshDuration is how long the last refresh took.
	LastRefreshDuration *time.Duration `json:"lastRefreshDuration,omitempty"`

	// Error is the last error for this source.
	Error string `json:"error,omitempty"`
}

// DiagnosticError represents a diagnostic-level error.
type DiagnosticError struct {
	// Source is the source ID where the error occurred (empty for plugin-level).
	Source string `json:"source,omitempty"`

	// Message describes the error.
	Message string `json:"message"`

	// Time is when the error occurred.
	Time time.Time `json:"time"`
}

// UIHints provides display hints for rendering entities in the UI.
type UIHints struct {
	// ListColumns defines the columns to show in a list view.
	ListColumns []ColumnHint `json:"listColumns,omitempty"`

	// DetailFields defines the fields to show in a detail view.
	DetailFields []FieldHint `json:"detailFields,omitempty"`

	// IdentityField is the field used as the unique entity identifier.
	IdentityField string `json:"identityField,omitempty"`

	// DisplayNameField is the field used for display name.
	DisplayNameField string `json:"displayNameField,omitempty"`

	// DescriptionField is the field used for entity description.
	DescriptionField string `json:"descriptionField,omitempty"`
}

// ColumnHint describes a column in the entity list view.
type ColumnHint struct {
	// Field is the JSON field name in the entity.
	Field string `json:"field"`

	// Label is the human-readable column header.
	Label string `json:"label"`

	// Sortable indicates if this column supports sorting.
	Sortable bool `json:"sortable,omitempty"`

	// Filterable indicates if this column supports filtering.
	Filterable bool `json:"filterable,omitempty"`
}

// FieldHint describes a field in the entity detail view.
type FieldHint struct {
	// Field is the JSON field name in the entity.
	Field string `json:"field"`

	// Label is the human-readable label.
	Label string `json:"label"`

	// Section groups this field under a heading (empty for default section).
	Section string `json:"section,omitempty"`
}

// SecretRef references a value stored in a Kubernetes Secret.
// Used instead of inlining sensitive values in source configuration.
type SecretRef struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Key       string `json:"key" yaml:"key"`
}

// CLIHints provides display hints for CLI table rendering.
type CLIHints struct {
	// DefaultColumns lists the field names to show by default.
	DefaultColumns []string `json:"defaultColumns,omitempty"`

	// SortField is the default sort field.
	SortField string `json:"sortField,omitempty"`

	// FilterableFields lists fields that support filterQuery.
	FilterableFields []string `json:"filterableFields,omitempty"`
}
