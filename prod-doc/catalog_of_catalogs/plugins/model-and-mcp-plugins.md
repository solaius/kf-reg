# Model and MCP Plugins

## Overview

The **Model** and **MCP** plugins are the two original plugins that proved the multi-plugin catalog architecture. Together they demonstrated that a single `catalog-server` process can host independent asset-type catalogs -- each with its own entity schema, source providers, REST endpoints, and management surface -- while sharing a database connection, configuration system, and plugin lifecycle.

Both plugins register themselves via Go `init()` functions and are activated by blank imports in `cmd/catalog-server/main.go`.

```
catalog-server process
  |
  +-- Model plugin (PluginName: "model")
  |     Entity kind: CatalogModel
  |     Source providers: yaml, hf (HuggingFace)
  |     Storage: GORM-backed CatalogModel table (PostgreSQL)
  |     BasePath: /api/model_catalog/v1alpha1
  |
  +-- MCP plugin (PluginName: "mcp")
        Entity kind: McpServer
        Source providers: yaml
        Storage: GORM-backed McpServer table (PostgreSQL)
        BasePath: /api/mcp_catalog/v1alpha1
```

Both plugins implement the full complement of optional plugin interfaces, making them first-class citizens of the capabilities-driven discovery and action framework introduced in Phase 5.

---

## Model Plugin

### Purpose

The model plugin is the **original catalog service** adapted to the plugin architecture. It wraps the existing catalog internals (`catalog/internal/catalog`, `catalog/internal/db`, `catalog/internal/server/openapi`) and exposes them through the `CatalogPlugin` interface without changing the underlying data model or REST contract.

**Location:** `catalog/plugins/model/`

### Config Key Compatibility

The model plugin implements `SourceKeyProvider` and returns `"models"` instead of the default (its plugin name `"model"`). This preserves backward compatibility with existing `sources.yaml` files that use the `models` key inside the `catalogs` map:

```yaml
# sources.yaml
catalogs:
  models:          # <-- SourceKey(), not the plugin Name()
    sources:
      - id: huggingface
        name: HuggingFace Models
        type: hf
        ...
```

### Initialization

During `Init()` the model plugin:

1. Creates a GORM service layer from the shared database connection using the `embedmd` connector with `SkipMigrations: true`.
2. Extracts typed repositories (`CatalogModelRepository`, `CatalogArtifactRepository`, `CatalogModelArtifactRepository`, `CatalogMetricsArtifactRepository`, `CatalogSourceRepository`, `PropertyOptionsRepository`) from the `RepoSet`.
3. Creates a `Loader` with no file paths -- sources are populated directly from the plugin config `Section.Sources`, converting them to the internal `catalog.Source` format and merging into the `SourceCollection`.
4. Handles label merging from `cfg.Section.Labels`.
5. Creates the `DBCatalog` API provider that backs all REST endpoints.

### Storage

The model plugin uses a full GORM-backed persistence layer. Entities are stored in the `CatalogModel` database table (plus related artifact and metrics tables). The schema is managed by the datastore migration system, not by the plugin's `Migrations()` method.

### Source Providers

| Type | Provider | Description |
|------|----------|-------------|
| `yaml` | Built-in YAML loader | Reads model definitions from local YAML files |
| `hf` | HuggingFace provider | Fetches model metadata from the HuggingFace Hub API |

### REST API

The model plugin has its own OpenAPI-generated REST API registered via `RegisterRoutes`. The `openapi.ModelCatalogServiceAPIService` and `openapi.ModelCatalogServiceAPIController` handle routing. Routes are stripped of the base path prefix before mounting on the chi sub-router.

### Implemented Interfaces

