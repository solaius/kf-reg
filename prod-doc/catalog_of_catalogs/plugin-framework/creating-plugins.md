# Creating a New Catalog Plugin

## Overview

This guide walks through creating a new asset-type catalog plugin from scratch
following the Phase 5 universal framework patterns.  A well-formed plugin
registers itself at process startup, serves list/get HTTP endpoints for its
entities, advertises V2 capabilities so that the generic UI and CLI can render
it without any code changes, and implements the standard action set (tag,
annotate, deprecate) via the shared `BuiltinActionHandler`.

The **knowledge sources plugin** (`catalog/plugins/knowledge/`) is the
canonical minimal example.  Every code snippet in this guide is drawn from it.

## Plugin Anatomy

```
catalog/plugins/myplugin/
+-- register.go         init() + plugin.Register()
+-- plugin.go           CatalogPlugin implementation
+-- asset_mapper.go     AssetMapperProvider
+-- actions.go          ActionProvider (uses BuiltinActionHandler)
+-- management.go       CapabilitiesV2Provider + optional SourceManager
+-- data/
    +-- entities.yaml   Sample YAML data
```

The server discovers the plugin at startup through a Go blank import in
`cmd/catalog-server/main.go`:

```
cmd/catalog-server/main.go          catalog/plugins/myplugin/register.go
+----------------------------------+   +--------------------------------+
| import (                         |   | func init() {                  |
|   _ ".../catalog/plugins/myplugin"|-->|   plugin.Register(&MyPlugin{})|
| )                                |   | }                              |
+----------------------------------+   +--------------------------------+
                                               |
                                               v
                                       plugin.Registry (global)
                                               |
                                               v
                                       server.Init(ctx)
                                         -> plugin.Init()
                                         -> plugin.RegisterRoutes()
                                         -> plugin.Start()
```

---

## Step 1: Define Entity Types

Create a Go struct that represents your native entity.  Include YAML and JSON
struct tags so the same type works for both file parsing and HTTP serialization.

```go
// catalog/plugins/myplugin/plugin.go

// MyEntity is the in-memory representation of one catalog item.
type MyEntity struct {
    Name        string  `yaml:"name"        json:"name"`
    ExternalId  string  `yaml:"externalId"  json:"externalId,omitempty"`
    Description *string `yaml:"description" json:"description,omitempty"`
    Category    *string `yaml:"category"    json:"category,omitempty"`
    Provider    *string `yaml:"provider"    json:"provider,omitempty"`
    Status      *string `yaml:"status"      json:"status,omitempty"`
    SourceId    string  `yaml:"-"           json:"sourceId,omitempty"`
}
```

Then create a sample YAML data file under `data/`:

```yaml
# catalog/plugins/myplugin/data/entities.yaml
myentities:
  - name: example-entity-1
    description: "First example entity"
    category: "general"
    provider: "Internal"
    status: active

  - name: example-entity-2
    description: "Second example entity"
    category: "specialized"
    provider: "External"
    status: active
```

The top-level YAML key (here `myentities`) matches a deserialization wrapper
struct:

```go
type myEntityCatalog struct {
    MyEntities []MyEntity `yaml:"myentities"`
}
```

---

## Step 2: Implement CatalogPlugin

The `CatalogPlugin` interface is the only required contract.  Every other
feature is opt-in via optional interfaces.

```go
// catalog/plugins/myplugin/plugin.go
package myplugin

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "sync"
    "sync/atomic"

    "github.com/go-chi/chi/v5"
    "gorm.io/gorm"

    "github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

const (
    PluginName    = "myplugin"
    PluginVersion = "v1alpha1"
)

type MyPlugin struct {
    cfg     plugin.Config
    logger  *slog.Logger
    db      *gorm.DB
    healthy atomic.Bool
    started atomic.Bool
    mu      sync.RWMutex
    sources map[string][]MyEntity // sourceID -> entries
}
```

### Identity Methods

```go
func (p *MyPlugin) Name() string        { return PluginName }
func (p *MyPlugin) Version() string     { return PluginVersion }
func (p *MyPlugin) Description() string { return "My custom entity catalog" }
```

### Init

`Init` receives the shared database, a namespaced logger, and the plugin's
section from `sources.yaml`.  Load data from configured sources and store
entities in memory.

