# M2: MCP Plugin End-to-End

**Date**: 2026-02-15
**Status**: Complete
**Phase**: Phase 1: Platform Architecture

## Summary

This milestone delivers a complete MCP (Model Context Protocol) server catalog plugin that runs alongside the existing model catalog inside the unified catalog server. It demonstrates the full plugin lifecycle -- YAML ingestion, MLMD-backed persistence, and REST API serving -- proving that the plugin framework can host multiple independent catalog types in a single process. The MCP plugin was scaffolded with `catalog-gen` and required minimal hand-written code.

## Motivation

- The catalog server needed a second plugin to validate the multi-plugin architecture beyond the original model catalog.
- MCP servers are a natural catalog entity: they have discoverable metadata (URL, transport type, tool/resource/prompt counts) and benefit from centralized registry and querying.
- Satisfies **FR1** (Multi-plugin hosting) -- the catalog server now starts with both model and MCP plugins.
- Satisfies **FR6** (Ingestion & persistence) -- YAML sources are ingested and stored in the shared MLMD database.
- Satisfies **FR7** (Query consistency) -- list and get endpoints support pagination, `filterQuery`, `orderBy`, and `sortOrder`.
- Satisfies **AC1** (catalog-server starts with model + MCP).
- Satisfies **AC2** (MCP plugin ingests YAML data and serves list/get).

## What Changed

### Files Created

| File | Purpose |
|------|---------|
| `catalog/plugins/mcp/catalog.yaml` | Entity schema definition for `McpServer` (properties, providers, API config) |
| `catalog/plugins/mcp/plugin.go` | Plugin lifecycle: Init, Start, Stop, route registration, service wiring |
| `catalog/plugins/mcp/register.go` | `init()` function that registers the plugin with the global registry |
| `catalog/plugins/mcp/internal/db/models/mcpserver.go` | `McpServer` entity interface, `McpServerImpl` struct, `McpServerRepository` interface, list options |
| `catalog/plugins/mcp/internal/db/service/mcpserver.go` | Repository implementation using `GenericRepository`, entity/schema mapping functions |
| `catalog/plugins/mcp/internal/db/service/spec.go` | `DatastoreSpec()` defining MLMD type `kf.McpServer` and its properties; `Services` aggregate |
| `catalog/plugins/mcp/internal/db/service/filter_mappings.go` | `filterQuery` property registration and `EntityMappingFunctions` implementation |
| `catalog/plugins/mcp/internal/catalog/loader.go` | Typed `Loader` wrapping the generic `catalog.Loader[McpServer, any]` |
| `catalog/plugins/mcp/internal/catalog/providers/yaml_provider.go` | YAML parser converting catalog files to `McpServer` records |
| `catalog/plugins/mcp/internal/server/openapi/api.go` | `DefaultAPIRouter` and `DefaultAPIServicer` interfaces |
| `catalog/plugins/mcp/internal/server/openapi/api_default_controller.go` | HTTP controller with route definitions and request parsing |
| `catalog/plugins/mcp/internal/server/openapi/api_mcpserver_service_impl.go` | Service implementation for `ListMcpServers` and `GetMcpServer` |
| `catalog/plugins/mcp/internal/server/openapi/api_mcpserver_service_impl_test.go` | Unit tests for the OpenAPI model converter |
| `catalog/plugins/mcp/internal/server/openapi/error.go` | Error types (`ParsingError`, `RequiredError`) and `DefaultErrorHandler` |
| `catalog/plugins/mcp/internal/server/openapi/helpers.go` | JSON response encoding, query parsing, `parseInt32` helper |
| `catalog/plugins/mcp/internal/server/openapi/impl.go` | `ImplResponse` struct |
| `catalog/plugins/mcp/internal/server/openapi/models.go` | `McpServer` and `McpServerList` OpenAPI model structs |
| `catalog/plugins/mcp/internal/server/openapi/routers.go` | `Route`, `Routes`, and `Router` types |
| `catalog/plugins/mcp/testdata/mcp-servers.yaml` | Sample MCP server catalog with three entries |
| `catalog/plugins/mcp/testdata/test-mcp-sources.yaml` | Test sources config pointing to the sample catalog |

