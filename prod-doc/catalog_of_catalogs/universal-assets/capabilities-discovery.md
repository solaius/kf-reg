# V2 Capabilities Discovery

## Overview

V2 Capabilities Discovery is the introspection mechanism that enables **generic UI and CLI rendering** for any catalog plugin without plugin-specific code. Each plugin advertises a structured `PluginCapabilitiesV2` document describing its entities, endpoints, fields, actions, and rendering hints. Generic consumers -- the React frontend and `catalogctl` CLI -- read these documents at startup and dynamically build navigation, list tables, detail views, filter bars, and action dialogs entirely from the advertised metadata.

**Location:** `pkg/catalog/plugin/`

### Design Goals

| Goal | How V2 Capabilities Achieves It |
|------|---------------------------------|
| Zero frontend code for new plugins | UI reads columns, filters, detailFields, and actions from the capabilities document |
| Zero CLI code for new plugins | `catalogctl` generates subcommands, table headers, and action flags at runtime |
| Single source of truth | The plugin itself owns the capabilities; no external mapping files |
| Progressive disclosure | V2 is optional; plugins that do not implement it get a V2 document built automatically from V1 interfaces |

### Discovery Flow

```
Plugin Startup
      |
      v
+--------------------------+
| Plugin implements one of |
| two paths:               |
|                          |
| A) CapabilitiesV2Provider|   --> Plugin returns handcrafted V2 document
|    GetCapabilitiesV2()   |
|                          |
| B) V1 interfaces only    |   --> BuildCapabilitiesV2() assembles from:
|    CapabilitiesProvider   |       - CapabilitiesProvider  (entity kinds, endpoints)
|    UIHintsProvider        |       - UIHintsProvider       (display hints)
|    CLIHintsProvider       |       - CLIHintsProvider      (table columns)
|    ActionProvider         |       - ActionProvider        (action definitions)
|    SourceManager          |       - SourceManager         (source management flags)
+--------------------------+
      |
      v
+----------------------------+
| Server caches V2 document  |
| per plugin at Init time    |
+----------------------------+
      |
      +------> GET /api/plugins                        (all plugins, inline V2)
      +------> GET /api/plugins/{pluginName}/capabilities  (single plugin, full V2)
      |
      v
+----------------------------+     +----------------------------+
| UI: CatalogContextProvider |     | CLI: discoverPlugins()     |
| fetches /api/plugins and   |     | fetches /api/plugins and   |
| builds nav items, routes,  |     | registers Cobra subcommands|
| tables, filters, actions   |     | with columns, flags, etc.  |
+----------------------------+     +----------------------------+
```

---

## V2 Capabilities Schema

### PluginCapabilitiesV2 (root)

The top-level document returned by the capabilities endpoints.

```go
// pkg/catalog/plugin/capabilities_types.go
type PluginCapabilitiesV2 struct {
    SchemaVersion string               `json:"schemaVersion"` // e.g. "v1"
    Plugin        PluginMeta           `json:"plugin"`
    Entities      []EntityCapabilities `json:"entities"`
    Sources       *SourceCapabilities  `json:"sources,omitempty"`
    Actions       []ActionDefinition   `json:"actions,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `schemaVersion` | string | Schema version for forward compatibility. Currently `"v1"`. |
| `plugin` | PluginMeta | Plugin identity and display metadata. |
| `entities` | []EntityCapabilities | One entry per entity kind the plugin manages. |
| `sources` | *SourceCapabilities | Source management capabilities. Nil if plugin has no manageable sources. |
| `actions` | []ActionDefinition | All actions the plugin supports, referenced by ID from entity `actions` arrays. |

### PluginMeta

```go
type PluginMeta struct {
    Name        string `json:"name"`
    Version     string `json:"version"`
    Description string `json:"description"`
    DisplayName string `json:"displayName,omitempty"`
    Icon        string `json:"icon,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Machine-readable plugin name (e.g. `"mcp"`, `"knowledge"`). Unique across the server. |