```go
func (p *MyPlugin) Init(ctx context.Context, cfg plugin.Config) error {
    p.cfg = cfg
    p.logger = cfg.Logger
    if p.logger == nil {
        p.logger = slog.Default()
    }
    p.db = cfg.DB
    p.sources = make(map[string][]MyEntity)

    p.logger.Info("initializing myplugin")

    for _, src := range cfg.Section.Sources {
        if !src.IsEnabled() {
            continue
        }
        if src.Type == "yaml" {
            entries, err := loadYAMLSource(src)
            if err != nil {
                p.logger.Error("failed to load source", "source", src.ID, "error", err)
                continue
            }
            for i := range entries {
                entries[i].SourceId = src.ID
            }
            p.sources[src.ID] = entries
            p.logger.Info("loaded source", "source", src.ID, "entries", len(entries))
        }
    }

    p.healthy.Store(true)
    return nil
}
```

### Start / Stop

Use `Start` for background goroutines (polling, sync) and `Stop` for cleanup.
For a simple YAML-backed plugin these can be minimal.

```go
func (p *MyPlugin) Start(ctx context.Context) error {
    p.started.Store(true)
    return nil
}

func (p *MyPlugin) Stop(ctx context.Context) error {
    p.started.Store(false)
    p.healthy.Store(false)
    return nil
}
```

### Healthy

Return `true` once entities are loaded.  The server checks this for the
`/readyz` probe and the `/api/plugins` listing.

```go
func (p *MyPlugin) Healthy() bool {
    return p.healthy.Load()
}
```

### RegisterRoutes

Mount list and get handlers on the chi sub-router that the server provides,
scoped to the plugin's base path.

```go
func (p *MyPlugin) RegisterRoutes(router chi.Router) error {
    router.Get("/myentities", p.listHandler)
    router.Get("/myentities/{name}", p.getHandler)
    return nil
}
```

The server mounts this sub-router at `/api/myplugin_catalog/v1alpha1/`, so the
full paths become:

```
GET /api/myplugin_catalog/v1alpha1/myentities
GET /api/myplugin_catalog/v1alpha1/myentities/{name}
```

### Migrations

Return an empty slice if using in-memory storage.  For database-backed plugins,
return GORM auto-migration targets.

```go
func (p *MyPlugin) Migrations() []plugin.Migration {
    return nil
}
```

### HTTP Handlers

List and get handlers follow a standard pattern.  The list handler supports
`filterQuery` for consistency with other plugins.

```go
func (p *MyPlugin) listHandler(w http.ResponseWriter, r *http.Request) {
    entries := p.allEntries()

    response := map[string]any{
        "items":    entries,
        "size":     len(entries),
        "pageSize": len(entries),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (p *MyPlugin) getHandler(w http.ResponseWriter, r *http.Request) {
    name := chi.URLParam(r, "name")
    entries := p.allEntries()

    for _, entry := range entries {
        if entry.Name == name {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(entry)
            return
        }
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    json.NewEncoder(w).Encode(map[string]string{
        "error": fmt.Sprintf("entity %q not found", name),
    })
}
```

---

## Step 3: Register via init()

Create a one-line registration file.  The `init()` function runs at import
time and adds the plugin to the global registry.

```go
// catalog/plugins/myplugin/register.go
package myplugin

import "github.com/kubeflow/model-registry/pkg/catalog/plugin"

func init() {
    plugin.Register(&MyPlugin{})
}
```

Then add a blank import in `cmd/catalog-server/main.go`:

```go
import (
    _ "github.com/kubeflow/model-registry/catalog/plugins/model"
    _ "github.com/kubeflow/model-registry/catalog/plugins/mcp"
    _ "github.com/kubeflow/model-registry/catalog/plugins/knowledge"
    _ "github.com/kubeflow/model-registry/catalog/plugins/myplugin"  // <-- add this
)
```

This is the only file outside `catalog/plugins/myplugin/` that needs editing
for the plugin to appear in the server.

---

## Step 4: Implement CapabilitiesV2Provider

The V2 capabilities document is what drives the generic UI and CLI.  Without
it, your plugin serves its API but does not appear in the UI navigation or
`catalogctl` output.

