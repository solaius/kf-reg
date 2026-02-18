// Package conformance provides an importable conformance test harness
// for validating catalog plugins against the universal framework contract.
// External plugin developers can import this package and call RunConformance
// in their own test suites.
package conformance

// PluginsResponse is the response from /api/plugins.
type PluginsResponse struct {
	Plugins []PluginInfo `json:"plugins"`
	Count   int          `json:"count"`
}

// PluginInfo describes a single plugin as returned by /api/plugins.
type PluginInfo struct {
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Description    string          `json:"description"`
	BasePath       string          `json:"basePath"`
	Healthy        bool            `json:"healthy"`
	CapabilitiesV2 *CapabilitiesV2 `json:"capabilitiesV2,omitempty"`
	Management     *ManagementCaps `json:"management,omitempty"`
	Status         *PluginStatus   `json:"status,omitempty"`
}

// ManagementCaps describes management capabilities for a plugin.
type ManagementCaps struct {
	SourceManager bool `json:"sourceManager"`
	Refresh       bool `json:"refresh"`
	Diagnostics   bool `json:"diagnostics"`
	Actions       bool `json:"actions"`
}

// PluginStatus describes the runtime status of a plugin.
type PluginStatus struct {
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
	Serving     bool   `json:"serving"`
	LastError   string `json:"lastError,omitempty"`
}

// CapabilitiesV2 describes the V2 capabilities of a plugin.
type CapabilitiesV2 struct {
	SchemaVersion string             `json:"schemaVersion"`
	Plugin        PluginMeta         `json:"plugin"`
	Entities      []EntityCaps       `json:"entities"`
	Sources       *SourceCaps        `json:"sources,omitempty"`
	Actions       []ActionDefinition `json:"actions,omitempty"`
}

// PluginMeta contains identity information for a plugin.
type PluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	DisplayName string `json:"displayName,omitempty"`
}

// EntityCaps describes the capabilities of a single entity type.
type EntityCaps struct {
	Kind        string          `json:"kind"`
	Plural      string          `json:"plural"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
	Endpoints   EntityEndpoints `json:"endpoints"`
	Fields      EntityFields    `json:"fields"`
	Actions     []string        `json:"actions,omitempty"`
}

// EntityEndpoints holds the endpoint paths for an entity type.
type EntityEndpoints struct {
	List   string `json:"list"`
	Get    string `json:"get"`
	Action string `json:"action,omitempty"`
}

// EntityFields holds the field definitions for an entity type.
type EntityFields struct {
	Columns      []ColumnHint  `json:"columns"`
	FilterFields []FilterField `json:"filterFields,omitempty"`
	DetailFields []FieldHint   `json:"detailFields,omitempty"`
}

// ColumnHint describes a column for list view rendering.
type ColumnHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Sortable    bool   `json:"sortable,omitempty"`
	Width       string `json:"width,omitempty"`
}

// FilterField describes a filterable field.
type FilterField struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Type        string   `json:"type"`
	Options     []string `json:"options,omitempty"`
}

// FieldHint describes a field for detail view rendering.
type FieldHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Section     string `json:"section,omitempty"`
}

// SourceCaps describes source management capabilities.
type SourceCaps struct {
	Manageable  bool     `json:"manageable"`
	Refreshable bool     `json:"refreshable"`
	Types       []string `json:"types,omitempty"`
}

// ActionDefinition describes an action that can be performed on entities or sources.
type ActionDefinition struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Scope          string `json:"scope"`
	SupportsDryRun bool   `json:"supportsDryRun"`
	Idempotent     bool   `json:"idempotent"`
}