### Files Modified

| File | Change |
|------|--------|
| `cmd/catalog-server/main.go` | Added blank import `_ "github.com/kubeflow/model-registry/catalog/plugins/mcp"` to wire the plugin |

## How It Works

### Entity Schema (catalog.yaml)

The plugin's schema is declared in `catalog.yaml`. It defines the entity name, typed properties, supported providers, and API base path. The `catalog-gen` tool reads this file and generates most of the boilerplate code.

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogConfig
metadata:
  name: mcp
spec:
  entity:
    name: McpServer
    properties:
      - name: serverUrl
        type: string
        required: true
      - name: transportType
        type: string
      - name: toolCount
        type: integer
      - name: resourceCount
        type: integer
      - name: promptCount
        type: integer
  providers:
    - type: yaml
  api:
    basePath: /api/mcp_catalog/v1alpha1
```

### Plugin Registration and Wiring

The plugin registers itself via `init()` in `register.go`. The catalog server imports the package with a blank import, triggering registration before `main()` runs.

```go
// register.go
func init() {
    plugin.Register(&McpServerCatalogPlugin{})
}

// cmd/catalog-server/main.go
import (
    _ "github.com/kubeflow/model-registry/catalog/plugins/model"
    _ "github.com/kubeflow/model-registry/catalog/plugins/mcp"
)
```

### Plugin Lifecycle

`McpServerCatalogPlugin` implements the full `CatalogPlugin` interface plus `BasePathProvider`. During `Init()` it creates the MLMD-backed repository, registers the YAML provider, and populates sources from the plugin config.

```go
func (p *McpServerCatalogPlugin) Init(ctx context.Context, cfg plugin.Config) error {
    // 1. Initialize services (repository) from the shared DB connection
    services, err := p.initServices(cfg.DB)
    // 2. Register the YAML provider
    registry := pkgcatalog.NewProviderRegistry[models.McpServer, any]()
    registry.Register("yaml", providers.NewMcpServerYAMLProvider())
    // 3. Create the loader and populate sources from plugin config
    p.loader = catalog.NewLoader(services, nil, registry)
    // 4. Merge sources into the loader's SourceCollection
    p.loader.Sources.Merge(origin, sources)
    return nil
}
```

`Start()` triggers the loader which reads YAML files, parses MCP server entries, and upserts them into the database.

### YAML Ingestion

The YAML provider parses catalog files structured as:

```yaml
mcpservers:
  - name: "filesystem-server"
    serverUrl: "https://mcp.example.com/filesystem"
    transportType: "stdio"
    toolCount: 5
    resourceCount: 3
    promptCount: 2
```

Each entry is converted to a `McpServer` entity with typed properties stored as `ContextProperty` rows in the MLMD schema. The provider uses the shared `yamlprovider.NewProvider` which includes automatic hot-reload via file watching.

### Database Layer

The repository uses `GenericRepository` from the shared infrastructure, parameterized with `McpServer`-specific types. The MLMD type is `kf.McpServer`.

```go
func DatastoreSpec() *datastore.Spec {
    return datastore.NewSpec().
        AddContext(McpServerTypeName, datastore.NewSpecType(NewMcpServerRepository).
            AddString("source_id").
            AddString("serverUrl").
            AddString("transportType").
            AddInt("toolCount").
            AddInt("resourceCount").
            AddInt("promptCount"),
        )
}
```

The `McpServerRepository` interface provides standard CRUD plus source-aware operations:

```go
type McpServerRepository interface {
    GetByID(id int32) (McpServer, error)
    GetByName(name string) (McpServer, error)
    List(options McpServerListOptions) (*models.ListWrapper[McpServer], error)
    Save(entity McpServer) (McpServer, error)
    DeleteBySource(sourceID string) error
    DeleteByID(id int32) error
    GetDistinctSourceIDs() ([]string, error)
}
```

### REST API

The plugin exposes two endpoints under `/api/mcp_catalog/v1alpha1`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mcpservers` | List MCP servers (supports `pageSize`, `pageToken`, `filterQuery`, `orderBy`, `sortOrder`) |
| GET | `/mcpservers/{name}` | Get a single MCP server by name |