```
CatalogPlugin (core)
  |
  +-- SourceKeyProvider         SourceKey() -> "models"
  +-- CapabilitiesProvider      V1 capabilities (entity kinds, list/get/artifacts)
  +-- CapabilitiesV2Provider    Full V2 capabilities discovery document
  +-- StatusProvider            (via Healthy() bool)
  +-- SourceManager             ListSources, ValidateSource, ApplySource, EnableSource, DeleteSource
  +-- RefreshProvider           Refresh (per-source), RefreshAll
  +-- DiagnosticsProvider       Per-source health and state
  +-- EntityGetter              GetEntityByName (searches across all sources)
  +-- AssetMapperProvider       Maps CatalogModel to universal AssetResource
  +-- ActionProvider            tag, annotate, deprecate (asset), refresh (source)
  +-- UIHintsProvider           Column/detail field display hints
  +-- CLIHintsProvider          Default columns, sort field, filterable fields
```

### V2 Capabilities

The model plugin advertises the following through `GetCapabilitiesV2()`:

```
Plugin meta:
  Name:         model
  DisplayName:  Models
  Icon:         model

Entity: CatalogModel
  Plural:       models
  DisplayName:  Model
  Endpoints:
    List: /api/model_catalog/v1alpha1/models
    Get:  /api/model_catalog/v1alpha1/sources/{source_id}/models/{name}
  Detail sections: Overview, Details, Documentation
  Actions: tag, annotate, deprecate, refresh

Sources:
  Manageable:  true
  Refreshable: true
  Types:       yaml, hf
```

### Actions

| Action | Scope | Dry-Run | Idempotent | Description |
|--------|-------|---------|------------|-------------|
| `tag` | asset | Yes | Yes | Add or remove tags via `BuiltinActionHandler` and `OverlayStore` |
| `annotate` | asset | Yes | Yes | Add or update annotations via `BuiltinActionHandler` |
| `deprecate` | asset | Yes | Yes | Mark entity as deprecated via `BuiltinActionHandler` |
| `refresh` | source | No | Yes | Trigger `Loader.Reload()` for the given source |

Asset actions delegate to `plugin.NewBuiltinActionHandler(store, "model", "CatalogModel")` which uses the shared `OverlayStore` backed by the plugin's database connection.

---

## MCP Plugin

### Purpose

The MCP plugin catalogs **Model Context Protocol server entries** -- tools, resources, and prompts exposed by MCP-compliant servers. It was the first non-model plugin, proving that the plugin architecture could host fundamentally different entity types.

**Location:** `catalog/plugins/mcp/`

This plugin's code is partially generated by `catalog-gen` from `catalog.yaml` (the `plugin.go` and `register.go` files carry a generated header).

### Config Key

The MCP plugin does **not** implement `SourceKeyProvider`, so the framework uses its plugin name `"mcp"` as the config key:

```yaml
catalogs:
  mcp:             # <-- defaults to plugin Name()
    sources:
      - id: default-mcp-servers
        name: Default MCP Servers
        type: yaml
        ...
```

### Initialization

During `Init()` the MCP plugin:

1. Builds config paths from source properties (`loaderConfigPath`) or source origins, resolving relative paths and deduplicating.
2. Creates a GORM service layer from the shared database via the `embedmd` connector.
3. Extracts the `McpServerRepository` from the `RepoSet`.
4. Sets up a `ProviderRegistry` and registers the YAML provider (`providers.NewMcpServerYAMLProvider()`).
5. Creates a `Loader` with the resolved config paths and provider registry.

### Storage

The MCP plugin uses a GORM-backed `McpServer` table in the shared database. The repository supports `CountBySource` and `DeleteBySource` operations used by the management and diagnostics interfaces.

### McpServer Entity Schema

The `McpServer` entity carries rich metadata about MCP-compliant servers:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Internal unique identifier |
| `name` | string | Server display name |
| `externalId` | string | External identifier |
| `description` | string | Human-readable description |
| `serverUrl` | string | Server URL for connection |
| `transportType` | string | Transport protocol (stdio, sse, streamable-http) |
| `toolCount` | *int32 | Number of tools exposed |
| `resourceCount` | *int32 | Number of resources exposed |
| `promptCount` | *int32 | Number of prompts exposed |
| `deploymentMode` | string | Deployment mode (local, remote, hybrid) |
| `image` | string | Container image reference |
| `endpoint` | string | Remote endpoint URL |
| `supportedTransports` | string | Comma-separated supported transports |
| `license` | string | License identifier |
| `verified` | *bool | Whether the server is verified |
| `certified` | *bool | Whether the server is certified |
| `provider` | string | Provider or vendor name |
| `logo` | string | Logo URL or reference |
| `category` | string | Server category |
| `customProperties` | map | Arbitrary key-value metadata |