```go
// Compile-time interface assertion.
var _ plugin.CapabilitiesV2Provider = (*MyPlugin)(nil)

func (p *MyPlugin) GetCapabilitiesV2() plugin.PluginCapabilitiesV2 {
    basePath := "/api/myplugin_catalog/v1alpha1"
    return plugin.PluginCapabilitiesV2{
        SchemaVersion: "v1",
        Plugin: plugin.PluginMeta{
            Name:        PluginName,
            Version:     PluginVersion,
            Description: "My custom entity catalog",
            DisplayName: "My Entities",
            Icon:        "cube",
        },
        Entities: []plugin.EntityCapabilities{
            {
                Kind:        "MyEntity",
                Plural:      "myentities",
                DisplayName: "My Entity",
                Description: "Custom asset type",
                Endpoints: plugin.EntityEndpoints{
                    List:   basePath + "/myentities",
                    Get:    basePath + "/myentities/{name}",
                    Action: basePath + "/myentities/{name}:action",
                },
                Fields: plugin.EntityFields{
                    Columns: []plugin.V2ColumnHint{
                        {Name: "name",     DisplayName: "Name",     Path: "name",     Type: "string",  Sortable: true, Width: "lg"},
                        {Name: "category", DisplayName: "Category", Path: "category", Type: "string",  Sortable: true, Width: "md"},
                        {Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string",  Sortable: true, Width: "md"},
                        {Name: "status",   DisplayName: "Status",   Path: "status",   Type: "string",  Sortable: true, Width: "sm"},
                    },
                    FilterFields: []plugin.V2FilterField{
                        {Name: "name",     DisplayName: "Name",     Type: "text",   Operators: []string{"=", "!=", "LIKE"}},
                        {Name: "category", DisplayName: "Category", Type: "select", Options: []string{"general", "specialized"}, Operators: []string{"=", "!="}},
                        {Name: "provider", DisplayName: "Provider", Type: "text",   Operators: []string{"=", "!=", "LIKE"}},
                        {Name: "status",   DisplayName: "Status",   Type: "select", Options: []string{"active", "draft", "archived"}, Operators: []string{"=", "!="}},
                    },
                    DetailFields: []plugin.V2FieldHint{
                        {Name: "name",        DisplayName: "Name",        Path: "name",        Type: "string", Section: "Overview"},
                        {Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
                        {Name: "category",    DisplayName: "Category",    Path: "category",    Type: "string", Section: "Overview"},
                        {Name: "provider",    DisplayName: "Provider",    Path: "provider",    Type: "string", Section: "Details"},
                        {Name: "status",      DisplayName: "Status",      Path: "status",      Type: "string", Section: "Details"},
                    },
                },
                UIHints: &plugin.EntityUIHints{
                    Icon:           "cube",
                    NameField:      "name",
                    DetailSections: []string{"Overview", "Details"},
                },
                Actions: []string{"tag", "annotate", "deprecate"},
            },
        },
        Sources: &plugin.SourceCapabilities{
            Manageable:  true,
            Refreshable: true,
            Types:       []string{"yaml"},
        },
        Actions: []plugin.ActionDefinition{
            {ID: "tag",       DisplayName: "Tag",       Description: "Add or remove tags on an entity",   Scope: "asset",  SupportsDryRun: true, Idempotent: true},
            {ID: "annotate",  DisplayName: "Annotate",  Description: "Add or update annotations",         Scope: "asset",  SupportsDryRun: true, Idempotent: true},
            {ID: "deprecate", DisplayName: "Deprecate", Description: "Mark an entity as deprecated",      Scope: "asset",  SupportsDryRun: true, Idempotent: true},
            {ID: "refresh",   DisplayName: "Refresh",   Description: "Refresh entities from a source",    Scope: "source", SupportsDryRun: false, Idempotent: true},
        },
    }
}
```

### Capabilities Field Reference

| Field Group | Purpose |
|-------------|---------|
| `Plugin` | Identity: name, version, display name, icon |
| `Entities[].Endpoints` | REST paths the generic UI fetches from |
| `Entities[].Fields.Columns` | Columns rendered in the list table (V2ColumnHint) |
| `Entities[].Fields.FilterFields` | Filter controls shown above the list (V2FilterField) |
| `Entities[].Fields.DetailFields` | Fields grouped by section on the detail page (V2FieldHint) |
| `Entities[].UIHints` | Icon, name field, ordered detail sections |
| `Entities[].Actions` | Action IDs this entity supports (references top-level Actions) |
| `Sources` | Whether sources are manageable and refreshable |
| `Actions` | Full action definitions (id, scope, dry-run, idempotent) |

---

## Step 5: Implement AssetMapperProvider

