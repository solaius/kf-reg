# Plugin Framework Architecture

## Overview

The catalog-of-catalogs system uses a **self-registering plugin architecture** where each asset type (models, MCP servers, knowledge sources, agents, etc.) is implemented as an independent plugin. Plugins register themselves via Go `init()` functions and are discovered, initialized, and served by a unified HTTP server.

**Location:** `pkg/catalog/plugin/`

## Core Interface

Every plugin must implement the `CatalogPlugin` interface:

```go
// pkg/catalog/plugin/plugin.go
type CatalogPlugin interface {
    Name() string                              // Plugin identity (e.g., "mcp", "agents")
    Version() string                           // API version (e.g., "v1alpha1")
    Description() string                       // Human-readable description
    Init(ctx context.Context, cfg Config) error // Initialize with configuration
    Start(ctx context.Context) error           // Begin background operations
    Stop(ctx context.Context) error            // Graceful shutdown
    Healthy() bool                             // Health check
    RegisterRoutes(router chi.Router) error    // Mount HTTP routes
    Migrations() []Migration                   // Database migrations
}
```

The `Config` struct passed during `Init` contains:

```go
type Config struct {
    Section     CatalogSection   // Plugin-specific config from sources.yaml
    DB          *gorm.DB         // Shared database connection
    Logger      *slog.Logger     // Namespaced logger
    BasePath    string           // API base path (e.g., "/api/mcp_catalog/v1alpha1")
    ConfigPaths []string         // Paths to sources.yaml files
}
```

## Optional Interfaces

Plugins can implement additional interfaces to opt into framework features:

| Interface | Purpose |
|-----------|---------|
| `BasePathProvider` | Custom API base path (default: `/api/{name}_catalog/{version}`) |
| `SourceKeyProvider` | Custom config key in sources.yaml (default: plugin name) |
| `CapabilitiesProvider` | V1 capability advertisement (entity kinds, list/get support) |
| `CapabilitiesV2Provider` | Full V2 capabilities discovery document |
| `StatusProvider` | Detailed status beyond `Healthy()` boolean |
| `SourceManager` | Runtime source CRUD (list, validate, apply, enable, delete) |
| `RefreshProvider` | On-demand data reload (per-source and all) |
| `DiagnosticsProvider` | Plugin health and per-source diagnostics |
| `UIHintsProvider` | Display hints for UI rendering |
| `CLIHintsProvider` | Display hints for CLI table rendering |
| `EntityGetter` | Generic entity retrieval by name |
| `AssetMapperProvider` | Project native entities to universal AssetResource |
| `AssetLister` | List entities as universal AssetResource items |
| `AssetGetter` | Get single entity as universal AssetResource |
| `ActionProvider` | Handle actions on entities and sources |

## Plugin Registry

Plugins register via a global singleton registry using blank imports:

```go
// catalog/plugins/mcp/register.go
package mcp

import "github.com/kubeflow/model-registry/pkg/catalog/plugin"

func init() {
    plugin.Register(&McpPlugin{})
}
```

The server discovers all registered plugins at startup:

```go
// pkg/catalog/plugin/registry.go
func Register(p CatalogPlugin)       // Add plugin to global registry
func All() []CatalogPlugin           // Return all registered plugins
func Get(name string) CatalogPlugin  // Find plugin by name
```

Plugins are activated by importing their registration package in the server entry point:

```go
// cmd/catalog-server/main.go
import (
    _ "github.com/.../catalog/plugins/model"
    _ "github.com/.../catalog/plugins/mcp"
    _ "github.com/.../catalog/plugins/knowledge"
    _ "github.com/.../catalog/plugins/agents"
    // ... all plugins
)
```

## Server Lifecycle