### Source Providers

| Type | Provider | Description |
|------|----------|-------------|
| `yaml` | `providers.NewMcpServerYAMLProvider()` | Reads MCP server definitions from YAML catalog files |

### REST API

The MCP plugin registers its own OpenAPI-generated API via `openapi.NewMcpServerCatalogServiceAPIService` and `openapi.NewDefaultAPIController`. Routes are mounted under `/api/mcp_catalog/v1alpha1` after stripping the base path prefix.

### Implemented Interfaces

```
CatalogPlugin (core)
  |
  +-- CapabilitiesProvider      V1 capabilities (entity kinds, list/get)
  +-- CapabilitiesV2Provider    Full V2 capabilities discovery document
  +-- SourceManager             ListSources, ValidateSource, ApplySource, EnableSource, DeleteSource
  +-- RefreshProvider           Refresh (per-source), RefreshAll
  +-- DiagnosticsProvider       Per-source health, entity counts
  +-- AssetMapperProvider       Maps McpServer to universal AssetResource
  +-- ActionProvider            tag, annotate, deprecate (asset), refresh (source)
  +-- UIHintsProvider           Column/detail field display hints
  +-- CLIHintsProvider          Default columns, sort field, filterable fields
```

### V2 Capabilities

```
Plugin meta:
  Name:         mcp
  DisplayName:  MCP Servers
  Icon:         server

Entity: McpServer
  Plural:       mcpservers
  DisplayName:  MCP Server
  Endpoints:
    List: /api/mcp_catalog/v1alpha1/mcpservers
    Get:  /api/mcp_catalog/v1alpha1/mcpservers/{name}
  Detail sections: Overview, Connection, Statistics
  Actions: tag, annotate, deprecate, refresh

Sources:
  Manageable:  true
  Refreshable: true
  Types:       yaml
```

### Management Interface

The MCP plugin provides a full source management surface:

```
ListSources        Iterates all sources in the SourceCollection.
   |               For YAML sources, reads file content into properties["content"].
   |               Queries McpServerRepository.CountBySource for entity counts.
   |
ValidateSource     Checks required fields (id, name, type).
   |               Verifies provider type is registered in the ProviderRegistry.
   |               Strict-decodes properties.content against mcpServerStrictEntry
   |               (KnownFields=true) to detect unknown YAML fields.
   |
ApplySource        If content + yamlCatalogPath both present, writes content to file.
   |               Strips inline content from properties (file is source of truth).
   |               Preserves origin from existing source for path resolution.
   |               Merges into SourceCollection.
   |
EnableSource       Toggles the Enabled flag on an existing source and re-merges.
   |
DeleteSource       Calls McpServerRepository.DeleteBySource to remove entities.
                   Disables the source to prevent re-loading.
```

The strict validation step deserializes YAML content using `yaml.Decoder` with `KnownFields(true)` against a struct that enumerates all valid MCP server fields (`mcpServerStrictEntry`). Any unrecognized field in the YAML produces a validation error.

### Actions

| Action | Scope | Dry-Run | Idempotent | Description |
|--------|-------|---------|------------|-------------|
| `tag` | asset | Yes | Yes | Add or remove tags via `BuiltinActionHandler` and `OverlayStore` |
| `annotate` | asset | Yes | Yes | Add or update annotations via `BuiltinActionHandler` |
| `deprecate` | asset | Yes | Yes | Mark entity as deprecated via `BuiltinActionHandler` |
| `refresh` | source | No | Yes | Trigger `Loader.Reload()` for the given source |

Asset actions delegate to `plugin.NewBuiltinActionHandler(store, "mcp", "McpServer")` which uses the shared `OverlayStore` backed by the plugin's database connection. The handler is created lazily on each action request.

---

## Key API Endpoints