The asset mapper converts your native entities into the universal
`AssetResource` envelope.  This powers the cross-plugin `/api/assets` endpoint
and the generic detail view.

```go
// catalog/plugins/myplugin/asset_mapper.go
package myplugin

import (
    "fmt"

    "github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.AssetMapperProvider = (*MyPlugin)(nil)

func (p *MyPlugin) GetAssetMapper() plugin.AssetMapper {
    return &myEntityMapper{}
}

type myEntityMapper struct{}

func (m *myEntityMapper) SupportedKinds() []string {
    return []string{"MyEntity"}
}

func (m *myEntityMapper) MapToAsset(entity any) (plugin.AssetResource, error) {
    switch e := entity.(type) {
    case MyEntity:
        return mapToAsset(e), nil
    case *MyEntity:
        if e == nil {
            return plugin.AssetResource{}, fmt.Errorf("nil MyEntity pointer")
        }
        return mapToAsset(*e), nil
    default:
        return plugin.AssetResource{}, fmt.Errorf("unsupported entity type %T", entity)
    }
}

func (m *myEntityMapper) MapToAssets(entities []any) ([]plugin.AssetResource, error) {
    return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}
```

The `mapToAsset` helper populates the universal envelope:

```go
func mapToAsset(e MyEntity) plugin.AssetResource {
    spec := make(map[string]any)
    if e.Category != nil {
        spec["category"] = *e.Category
    }
    if e.Provider != nil {
        spec["provider"] = *e.Provider
    }
    if e.Status != nil {
        spec["status"] = *e.Status
    }

    desc := ""
    if e.Description != nil {
        desc = *e.Description
    }

    asset := plugin.AssetResource{
        APIVersion: "catalog/v1alpha1",
        Kind:       "MyEntity",
        Metadata: plugin.AssetMetadata{
            Name:        e.Name,
            Description: desc,
        },
        Spec:   spec,
        Status: plugin.DefaultAssetStatus(),
    }

    if e.SourceId != "" {
        asset.Metadata.SourceRef = &plugin.SourceRef{
            SourceID: e.SourceId,
        }
    }

    return asset
}
```

### Mapping Rules

```
Native Entity Field           AssetResource Target
-----------------------------  --------------------------------
Name                           Metadata.Name
Description                    Metadata.Description
Labels / Tags                  Metadata.Labels / Metadata.Tags
SourceId                       Metadata.SourceRef.SourceID
Entity-specific fields         Spec map (key = field name)
(no explicit status)           Status = DefaultAssetStatus()
```

Use `plugin.DefaultAssetStatus()` to set lifecycle to `active` and health to
`unknown`.  The overlay store can modify these later via actions.

---

## Step 6: Implement ActionProvider

Actions let users tag, annotate, and deprecate entities through the `:action`
endpoints.  The `BuiltinActionHandler` provides the standard three actions
backed by the shared `OverlayStore` database table.

```go
// catalog/plugins/myplugin/actions.go
package myplugin

import (
    "context"
    "fmt"

    "github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

// Compile-time interface assertion.
var _ plugin.ActionProvider = (*MyPlugin)(nil)

func (p *MyPlugin) HandleAction(ctx context.Context, scope plugin.ActionScope, targetID string, req plugin.ActionRequest) (*plugin.ActionResult, error) {
    switch scope {
    case plugin.ActionScopeSource:
        return p.handleSourceAction(ctx, targetID, req)
    case plugin.ActionScopeAsset:
        return p.handleAssetAction(ctx, targetID, req)
    default:
        return nil, fmt.Errorf("unknown action scope %q", scope)
    }
}

func (p *MyPlugin) ListActions(scope plugin.ActionScope) []plugin.ActionDefinition {
    switch scope {
    case plugin.ActionScopeSource:
        return []plugin.ActionDefinition{
            {
                ID:          "refresh",
                DisplayName: "Refresh",
                Description: "Refresh entities from source",
                Scope:       string(plugin.ActionScopeSource),
                Idempotent:  true,
            },
        }
    case plugin.ActionScopeAsset:
        return plugin.BuiltinActionDefinitions()
    default:
        return nil
    }
}
```

### Source Actions

Handle source-scoped actions (typically just `refresh`):