| `version` | string | API version string (e.g. `"v1alpha1"`). |
| `description` | string | Human-readable description of the plugin. |
| `displayName` | string | UI-friendly label (e.g. `"MCP Servers"`). Falls back to `name` if empty. |
| `icon` | string | Icon identifier for UI rendering (e.g. `"server"`, `"book"`, `"robot"`). |

### EntityCapabilities

Describes one entity kind managed by a plugin.

```go
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
```

| Field | Type | Description |
|-------|------|-------------|
| `kind` | string | Singular PascalCase entity kind (e.g. `"McpServer"`, `"KnowledgeSource"`). |
| `plural` | string | Lowercase plural used in URL paths (e.g. `"mcpservers"`, `"knowledgesources"`). |
| `displayName` | string | Human label for the UI sidebar and page headers. |
| `description` | string | Optional description shown in UI tooltips or CLI help text. |
| `endpoints` | EntityEndpoints | REST endpoint templates for list, get, and action operations. |
| `fields` | EntityFields | Column, filter, and detail field definitions that drive rendering. |
| `uiHints` | *EntityUIHints | Optional visual rendering hints (icon, color, section ordering). |
| `actions` | []string | IDs referencing top-level `ActionDefinition` entries that apply to this entity. |

### EntityEndpoints

```go
type EntityEndpoints struct {
    List   string `json:"list"`             // e.g. "/api/mcp_catalog/v1alpha1/mcpservers"
    Get    string `json:"get"`              // e.g. "/api/mcp_catalog/v1alpha1/mcpservers/{name}"
    Action string `json:"action,omitempty"` // e.g. "/api/mcp_catalog/v1alpha1/mcpservers/{name}:action"
}
```

The `{name}` placeholder is substituted by the client at call time. The `Action` endpoint uses the `:action` URL suffix pattern.

### EntityFields

Groups column, filter, and detail field definitions into one object.

```go
type EntityFields struct {
    Columns      []V2ColumnHint  `json:"columns"`
    FilterFields []V2FilterField `json:"filterFields,omitempty"`
    DetailFields []V2FieldHint   `json:"detailFields,omitempty"`
}
```

### V2ColumnHint

Describes a column in list-view tables.