### Model Plugin (`/api/model_catalog/v1alpha1`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/sources` | List configured model sources |
| GET | `/sources/{source_id}/models` | List models from a specific source |
| GET | `/sources/{source_id}/models/{model_name}` | Get a single model by source and name |
| GET | `/models` | List all models across sources |
| GET | `/models/{model_name}` | Get a single model by name |
| GET | `/models/{model_name}/artifacts` | List artifacts for a model |
| POST | `/management/sources` | Apply a source configuration |
| POST | `/management/validate-source` | Validate a source without applying |
| GET | `/management/diagnostics` | Plugin diagnostics |
| POST | `/management/refresh` | Refresh all sources |
| POST | `/management/sources/{id}:action` | Execute a source action (refresh) |
| POST | `/management/entities/{name}:action` | Execute an asset action (tag, annotate, deprecate) |

### MCP Plugin (`/api/mcp_catalog/v1alpha1`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mcpservers` | List all MCP server entries |
| GET | `/mcpservers/{name}` | Get a single MCP server by name |
| POST | `/management/sources` | Apply a source configuration |
| POST | `/management/validate-source` | Validate a source without applying |
| GET | `/management/sources` | List configured sources with status |
| PUT | `/management/sources/{id}/enable` | Enable or disable a source |
| DELETE | `/management/sources/{id}` | Delete a source and its entities |
| GET | `/management/diagnostics` | Plugin diagnostics |
| POST | `/management/refresh` | Refresh all sources |
| POST | `/management/sources/{id}:action` | Execute a source action (refresh) |
| POST | `/management/entities/{name}:action` | Execute an asset action (tag, annotate, deprecate) |

### Shared Endpoints (served by plugin framework)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/plugins` | List all registered plugins with metadata |
| GET | `/api/plugins/{name}/capabilities` | V2 capabilities for a specific plugin |
| GET | `/healthz` | Liveness check |
| GET | `/readyz` | Readiness check (DB + plugins) |

---

## Comparison

| Aspect | Model Plugin | MCP Plugin |
|--------|-------------|------------|
| Plugin name | `model` | `mcp` |
| Config key (SourceKey) | `models` (custom) | `mcp` (default) |
| Entity kind | `CatalogModel` | `McpServer` |
| Entity plural | `models` | `mcpservers` |
| Base path | `/api/model_catalog/v1alpha1` | `/api/mcp_catalog/v1alpha1` |
| Source types | `yaml`, `hf` | `yaml` |
| Artifacts support | Yes | No |
| EntityGetter | Yes (cross-source search) | No |
| Code generation | Manual | Partial (`catalog-gen`) |
| Detail sections | Overview, Details, Documentation | Overview, Connection, Statistics |
| Strict content validation | No | Yes (`KnownFields` YAML decoding) |

---

## Key Files

### Model Plugin

| File | Purpose |
|------|---------|
| `catalog/plugins/model/plugin.go` | Plugin struct, Init, Start, Stop, RegisterRoutes, GORM service setup |
| `catalog/plugins/model/register.go` | `init()` registration with global plugin registry |
| `catalog/plugins/model/management.go` | Capabilities, SourceManager, RefreshProvider, DiagnosticsProvider, UIHints, CLIHints, EntityGetter |
| `catalog/plugins/model/asset_mapper.go` | AssetMapperProvider mapping CatalogModel to AssetResource |
| `catalog/plugins/model/actions.go` | ActionProvider with tag, annotate, deprecate, refresh |

### MCP Plugin

| File | Purpose |
|------|---------|
| `catalog/plugins/mcp/plugin.go` | Plugin struct, Init, Start, Stop, RegisterRoutes (generated) |
| `catalog/plugins/mcp/register.go` | `init()` registration (generated) |
| `catalog/plugins/mcp/management.go` | Capabilities, SourceManager (with strict validation), RefreshProvider, DiagnosticsProvider, UIHints, CLIHints |
| `catalog/plugins/mcp/asset_mapper.go` | AssetMapperProvider mapping McpServer to AssetResource |
| `catalog/plugins/mcp/actions.go` | ActionProvider with tag, annotate, deprecate, refresh |

---

[Back to Plugins](./README.md) | [Next: Asset Type Plugins](./asset-type-plugins.md)