```go
func (p *MyPlugin) handleSourceAction(ctx context.Context, sourceID string, req plugin.ActionRequest) (*plugin.ActionResult, error) {
    switch req.Action {
    case "refresh":
        // reload the YAML file for this source
        // ... (see knowledge plugin Refresh method for full example)
        return &plugin.ActionResult{
            Action:  "refresh",
            Status:  "completed",
            Message: fmt.Sprintf("Refreshed source %s", sourceID),
        }, nil
    default:
        return nil, fmt.Errorf("unknown source action %q", req.Action)
    }
}
```

### Asset Actions via BuiltinActionHandler

Delegate to the builtin handler.  It reads and writes the `OverlayStore` which
is backed by the shared database:

```go
func (p *MyPlugin) handleAssetAction(ctx context.Context, entityName string, req plugin.ActionRequest) (*plugin.ActionResult, error) {
    handler := p.builtinActionHandler()
    if handler == nil {
        return nil, fmt.Errorf("overlay store not available")
    }

    switch req.Action {
    case "tag":
        return handler.HandleTag(ctx, entityName, req)
    case "annotate":
        return handler.HandleAnnotate(ctx, entityName, req)
    case "deprecate":
        return handler.HandleDeprecate(ctx, entityName, req)
    default:
        return nil, fmt.Errorf("unknown asset action %q", req.Action)
    }
}

func (p *MyPlugin) builtinActionHandler() *plugin.BuiltinActionHandler {
    if p.cfg.DB == nil {
        return nil
    }
    store := plugin.NewOverlayStore(p.cfg.DB)
    return plugin.NewBuiltinActionHandler(store, PluginName, "MyEntity")
}
```

### Action Dispatch Flow

```
POST /api/myplugin_catalog/v1alpha1/myentities/{name}:action
     |
     v
actionHandler (pkg/catalog/plugin/action_handler.go)
     |  parse ActionRequest from body
     |  verify action is declared in ListActions
     |  check dry-run support
     v
MyPlugin.HandleAction(scope=asset, targetID=name, req)
     |
     v
builtinActionHandler().HandleTag / HandleAnnotate / HandleDeprecate
     |
     v
OverlayStore.Upsert (persisted to shared DB)
```

### Adding Custom Actions

To add plugin-specific actions beyond the builtin set:

1. Add the action ID to `Entities[].Actions` in `GetCapabilitiesV2()`
2. Add an `ActionDefinition` to the top-level `Actions` list
3. Return it from `ListActions(ActionScopeAsset)`
4. Handle it in `handleAssetAction`

---

## Step 7: Add Configuration

Add a section for your plugin in `catalog/config/sources.yaml`:

```yaml
catalogs:
  # ... existing plugins ...

  myplugin:
    sources:
      - id: myplugin-default
        name: "My Entities"
        type: yaml
        enabled: true
        labels: ["My Entities"]
        properties:
          yamlCatalogPath: "../plugins/myplugin/data/entities.yaml"
```

The config key (`myplugin`) must match your plugin's `Name()` return value
unless you implement `SourceKeyProvider` to override it.

Place your YAML data file at `catalog/plugins/myplugin/data/entities.yaml`
with the content from Step 1.

### Config Resolution

```
sources.yaml                   Plugin
+--------------------------+   +----------------------------+
| catalogs:                |   | Name() = "myplugin"       |
|   myplugin:         <--------+ (or SourceKey() override) |
|     sources:             |   +----------------------------+
|       - id: myplugin-default
|         type: yaml       |
|         properties:      |
|           yamlCatalogPath|
+--------------------------+
         |
         v
   Init(ctx, Config{
       Section: CatalogSection{Sources: [...]},
       DB:      sharedDB,
       Logger:  slog.With("plugin", "myplugin"),
   })
```

---

## Step 8: Test with Conformance Suite

The conformance suite verifies that every registered plugin meets the framework
contract: capabilities discovery, entity list/get, action dispatch, and asset
mapping.

### Quick Start

Start the full stack:

```bash
docker compose -f docker-compose.catalog.yaml up --build -d
```

Run the conformance tests:

```bash
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1
```

### What the Suite Checks