The `DefaultAPIServicer` interface defines the service contract:

```go
type DefaultAPIServicer interface {
    ListMcpServers(ctx context.Context, pageSize int32, pageToken string,
        q string, filterQuery string, orderBy string, sortOrder string) (ImplResponse, error)
    GetMcpServer(ctx context.Context, name string) (ImplResponse, error)
}
```

The `filterQuery` parameter supports SQL-like syntax (e.g., `transportType='stdio' AND toolCount>3`) via the filter mappings registered in `filter_mappings.go`.

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Generate code with `catalog-gen` from `catalog.yaml` | Ensures consistency across plugins; reduces boilerplate; schema changes propagate automatically | Hand-write all code (more flexible but error-prone and inconsistent) |
| Store properties as MLMD `ContextProperty` rows | Reuses the existing MLMD schema and shared `GenericRepository` infrastructure | Separate SQL tables per plugin (more normalized but duplicates infrastructure) |
| Blank import for plugin registration | Standard Go pattern; no explicit wiring code; adding a plugin is a one-line import | Explicit registration in `main()` (more visible but more coupling) |
| Delegate to `yamlprovider.NewProvider` for hot-reload | Reuses the shared YAML provider with file-watching built in | Custom file watcher per plugin (unnecessary duplication) |
| `Save()` does upsert by name | Idempotent ingestion -- re-running the loader updates existing entries rather than creating duplicates | Insert-only with manual dedup (fragile) |

## Testing

Two categories of tests were added:

**OpenAPI model conversion tests** (`api_mcpserver_service_impl_test.go`):
- `TestConvertToOpenAPIModel` -- Full entity with all properties and custom properties; verifies correct mapping.
- `TestConvertToOpenAPIModelMinimal` -- Entity with only a name; verifies nil/zero handling.

**Plugin framework tests** (in `pkg/catalog/plugin/server_test.go`, covered in M1):
- `TestServerInitializesUnconfiguredPlugins` -- Verifies plugins initialize even without config entries.

Run with:
```bash
# MCP plugin tests
go test ./catalog/plugins/mcp/internal/server/openapi/ -v

# Plugin framework tests
go test ./pkg/catalog/plugin/ -v
```

## Verification

```bash
# 1. Build the catalog server
go build -o catalog-server ./cmd/catalog-server

# 2. Start with a sources config that includes MCP entries
#    (see catalog/plugins/mcp/testdata/test-mcp-sources.yaml for format)
./catalog-server -sources /path/to/sources.yaml -db-type postgres -db-dsn "host=localhost ..."

# 3. Verify both plugins are registered
curl -s http://localhost:8080/api/plugins | jq '.plugins[].name'
# Expected output: "model" and "mcp"

# 4. List MCP servers
curl -s http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers | jq .

# 5. Get a specific MCP server by name
curl -s http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers/filesystem-server | jq .

# 6. Filter MCP servers
curl -s "http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers?filterQuery=transportType='stdio'" | jq .

# 7. Verify readiness includes both plugins
curl -s http://localhost:8080/readyz | jq .
```

## Dependencies & Impact

- **Depends on**: M1 (Plugin Framework Hardening) for failure isolation and enhanced endpoints; the shared `GenericRepository`, `catalog.Loader`, and `yamlprovider` infrastructure; the `catalog-gen` code generator.
- **Enables**: Future plugins can follow the same pattern (define `catalog.yaml`, run `catalog-gen`, add a blank import). The MCP plugin also validates the ingestion pipeline for non-model entity types.
- **Backward compatibility**: The MCP plugin is additive. Existing model catalog functionality is unchanged. The MCP plugin routes are mounted under a separate base path (`/api/mcp_catalog/v1alpha1`).

## Open Items

- The MCP plugin currently only supports the YAML provider. A future milestone could add a live MCP discovery provider that probes running MCP servers for their capabilities.
- Artifact support is stubbed out (`saveArtifact` and `deleteArtifactsByEntity` are no-ops). MCP server artifacts (e.g., tool schemas, prompt templates) could be added via `catalog.yaml` and `catalog-gen`.
- The OpenAPI spec for the MCP plugin endpoints is not yet published as a standalone YAML file alongside the model registry spec.