```
Register (init)     Global registry populated via blank imports
       │
       ▼
NewServer()         Create server with DB, config, options
       │
       ▼
Init()              For each registered plugin:
       │              1. Resolve config key (SourceKeyProvider or Name)
       │              2. Look up CatalogSection in sources.yaml
       │              3. Compute basePath (BasePathProvider or default)
       │              4. Call plugin.Init(ctx, cfg)
       │              5. On failure: record in failedPlugins, continue
       │
       ▼
MountRoutes()       Create chi.Router:
       │              1. Add middleware (RequestID, RealIP, CORS, Recovery)
       │              2. For each plugin: mount routes under basePath
       │              3. Mount management routes at {basePath}/management
       │              4. Add health endpoints (/healthz, /livez, /readyz)
       │              5. Add plugin info (/api/plugins, /api/plugins/{name}/capabilities)
       │
       ▼
Start()             Call plugin.Start(ctx) for each plugin
       │
       ▼
ReconcileLoop()     Every 30 seconds:
       │              1. Load config from ConfigStore
       │              2. Compare version hash
       │              3. If changed: update config, re-init affected plugins
       │
       ▼
Stop()              Call plugin.Stop(ctx) for each plugin
```

## Failure Isolation

A key design property: **one broken plugin does not crash the server**. During `Init()`, if a plugin fails:

1. The error is logged with the plugin name
2. The plugin is recorded in `failedPlugins` (not added to active `plugins`)
3. Initialization continues with remaining plugins
4. Failed plugins appear in `/api/plugins` with health status `false` and their error message

```go
if err := p.Init(ctx, pluginCfg); err != nil {
    s.logger.Error("plugin init failed, continuing with remaining plugins",
        "plugin", p.Name(), "error", err)
    s.failedPlugins = append(s.failedPlugins, failedPlugin{plugin: p, err: err})
    continue
}
```

## Route Mounting

Each plugin gets a sub-router scoped to its base path:

```
/api/model_catalog/v1alpha1/        # Model plugin routes
/api/model_catalog/v1alpha1/management/  # Model management routes
/api/mcp_catalog/v1alpha1/          # MCP plugin routes
/api/mcp_catalog/v1alpha1/management/    # MCP management routes
/api/agents_catalog/v1alpha1/       # Agents plugin routes
...
```

Management routes are automatically mounted when the plugin implements `SourceManager`, `RefreshProvider`, `DiagnosticsProvider`, or `ActionProvider`.

## Health Endpoints

| Endpoint | Method | Status Codes | Checks |
|----------|--------|-------------|--------|
| `/healthz` | GET | 200 | Always returns alive status with uptime |
| `/livez` | GET | 200 | Always returns alive status with uptime |
| `/readyz` | GET | 200 / 503 | Database connectivity, initial load completion, plugin health |

The `/readyz` response includes component-level status:

```json
{
  "status": "ready",
  "components": {
    "database": { "status": "up" },
    "initial_load": { "status": "complete" },
    "plugins": { "status": "healthy", "details": "all 8 plugins healthy" }
  }
}
```

## Plugin Info Endpoint

`GET /api/plugins` returns metadata about all registered plugins, including inline V2 capabilities:

```json
{
  "plugins": [
    {
      "name": "mcp",
      "version": "v1alpha1",
      "description": "MCP Server Catalog",
      "basePath": "/api/mcp_catalog/v1alpha1",
      "healthy": true,
      "entityKinds": ["McpServer"],
      "management": {
        "sourceManager": true,
        "refresh": true,
        "diagnostics": true,
        "actions": true
      },
      "capabilitiesV2": { ... }
    }
  ],
  "count": 8
}
```

`GET /api/plugins/{pluginName}/capabilities` returns the full V2 capabilities document for a single plugin.

## Config Reconciliation

When a `ConfigStore` is configured, the server runs a reconciliation loop every 30 seconds:

1. Load the current config and version hash from the store
2. Compare with the in-memory version hash
3. If different (external edit detected), update in-memory config and re-initialize all plugins with new configuration

This enables external tools or Kubernetes controllers to modify catalog sources without restarting the server.

## Tenancy Integration (Phase 8)

Phase 8 introduces multi-tenant support through a middleware stack that injects tenant context before plugin handlers execute. Plugins receive namespace context automatically without needing tenant-specific code.

### Middleware Stack

The catalog-server applies middleware in the following order for every incoming request:

```
Incoming HTTP Request
        |
        v
+-------------------+
|  CORS             |  Standard cross-origin headers
+-------------------+
        |
        v
+-------------------+
|  Tenancy          |  Resolve namespace from ?namespace= or X-Namespace header
|  (pkg/tenancy)    |  -> inject TenantContext into request context
+-------------------+
        |
        v
+-------------------+
|  Identity         |  Extract X-Remote-User and X-Remote-Group headers
|  (pkg/authz)      |  -> inject Identity into request context
+-------------------+
        |
        v
+-------------------+
|  Authorization    |  Map (method, path) -> (resource, verb)
|  (pkg/authz)      |  -> call Authorizer (SAR or noop)
|                   |  -> deny with 403 if unauthorized
+-------------------+
        |
        v
+-------------------+
|  Audit            |  Capture response status code after handler completes
|  (pkg/audit)      |  -> write AuditEventRecord with actor, namespace, outcome
+-------------------+
        |
        v
+-------------------+
|  Cache            |  For /api/plugins and /api/plugins/*/capabilities only
|  (pkg/cache)      |  -> serve from LRU cache on HIT
|                   |  -> capture response on MISS
+-------------------+
        |
        v
+-------------------+
|  Plugin Handler   |  Plugin-specific endpoint logic
|                   |  Can read tenancy.NamespaceFromContext(ctx)
|                   |  Can read authz.IdentityFromContext(ctx)
+-------------------+
```

### TenantContext Flow

The `TenantContext` (from `pkg/tenancy`) flows through the request context and is available to all downstream handlers:

```go
// In any handler or middleware:
tc, ok := tenancy.TenantFromContext(r.Context())
// tc.Namespace = "team-a"
// tc.User = "alice"
// tc.Groups = ["team-a-engineers"]

// Convenience:
ns := tenancy.NamespaceFromContext(r.Context())
// ns = "team-a"
```

Plugins that need namespace-aware queries use `tenancy.NamespaceFromContext(ctx)` to scope their database queries or in-memory filters.

### Plugin Namespace Awareness

Plugins receive namespace context implicitly through the request context. No plugin code changes are needed for namespace isolation -- the framework handles it:

1. **DB-backed plugins** (MCP, Knowledge) add `WHERE namespace = ?` to queries using the context namespace
2. **In-memory plugins** (Agents, Prompts, etc.) filter their in-memory stores by the context namespace
3. **Management handlers** scope source operations to the request namespace

### Authorization Integration

The authorization middleware maps every HTTP request to a `(resource, verb)` tuple and checks it against the configured `Authorizer`. This is transparent to plugins:

| URL Pattern | Resource | Verb |
|-------------|----------|------|
| `GET /api/plugins` | `plugins` | `list` |
| `GET /api/plugins/{name}/capabilities` | `capabilities` | `get` |
| `GET /{basePath}/{entities}` | `assets` | `list` |
| `GET /{basePath}/{entities}/{name}` | `assets` | `get` |
| `POST /{basePath}/management/apply-source` | `catalogsources` | `create` |
| `POST /{basePath}/management/refresh` | `jobs` | `create` |
| `POST *:action` | `actions` | `execute` |
| `DELETE /{basePath}/management/sources/{id}` | `catalogsources` | `delete` |

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/plugin.go` | CatalogPlugin interface and all optional interfaces |
| `pkg/catalog/plugin/registry.go` | Global plugin registry (Register, All, Get) |
| `pkg/catalog/plugin/server.go` | Server lifecycle, route mounting, health endpoints, reconcile loop |
| `cmd/catalog-server/main.go` | Server entry point with plugin imports and startup |
| `pkg/tenancy/` | Tenant context, middleware, resolvers (Phase 8) |
| `pkg/authz/` | Authorization: SAR, identity, caching, middleware, mapper (Phase 8) |
| `pkg/audit/` | Audit events: middleware, handlers, retention (Phase 8) |
| `pkg/jobs/` | Async refresh: job store, worker pool, handlers (Phase 8) |
| `pkg/cache/` | LRU caching: middleware, invalidation (Phase 8) |
| `pkg/ha/` | HA: migration lock, leader election (Phase 8) |

---

[Back to Plugin Framework](./README.md) | [Next: Creating Plugins](./creating-plugins.md)