```go
type V2ColumnHint struct {
    Name        string `json:"name"`
    DisplayName string `json:"displayName"`
    Path        string `json:"path"`                // JSON path in entity response, e.g. "protocol"
    Type        string `json:"type"`                // "string", "integer", "boolean", "array", "date"
    Sortable    bool   `json:"sortable,omitempty"`
    Width       string `json:"width,omitempty"`     // "sm", "md", "lg"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Machine identifier for the column. |
| `displayName` | string | Table header label. |
| `path` | string | Dot-separated JSON path to extract the value from the entity response. |
| `type` | string | Data type for formatting. One of `"string"`, `"integer"`, `"boolean"`, `"array"`, `"date"`. |
| `sortable` | bool | Whether the column supports server-side sorting via `orderBy`. |
| `width` | string | Width hint: `"sm"`, `"md"`, or `"lg"`. Interpreted by the UI table component. |

### V2FilterField

Describes a filterable field for the list-view filter bar.

```go
type V2FilterField struct {
    Name        string   `json:"name"`
    DisplayName string   `json:"displayName"`
    Type        string   `json:"type"`                   // "text", "select", "multiselect", "boolean"
    Options     []string `json:"options,omitempty"`       // for select/multiselect
    Operators   []string `json:"operators,omitempty"`     // "=", "!=", "LIKE", etc.
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Filter widget type: `"text"` (free-form input), `"select"` (dropdown), `"multiselect"`, `"boolean"` (toggle), `"number"`. |
| `options` | []string | Pre-populated choices for `select`/`multiselect` types. |
| `operators` | []string | Supported comparison operators for building `filterQuery` expressions. |

### V2FieldHint

Describes a field for the entity detail view.

```go
type V2FieldHint struct {
    Name        string `json:"name"`
    DisplayName string `json:"displayName"`
    Path        string `json:"path"`
    Type        string `json:"type"`
    Section     string `json:"section,omitempty"` // grouping for detail view
}
```

| Field | Type | Description |
|-------|------|-------------|
| `section` | string | Groups detail fields into collapsible sections (e.g. `"Overview"`, `"Connection"`, `"Statistics"`). The order of sections is controlled by `EntityUIHints.DetailSections`. |

### EntityUIHints

Optional visual rendering hints for a specific entity kind.

```go
type EntityUIHints struct {
    Icon           string   `json:"icon,omitempty"`
    Color          string   `json:"color,omitempty"`
    NameField      string   `json:"nameField,omitempty"`      // field to use as display name
    DetailSections []string `json:"detailSections,omitempty"` // ordered section names
}
```

| Field | Type | Description |
|-------|------|-------------|
| `icon` | string | Icon identifier for nav items and page headers. |
| `color` | string | Accent color for cards and badges. |
| `nameField` | string | JSON path to the field used as the primary display name (default: `"name"`). |
| `detailSections` | []string | Ordered list of section names. Detail fields are grouped and ordered by this list. |

### SourceCapabilities

Describes the plugin's data source management abilities.

```go
type SourceCapabilities struct {
    Manageable  bool     `json:"manageable"`
    Refreshable bool     `json:"refreshable"`
    Types       []string `json:"types,omitempty"` // "yaml", "http", etc.
}
```

| Field | Type | Description |
|-------|------|-------------|
| `manageable` | bool | Whether the plugin implements `SourceManager` (CRUD on sources). |
| `refreshable` | bool | Whether the plugin implements `RefreshProvider` (on-demand data reload). |
| `types` | []string | Source types the plugin accepts (e.g. `["yaml"]`). |

### ActionDefinition

Describes an action that can be invoked on entities or sources.

```go
type ActionDefinition struct {
    ID             string `json:"id"`
    DisplayName    string `json:"displayName"`
    Description    string `json:"description"`
    Scope          string `json:"scope"`          // "source" or "asset"
    SupportsDryRun bool   `json:"supportsDryRun"`
    Idempotent     bool   `json:"idempotent"`
    Destructive    bool   `json:"destructive,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Machine identifier referenced from `EntityCapabilities.Actions` arrays. |
| `scope` | string | `"source"` for source-level actions (refresh), `"asset"` for entity-level actions (tag, deprecate). |
| `supportsDryRun` | bool | Whether the action can be previewed without side effects. |
| `idempotent` | bool | Whether repeated invocations produce the same result. |
| `destructive` | bool | Whether the action has irreversible consequences. UI shows confirmation dialogs for destructive actions. |

---

## Capabilities Builder

When a plugin does not implement `CapabilitiesV2Provider` directly, the framework assembles a V2 document automatically via `BuildCapabilitiesV2()`.

```go
// pkg/catalog/plugin/capabilities_builder.go
func BuildCapabilitiesV2(p CatalogPlugin, basePath string) PluginCapabilitiesV2
```

### Assembly Logic

```
BuildCapabilitiesV2(plugin, basePath)
       |
       +-- Does plugin implement CapabilitiesV2Provider?
       |       |
       |      YES --> return plugin.GetCapabilitiesV2() directly
       |       |
       |      NO  --> assemble from V1 interfaces:
       |
       +-- Set SchemaVersion = "v1"
       +-- Set Plugin.Name, Version, Description from CatalogPlugin methods
       |
       +-- CapabilitiesProvider present?
       |       |
       |      YES --> For each EntityKind:
       |              - Create EntityCapabilities with Kind, Plural (auto-pluralized)
       |              - If ListEntities: set Endpoints.List = basePath + "/" + plural
       |              - If GetEntity:    set Endpoints.Get  = basePath + "/" + plural + "/{name}"
       |
       +-- SourceManager present?        --> Sources.Manageable = true
       +-- RefreshProvider present?       --> Sources.Refreshable = true
       |
       +-- Return assembled PluginCapabilitiesV2
```

### Pluralization

Entity kinds are automatically pluralized for URL paths by lowercasing the kind and appending `"s"`:

| Kind | Plural |
|------|--------|
| `Model` | `models` |
| `McpServer` | `mcpservers` |
| `ModelVersion` | `modelversions` |
| `KnowledgeSource` | `knowledgesources` |

### When to Implement CapabilitiesV2Provider

The builder fallback produces minimal V2 documents with no field definitions, no UI hints, and no actions. Plugins that want rich generic rendering should implement `CapabilitiesV2Provider` directly and return a fully populated document with columns, filters, detail fields, UI hints, and action references. All production plugins (MCP, Model, Knowledge, Agents, etc.) implement this interface.

---

## API Endpoints

### GET /api/plugins

Returns metadata for **all** registered plugins with inline V2 capabilities. This is the primary discovery endpoint used by both the UI and CLI at startup.

**Response Structure:**

```json
{
  "plugins": [
    {
      "name": "mcp",
      "version": "v1alpha1",
      "description": "McpServer catalog",
      "basePath": "/api/mcp_catalog/v1alpha1",
      "healthy": true,
      "entityKinds": ["McpServer"],
      "capabilities": { ... },
      "capabilitiesV2": { ... },
      "status": { ... },
      "management": {
        "sourceManager": true,
        "refresh": true,
        "diagnostics": true,
        "actions": true
      },
      "uiHints": { ... },
      "cliHints": { ... }
    }
  ],
  "count": 8
}
```

The `capabilitiesV2` field contains the full `PluginCapabilitiesV2` document, built via `BuildCapabilitiesV2()` for every plugin. Failed plugins are also included with `healthy: false` and their error message in `status.lastError`.

### GET /api/plugins/{pluginName}/capabilities

Returns the full V2 capabilities document for a **single** plugin. Returns `404` if the plugin name is not found among initialized plugins.

**Handler implementation:**

```go
// pkg/catalog/plugin/server.go
func (s *Server) capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
    pluginName := chi.URLParam(r, "pluginName")
    // ... find plugin among s.plugins ...
    v2caps := BuildCapabilitiesV2(found, basePath)
    json.NewEncoder(w).Encode(v2caps)
}
```

| Status Code | Condition |
|-------------|-----------|
| `200 OK` | Plugin found; returns `PluginCapabilitiesV2` JSON |
| `404 Not Found` | Plugin name not in registry; returns `{"error": "plugin \"x\" not found"}` |

---

## Example Response

Below is an annotated V2 capabilities document for the **MCP plugin**, as returned by `GET /api/plugins/mcp/capabilities`.

```json
{
  "schemaVersion": "v1",

  "plugin": {
    "name": "mcp",
    "version": "v1alpha1",
    "description": "McpServer catalog",
    "displayName": "MCP Servers",
    "icon": "server"
  },

  "entities": [
    {
      "kind": "McpServer",
      "plural": "mcpservers",
      "displayName": "MCP Server",
      "description": "Model Context Protocol server entries",

      "endpoints": {
        "list": "/api/mcp_catalog/v1alpha1/mcpservers",
        "get":  "/api/mcp_catalog/v1alpha1/mcpservers/{name}"
      },

      "fields": {
        "columns": [
          { "name": "name",           "displayName": "Name",       "path": "name",           "type": "string",  "sortable": true, "width": "lg" },
          { "name": "deploymentMode", "displayName": "Deployment", "path": "deploymentMode", "type": "string",  "sortable": true, "width": "md" },
          { "name": "provider",       "displayName": "Provider",   "path": "provider",       "type": "string",  "sortable": true, "width": "md" },
          { "name": "transportType",  "displayName": "Transport",  "path": "transportType",  "type": "string",  "sortable": true, "width": "sm" },
          { "name": "toolCount",      "displayName": "Tools",      "path": "toolCount",      "type": "integer", "sortable": true, "width": "sm" },
          { "name": "license",        "displayName": "License",    "path": "license",        "type": "string",  "sortable": true, "width": "md" },
          { "name": "category",       "displayName": "Category",   "path": "category",       "type": "string",  "sortable": true, "width": "md" }
        ],

        "filterFields": [
          { "name": "name",           "displayName": "Name",            "type": "text",   "operators": ["=", "!=", "LIKE"] },
          { "name": "deploymentMode", "displayName": "Deployment Mode", "type": "select", "options": ["local","remote","hybrid"], "operators": ["=","!="] },
          { "name": "provider",       "displayName": "Provider",        "type": "text",   "operators": ["=", "!=", "LIKE"] },
          { "name": "category",       "displayName": "Category",        "type": "text",   "operators": ["=", "!=", "LIKE"] },
          { "name": "license",        "displayName": "License",         "type": "text",   "operators": ["=", "!=", "LIKE"] },
          { "name": "transportType",  "displayName": "Transport Type",  "type": "select", "options": ["stdio","sse","streamable-http"], "operators": ["=","!="] },
          { "name": "toolCount",      "displayName": "Tool Count",      "type": "number", "operators": ["=", ">", "<", ">=", "<="] }
        ],

        "detailFields": [
          { "name": "name",                "displayName": "Name",                 "path": "name",                "type": "string",  "section": "Overview" },
          { "name": "description",         "displayName": "Description",          "path": "description",         "type": "string",  "section": "Overview" },
          { "name": "deploymentMode",      "displayName": "Deployment Mode",      "path": "deploymentMode",      "type": "string",  "section": "Overview" },
          { "name": "provider",            "displayName": "Provider",             "path": "provider",            "type": "string",  "section": "Overview" },
          { "name": "category",            "displayName": "Category",             "path": "category",            "type": "string",  "section": "Overview" },
          { "name": "license",             "displayName": "License",              "path": "license",             "type": "string",  "section": "Overview" },
          { "name": "serverUrl",           "displayName": "Server URL",           "path": "serverUrl",           "type": "string",  "section": "Connection" },
          { "name": "image",               "displayName": "Container Image",      "path": "image",               "type": "string",  "section": "Connection" },
          { "name": "endpoint",            "displayName": "Remote Endpoint",      "path": "endpoint",            "type": "string",  "section": "Connection" },
          { "name": "supportedTransports", "displayName": "Supported Transports", "path": "supportedTransports", "type": "string",  "section": "Connection" },
          { "name": "transportType",       "displayName": "Transport Type",       "path": "transportType",       "type": "string",  "section": "Connection" },
          { "name": "toolCount",           "displayName": "Tool Count",           "path": "toolCount",           "type": "integer", "section": "Statistics" },
          { "name": "resourceCount",       "displayName": "Resource Count",       "path": "resourceCount",       "type": "integer", "section": "Statistics" },
          { "name": "promptCount",         "displayName": "Prompt Count",         "path": "promptCount",         "type": "integer", "section": "Statistics" }
        ]
      },

      "uiHints": {
        "icon": "server",
        "nameField": "name",
        "detailSections": ["Overview", "Connection", "Statistics"]
      },

      "actions": ["tag", "annotate", "deprecate", "refresh"]
    }
  ],

  "sources": {
    "manageable": true,
    "refreshable": true,
    "types": ["yaml"]
  },

  "actions": [
    { "id": "tag",       "displayName": "Tag",       "description": "Add or remove tags on an entity",        "scope": "asset",  "supportsDryRun": true,  "idempotent": true },
    { "id": "annotate",  "displayName": "Annotate",  "description": "Add or update annotations on an entity", "scope": "asset",  "supportsDryRun": true,  "idempotent": true },
    { "id": "deprecate", "displayName": "Deprecate", "description": "Mark an entity as deprecated",           "scope": "asset",  "supportsDryRun": true,  "idempotent": true },
    { "id": "refresh",   "displayName": "Refresh",   "description": "Refresh entities from a source",         "scope": "source", "supportsDryRun": false, "idempotent": true }
  ]
}
```

---

## How the UI Consumes Capabilities

The React frontend uses a `CatalogContextProvider` that fetches capabilities at application startup and exposes them to all downstream components.

### Data Flow

```
Application Mount
       |
       v
CatalogContextProvider
       |
       +-- GET /api/plugins  (via BFF proxy)
       |       |
       |       v
       |   setPlugins(pluginList)
       |
       +-- For each plugin:
       |       GET /api/plugins/{name}/capabilities
       |       |
       |       v
       |   setCapabilitiesMap({ [name]: caps })
       |
       v
Context provides:
  - plugins[]                    # All registered plugins
  - capabilitiesMap{}            # name -> PluginCapabilitiesV2
  - getPluginCaps(name)          # Lookup helper
       |
       v
+------------------------------+
| GenericCatalogRoutes          |
| Reads capabilitiesMap and     |
| builds routes for each entity |
+------------------------------+
       |
       +----> PluginEntityListPage
       |        |
       |        +-- entity.fields.columns     --> Table headers + cell rendering
       |        +-- entity.fields.filterFields --> Filter bar dropdowns / inputs
       |        +-- entity.endpoints.list      --> Data fetch URL
       |        +-- GenericListView component
       |
       +----> PluginEntityDetailPage
       |        |
       |        +-- entity.fields.detailFields --> Field labels, values, sections
       |        +-- entity.uiHints.detailSections --> Section ordering
       |        +-- entity.endpoints.get       --> Data fetch URL
       |        +-- GenericDetailView component
       |
       +----> GenericActionDialog
                |
                +-- Top-level actions[]         --> Available action buttons
                +-- entity.actions[]            --> Which actions apply to this entity
                +-- action.supportsDryRun       --> Show "Preview" button
                +-- action.destructive          --> Show confirmation warning
```

### Key Components

| Component | Location | Reads From Capabilities |
|-----------|----------|------------------------|
| `CatalogContextProvider` | `clients/ui/frontend/src/app/context/catalog/CatalogContext.tsx` | Fetches `/api/plugins` and per-plugin capabilities |
| `GenericCatalogRoutes` | `clients/ui/frontend/src/app/pages/genericCatalog/GenericCatalogRoutes.tsx` | Builds React Router routes from entity definitions |
| `GenericListView` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericListView.tsx` | `entity.fields.columns` for table headers and cell rendering |
| `GenericDetailView` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericDetailView.tsx` | `entity.fields.detailFields` grouped by `section` |
| `GenericActionDialog` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericActionDialog.tsx` | `ActionDefinition` for action parameters, dry-run toggle, destructive warning |
| `PluginEntityListPage` | `clients/ui/frontend/src/app/pages/genericCatalog/screens/PluginEntityListPage.tsx` | Combines list view, filter bar, and pagination |
| `PluginEntityDetailPage` | `clients/ui/frontend/src/app/pages/genericCatalog/screens/PluginEntityDetailPage.tsx` | Combines detail view and action buttons |

### Column Rendering

The `GenericListView` iterates over `entity.fields.columns` to build table headers, then for each entity row extracts values using the `path` field:

```
columns: [
  { "path": "name",      "displayName": "Name",      "type": "string"  }
  { "path": "toolCount", "displayName": "Tools",      "type": "integer" }
]

Entity JSON: { "name": "filesystem", "toolCount": 11 }

Rendered row:  | filesystem | 11 |
```

The `getFieldValue(entity, path)` utility supports dot-separated paths for nested fields, and `formatFieldValue(value, type)` applies type-appropriate formatting (number locale, boolean badges, array comma-join, etc.).

---

## How catalogctl Consumes Capabilities

The `catalogctl` CLI dynamically generates its entire command tree from V2 capabilities at startup.

### Startup Sequence

```
catalogctl <command>
       |
       v
discoverPlugins()
       |
       +-- GET /api/plugins
       |       |
       |       v
       |   Parse pluginsResponse
       |
       +-- For each plugin with capabilitiesV2 != nil:
       |       |
       |       v
       |   buildPluginCommand(plugin)
       |       |
       |       +-- Create "catalogctl <plugin-name>" command group
       |       |
       |       +-- For each entity in capabilities.entities:
       |       |       buildEntityCommand(plugin, entity)
       |       |           +-- "list"  subcommand (uses entity.endpoints.list)
       |       |           +-- "get"   subcommand (uses entity.endpoints.get)
       |       |           +-- One subcommand per entity.actions[] entry
       |       |
       |       +-- If capabilities.sources != nil:
       |       |       buildSourcesCommand(plugin)
       |       |           +-- "list"    subcommand
       |       |           +-- "refresh" subcommand (if sources.refreshable)
       |       |
       |       +-- If len(capabilities.actions) > 0:
       |               buildActionsListCommand(plugin)
       |                   +-- "actions" subcommand (tabular summary)
       |
       v
rootCmd.AddCommand(pluginCmd) for each plugin
```

### Generated Command Tree (example)

```
catalogctl
  mcp                           # Plugin group
    mcpservers                  # Entity group
      list                      # List all MCP servers (table from columns)
      get [name]                # Get single server (detail output)
      tag [name]                # Asset action
      annotate [name]           # Asset action
      deprecate [name]          # Asset action
      refresh [name]            # Source action
    sources                     # Source management
      list                      # List configured sources
      refresh [source-id]       # Refresh specific or all sources
    actions                     # List all available actions
  knowledge                     # Another plugin group
    knowledgesources
      list
      get [name]
      ...
```

### Table Rendering From Columns

The `list` subcommand uses `entity.fields.columns` to build table headers and extract cell values:

```go
// cmd/catalogctl/entity.go
columns := entity.Fields.Columns
headers := make([]string, len(columns))
for i, col := range columns {
    headers[i] = col.DisplayName
}
for _, item := range items {
    row := make([]string, len(columns))
    for i, col := range columns {
        row[i] = truncate(extractValue(item, col.Path), 50)
    }
    rows = append(rows, row)
}
printTable(headers, rows)
```

The `extractValue()` utility traverses nested JSON maps using dot-separated paths, handling strings, numbers, booleans, and arrays.

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/capabilities_types.go` | All V2 capabilities type definitions (`PluginCapabilitiesV2`, `EntityCapabilities`, `V2ColumnHint`, `V2FilterField`, `V2FieldHint`, `EntityUIHints`, `SourceCapabilities`, `ActionDefinition`) |
| `pkg/catalog/plugin/capabilities_builder.go` | `BuildCapabilitiesV2()` function that assembles V2 from V1 interfaces; `pluralize()` helper |
| `pkg/catalog/plugin/capabilities_test.go` | Unit tests for builder, JSON round-trip, HTTP endpoint handlers, pluralization |
| `pkg/catalog/plugin/plugin.go` | `CapabilitiesV2Provider` interface definition |
| `pkg/catalog/plugin/server.go` | `capabilitiesHandler` and `pluginsHandler` HTTP handlers |
| `catalog/plugins/mcp/management.go` | MCP plugin `GetCapabilitiesV2()` implementation (reference example) |
| `clients/ui/frontend/src/app/context/catalog/CatalogContext.tsx` | `CatalogContextProvider` that fetches and caches capabilities for the UI |
| `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericListView.tsx` | Generic list table driven by `V2ColumnHint` |
| `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericDetailView.tsx` | Generic detail view driven by `V2FieldHint` and sections |
| `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericActionDialog.tsx` | Generic action dialog driven by `ActionDefinition` |
| `cmd/catalogctl/discover.go` | `discoverPlugins()` fetches `/api/plugins` and registers Cobra subcommands |
| `cmd/catalogctl/entity.go` | `buildEntityCommand()` creates list/get/action commands from entity capabilities |

---

[Back to Universal Assets](./README.md) | [Next: Asset Contract](./asset-contract.md)
