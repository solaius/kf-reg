package main

// pluginInfo mirrors the server /api/plugins response structure.
// The CLI is self-contained and does not import from the server.
type pluginInfo struct {
	Name           string               `json:"name"`
	Version        string               `json:"version"`
	Description    string               `json:"description"`
	BasePath       string               `json:"basePath"`
	Healthy        bool                 `json:"healthy"`
	EntityKinds    []string             `json:"entityKinds,omitempty"`
	CapabilitiesV2 *capabilitiesV2      `json:"capabilitiesV2,omitempty"`
	Management     *managementCaps      `json:"management,omitempty"`
	Status         *pluginStatusInfo    `json:"status,omitempty"`
	CLIHintsData   *cliHints            `json:"cliHints,omitempty"`
}

// capabilitiesV2 mirrors the server's PluginCapabilitiesV2 type.
type capabilitiesV2 struct {
	SchemaVersion string             `json:"schemaVersion"`
	Plugin        pluginMeta         `json:"plugin"`
	Entities      []entityCaps       `json:"entities"`
	Sources       *sourceCaps        `json:"sources,omitempty"`
	Actions       []actionDef        `json:"actions,omitempty"`
}

// pluginMeta describes the plugin identity.
type pluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	DisplayName string `json:"displayName,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// entityCaps describes one entity kind that a plugin manages.
type entityCaps struct {
	Kind        string          `json:"kind"`
	Plural      string          `json:"plural"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
	Endpoints   entityEndpoints `json:"endpoints"`
	Fields      entityFields    `json:"fields"`
	UIHints     *entityUIHints  `json:"uiHints,omitempty"`
	Actions     []string        `json:"actions,omitempty"`
}

// entityEndpoints lists the REST paths for an entity kind.
type entityEndpoints struct {
	List   string `json:"list"`
	Get    string `json:"get"`
	Action string `json:"action,omitempty"`
}

// entityFields groups column, filter, and detail field definitions.
type entityFields struct {
	Columns      []columnHint  `json:"columns"`
	FilterFields []filterField `json:"filterFields,omitempty"`
	DetailFields []fieldHint   `json:"detailFields,omitempty"`
}

// columnHint describes a column for list views.
type columnHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Sortable    bool   `json:"sortable,omitempty"`
	Width       string `json:"width,omitempty"`
}

// filterField describes a filterable field for list views.
type filterField struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Type        string   `json:"type"`
	Options     []string `json:"options,omitempty"`
	Operators   []string `json:"operators,omitempty"`
}

// fieldHint describes a field for detail views.
type fieldHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Section     string `json:"section,omitempty"`
}

// entityUIHints provides rendering hints for a specific entity kind.
type entityUIHints struct {
	Icon           string   `json:"icon,omitempty"`
	Color          string   `json:"color,omitempty"`
	NameField      string   `json:"nameField,omitempty"`
	DetailSections []string `json:"detailSections,omitempty"`
}

// sourceCaps describes a plugin's data source management abilities.
type sourceCaps struct {
	Manageable  bool     `json:"manageable"`
	Refreshable bool     `json:"refreshable"`
	Types       []string `json:"types,omitempty"`
}

// actionDef describes an action that can be invoked on entities or sources.
type actionDef struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Scope          string `json:"scope"`
	SupportsDryRun bool   `json:"supportsDryRun"`
	Idempotent     bool   `json:"idempotent"`
	Destructive    bool   `json:"destructive,omitempty"`
}

// managementCaps reports plugin management capabilities.
type managementCaps struct {
	SourceManager bool `json:"sourceManager"`
	Refresh       bool `json:"refresh"`
	Diagnostics   bool `json:"diagnostics"`
	Actions       bool `json:"actions"`
}

// pluginStatusInfo provides detailed status information about a plugin.
type pluginStatusInfo struct {
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
	Serving     bool   `json:"serving"`
	LastError   string `json:"lastError,omitempty"`
}

// cliHints provides display hints for CLI table rendering.
type cliHints struct {
	DefaultColumns   []string `json:"defaultColumns,omitempty"`
	SortField        string   `json:"sortField,omitempty"`
	FilterableFields []string `json:"filterableFields,omitempty"`
}

// pluginsResponse is the top-level response from GET /api/plugins.
type pluginsResponse struct {
	Plugins []pluginInfo `json:"plugins"`
	Count   int          `json:"count"`
}

// sourceInfo mirrors the server SourceInfo type.
type sourceInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Enabled    bool              `json:"enabled"`
	Labels     []string          `json:"labels,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
	Status     sourceStatusInfo  `json:"status,omitempty"`
}

// sourceStatusInfo mirrors server source status.
type sourceStatusInfo struct {
	State     string `json:"state"`
	LastError string `json:"lastError,omitempty"`
}

// actionRequest is the payload for executing an action.
type actionRequest struct {
	ActionID   string         `json:"actionId"`
	TargetName string         `json:"targetName,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
	DryRun     bool           `json:"dryRun,omitempty"`
}

// actionResponse is the response from executing an action.
type actionResponse struct {
	ActionID string `json:"actionId"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	DryRun   bool   `json:"dryRun,omitempty"`
}