```
Conformance Suite
+-- Plugin Discovery
|   +-- GET /api/plugins returns your plugin
|   +-- Plugin has name, version, description, basePath
|   +-- healthy = true
|
+-- Capabilities V2
|   +-- GET /api/plugins/{name}/capabilities returns document
|   +-- At least one entity with kind, plural, endpoints
|   +-- Columns and filter fields are defined
|
+-- Entity Endpoints
|   +-- GET {list endpoint} returns items array
|   +-- GET {get endpoint} returns single entity
|   +-- 404 for unknown entity name
|
+-- Actions
|   +-- POST {action endpoint} dispatches tag action
|   +-- Dry-run returns status "dry-run"
|   +-- Unknown action returns 400
|
+-- Asset Mapping
    +-- Entities project to valid AssetResource
    +-- Metadata.Name is non-empty
    +-- Kind matches entity kind from capabilities
```

### Manual Verification

After starting the stack, verify your plugin manually:

```bash
# Plugin appears in listing
curl -s http://localhost:8080/api/plugins | python3 -m json.tool

# V2 capabilities
curl -s http://localhost:8080/api/plugins/myplugin/capabilities | python3 -m json.tool

# List entities
curl -s http://localhost:8080/api/myplugin_catalog/v1alpha1/myentities | python3 -m json.tool

# Get single entity
curl -s http://localhost:8080/api/myplugin_catalog/v1alpha1/myentities/example-entity-1 | python3 -m json.tool

# Tag action (dry run)
curl -s -X POST \
  http://localhost:8080/api/myplugin_catalog/v1alpha1/myentities/example-entity-1:action \
  -H 'Content-Type: application/json' \
  -d '{"action":"tag","dryRun":true,"params":{"tags":["test"]}}' \
  | python3 -m json.tool
```

---

## Readiness Checklist

| Check | Description |
|-------|-------------|
| Plugin registered | Blank import in `cmd/catalog-server/main.go`, `init()` calls `plugin.Register()` |
| V2 capabilities | `GetCapabilitiesV2()` returns a complete document with entities, columns, filters, detail fields |
| Entity endpoints | List and Get return proper JSON responses with `items` array and single entity |
| Asset mapper | `MapToAsset` produces a valid `AssetResource` with `Kind`, `Metadata.Name`, `Spec`, `Status` |
| Actions | `HandleAction` dispatches builtin actions (tag, annotate, deprecate) and source refresh |
| Configuration | `sources.yaml` section with at least one enabled source and a valid `yamlCatalogPath` |
| Health | `Healthy()` returns `true` after `Init` completes successfully |
| Conformance | All conformance tests pass for your plugin |

---

## Key Files (Knowledge Plugin Reference)

The knowledge plugin is the simplest complete plugin.  Use it as a template.

| File | Purpose |
|------|---------|
| `catalog/plugins/knowledge/register.go` | `init()` registration (3 lines) |
| `catalog/plugins/knowledge/plugin.go` | `CatalogPlugin` implementation: entity struct, Init, routes, handlers, YAML loader |
| `catalog/plugins/knowledge/asset_mapper.go` | `AssetMapperProvider`: maps `KnowledgeSourceEntry` to `AssetResource` |
| `catalog/plugins/knowledge/actions.go` | `ActionProvider`: dispatches builtin tag/annotate/deprecate and source refresh |
| `catalog/plugins/knowledge/management.go` | `CapabilitiesV2Provider`, `SourceManager`, `RefreshProvider`, `DiagnosticsProvider` |
| `catalog/plugins/knowledge/data/sample-knowledge-sources.yaml` | Sample YAML data file |
| `pkg/catalog/plugin/plugin.go` | `CatalogPlugin` interface and all optional interfaces |
| `pkg/catalog/plugin/capabilities_types.go` | V2 capabilities types: `PluginCapabilitiesV2`, `EntityCapabilities`, `V2ColumnHint`, etc. |
| `pkg/catalog/plugin/asset_types.go` | `AssetResource`, `AssetMetadata`, `AssetStatus` types |
| `pkg/catalog/plugin/asset_mapper.go` | `AssetMapper` interface, `MapToAssetsBatch` helper, `DefaultAssetStatus()` |
| `pkg/catalog/plugin/builtin_actions.go` | `BuiltinActionHandler`, `BuiltinActionDefinitions()` |
| `pkg/catalog/plugin/action_types.go` | `ActionProvider` interface, `ActionRequest`, `ActionResult` |
| `cmd/catalog-server/main.go` | Server entry point with blank imports for all plugins |
| `catalog/config/sources.yaml` | Development configuration with all plugin sections |

---

[Back to Plugin Framework](./README.md) | [Prev: Architecture](./architecture.md) | [Next: Configuration](./configuration.md)
