package plugin

// PluginCapabilitiesV2 is the expanded capabilities discovery document.
// It provides a complete description of a plugin's entities, endpoints,
// fields, and actions for generic UI/CLI rendering.
type PluginCapabilitiesV2 struct {
	SchemaVersion string               `json:"schemaVersion"` // e.g. "v1"
	Plugin        PluginMeta           `json:"plugin"`
	Entities      []EntityCapabilities `json:"entities"`
	Sources       *SourceCapabilities  `json:"sources,omitempty"`
	Actions       []ActionDefinition   `json:"actions,omitempty"`
}

// PluginMeta describes the plugin identity.
type PluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	DisplayName string `json:"displayName,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// EntityCapabilities describes one entity kind that a plugin manages.
type EntityCapabilities struct {
	Kind        string          `json:"kind"`
	Plural      string          `json:"plural"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
	Endpoints   EntityEndpoints `json:"endpoints"`
	Fields      EntityFields    `json:"fields"`
	UIHints     *EntityUIHints  `json:"uiHints,omitempty"`
	Actions     []string        `json:"actions,omitempty"` // references ActionDefinition.ID
}

// EntityEndpoints lists the REST paths for an entity kind.
type EntityEndpoints struct {
	List   string `json:"list"`             // e.g. "/api/mcp_catalog/v1alpha1/mcpservers"
	Get    string `json:"get"`              // e.g. "/api/mcp_catalog/v1alpha1/mcpservers/{name}"
	Action string `json:"action,omitempty"` // e.g. "/api/mcp_catalog/v1alpha1/mcpservers/{name}:action"
}

// EntityFields groups column, filter, and detail field definitions.
type EntityFields struct {
	Columns      []V2ColumnHint  `json:"columns"`
	FilterFields []V2FilterField `json:"filterFields,omitempty"`
	DetailFields []V2FieldHint   `json:"detailFields,omitempty"`
}

// V2ColumnHint describes a column for list views (V2 capabilities).
type V2ColumnHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`                // JSON path in entity response, e.g. "protocol"
	Type        string `json:"type"`                // "string", "integer", "boolean", "array", "date"
	Sortable    bool   `json:"sortable,omitempty"`
	Width       string `json:"width,omitempty"`     // "sm", "md", "lg"
}

// V2FilterField describes a filterable field for list views (V2 capabilities).
type V2FilterField struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Type        string   `json:"type"`                   // "text", "select", "multiselect", "boolean"
	Options     []string `json:"options,omitempty"`       // for select/multiselect
	Operators   []string `json:"operators,omitempty"`     // "=", "!=", "LIKE", etc.
}

// V2FieldHint describes a field for detail views (V2 capabilities).
type V2FieldHint struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Section     string `json:"section,omitempty"` // grouping for detail view
}

// EntityUIHints provides rendering hints for a specific entity kind.
type EntityUIHints struct {
	Icon           string   `json:"icon,omitempty"`
	Color          string   `json:"color,omitempty"`
	NameField      string   `json:"nameField,omitempty"`      // field to use as display name
	DetailSections []string `json:"detailSections,omitempty"` // ordered section names
}

// SourceCapabilities describes a plugin's data source management abilities.
type SourceCapabilities struct {
	Manageable  bool     `json:"manageable"`
	Refreshable bool     `json:"refreshable"`
	Types       []string `json:"types,omitempty"` // "yaml", "http", etc.
}

// ActionDefinition describes an action that can be invoked on entities or sources.
type ActionDefinition struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Scope          string `json:"scope"`          // "source" or "asset"
	SupportsDryRun bool   `json:"supportsDryRun"`
	Idempotent     bool   `json:"idempotent"`
	Destructive    bool   `json:"destructive,omitempty"`
}
