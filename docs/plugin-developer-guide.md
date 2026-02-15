# Plugin Developer Guide

This guide walks through creating a new catalog plugin using `catalog-gen`, the scaffolding tool for the unified catalog server. The MCP (Model Context Protocol) plugin is used as the reference example throughout.

## Prerequisites

- Go 1.24+
- `catalog-gen` binary (build with `go build ./cmd/catalog-gen`)
- Docker (for database and `make gen/gorm`)
- PostgreSQL or MySQL for local testing

## Architecture Overview

Each plugin in the catalog server:

1. Implements the `CatalogPlugin` interface defined in `pkg/catalog/plugin/plugin.go`
2. Registers itself via `init()` using `plugin.Register()`
3. Is imported (blank import) in `cmd/catalog-server/main.go`
4. Receives its configuration section from `sources.yaml` at init time
5. Mounts HTTP routes under its own base path (e.g., `/api/mcp_catalog/v1alpha1`)

The `catalog-gen` tool generates the boilerplate so you can focus on business logic.

## Step-by-Step Walkthrough

### 1. Scaffold the plugin

Run `catalog-gen init` from the `catalog/plugins/` directory:

```bash
cd catalog/plugins
catalog-gen init mcp \
  --entity=McpServer \
  --package=github.com/kubeflow/model-registry/catalog/plugins/mcp
```

This creates:

```
mcp/
  catalog.yaml              # Plugin configuration (editable)
  plugin.go                 # Plugin implementation (generated, regenerated on changes)
  register.go               # init() registration (generated)
  internal/
    db/models/              # Entity and artifact GORM models
    db/service/             # Repository and service layer
    catalog/providers/      # Data providers (YAML, HTTP)
    server/openapi/         # OpenAPI handlers
  api/openapi/              # OpenAPI specification
```

### 2. Edit catalog.yaml

The `catalog.yaml` file defines your entity schema, properties, providers, and API settings. Here is the MCP plugin's configuration:

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogConfig
metadata:
  name: mcp
spec:
  package: github.com/kubeflow/model-registry/catalog/plugins/mcp
  entity:
    name: McpServer
    properties:
      - name: serverUrl
        type: string
        required: true
        description: URL of the MCP server
      - name: transportType
        type: string
        description: "Transport protocol: stdio, http, sse"
      - name: toolCount
        type: integer
        description: Number of tools exposed
      - name: resourceCount
        type: integer
        description: Number of resources exposed
      - name: promptCount
        type: integer
        description: Number of prompts exposed
  providers:
    - type: yaml
  api:
    basePath: /api/mcp_catalog/v1alpha1
    port: 8081
```

Every entity automatically inherits these base fields (do not add them to properties):
`name`, `id`, `externalId`, `description`, `customProperties`, `createTimeSinceEpoch`, `lastUpdateTimeSinceEpoch`.

#### Property types

| YAML Type | Go Type    | OpenAPI Type           |
|-----------|------------|------------------------|
| string    | `*string`  | string                 |
| integer   | `*int32`   | integer                |
| int64     | `*int64`   | integer (format: int64)|
| boolean   | `*bool`    | boolean                |
| number    | `*float64` | number                 |
| array     | `[]string` | array                  |

### 3. Regenerate code

After editing `catalog.yaml`, regenerate the non-editable files:

```bash
cd catalog/plugins/mcp
catalog-gen generate
```

This regenerates:
- `plugin.go` and `register.go`
- Entity/artifact models in `internal/db/models/`
- Datastore spec and filter mappings in `internal/db/service/`
- OpenAPI spec in `api/openapi/`
- Loader in `internal/catalog/`

Files that are only created once (not overwritten):
- `internal/catalog/providers/yaml_provider.go` (editable)
- `internal/server/openapi/api_*_service_impl.go` (editable)

Then regenerate the OpenAPI server code:

```bash
make gen/openapi-server
```

### 4. Implement the service layer

Edit `internal/server/openapi/api_<entity>_service_impl.go` to implement the list and get endpoints. This file maps between the internal GORM models and the OpenAPI response types.

Key methods to implement:
- `ListMcpServers` - list entities with pagination and filtering
- `GetMcpServer` - get a single entity by name

See `catalog/plugins/mcp/internal/server/openapi/api_mcpserver_service_impl.go` for a working example.

### 5. Implement the YAML provider

Edit `internal/catalog/providers/yaml_provider.go` to parse your entity data from YAML files. The provider must implement:

```go
type Provider[T any, C any] interface {
    Load(ctx context.Context, src Source, config C) ([]T, error)
}
```

See `catalog/plugins/mcp/internal/catalog/providers/yaml_provider.go` for the MCP implementation.

### 6. Create test data

Create a `testdata/` directory with sample YAML data files:

```bash
mkdir -p testdata
```

Example `testdata/mcp-servers.yaml`:

```yaml
mcpservers:
  - name: "filesystem-server"
    description: "MCP server providing filesystem operations"
    serverUrl: "https://mcp.example.com/filesystem"
    transportType: "stdio"
    toolCount: 5
    resourceCount: 3
    promptCount: 2
```

And a sources config `testdata/test-mcp-sources.yaml`:

```yaml
apiVersion: catalog/v1alpha1
kind: CatalogSources
catalogs:
  mcp:
    sources:
      - id: test-mcp
        name: "Test MCP Servers"
        type: yaml
        properties:
          yamlCatalogPath: "./mcp-servers.yaml"
```

You can also generate sample test data automatically:

```bash
catalog-gen gen-testdata
```

### 7. Wire into catalog-server

In `cmd/catalog-server/main.go`, add a blank import for your plugin package:

```go
import (
    // Import plugins - their init() registers them
    _ "github.com/kubeflow/model-registry/catalog/plugins/model"
    _ "github.com/kubeflow/model-registry/catalog/plugins/mcp"
)
```

The plugin's `init()` function calls `plugin.Register()`, which adds the plugin to the global registry. The server discovers it at startup.

Add your plugin's sources to `sources.yaml`:

```yaml
apiVersion: catalog/v1alpha1
kind: CatalogSources
catalogs:
  models:
    sources:
      - id: my-models
        name: "My Models"
        type: yaml
        properties:
          yamlCatalogPath: "/data/models.yaml"
  mcp:
    sources:
      - id: my-mcp-servers
        name: "My MCP Servers"
        type: yaml
        properties:
          yamlCatalogPath: "/data/mcp-servers.yaml"
```

### 8. Build and test

Build the catalog server:

```bash
go build ./cmd/catalog-server
```

Run with a test database:

```bash
./catalog-server --sources=testdata/test-mcp-sources.yaml --db-type=postgres --db-dsn="host=localhost user=postgres dbname=catalog sslmode=disable"
```

Verify the plugin is registered:

```bash
curl http://localhost:8080/api/plugins
```

Query plugin entities:

```bash
curl http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers
```

## Key Files Reference

| File | Purpose | Editable? |
|------|---------|-----------|
| `catalog.yaml` | Plugin schema definition | Yes |
| `plugin.go` | Plugin lifecycle (init, start, stop, routes) | Regenerated |
| `register.go` | Auto-registration via init() | Regenerated |
| `internal/db/models/*.go` | GORM entity models | Regenerated |
| `internal/db/service/spec.go` | Datastore specification | Regenerated |
| `internal/db/service/filter_mappings.go` | FilterQuery field mapping | Regenerated |
| `internal/db/service/<entity>.go` | Service/repository layer | Yes |
| `internal/catalog/loader.go` | Source loading and hot-reload | Regenerated |
| `internal/catalog/providers/yaml_provider.go` | YAML data provider | Yes |
| `internal/server/openapi/api_*_service_impl.go` | API business logic | Yes |
| `api/openapi/` | OpenAPI specification | Regenerated |

## Adding Properties After Initial Scaffolding

1. Edit `catalog.yaml` to add the new property under `spec.entity.properties`
2. Run `catalog-gen generate` to regenerate models and specs
3. Run `make gen/openapi-server` to regenerate OpenAPI handlers
4. Update `internal/db/service/<entity>.go` if new property mapping is needed
5. Update `internal/server/openapi/api_*_service_impl.go` for the OpenAPI conversion
6. Update `internal/catalog/providers/yaml_provider.go` to parse the new field

## Adding Artifact Types

1. Run `catalog-gen add-artifact <Name>` (e.g., `catalog-gen add-artifact Tool`)
2. Run `catalog-gen generate` to generate artifact models and repositories
3. Run `make gen/openapi-server` to regenerate OpenAPI handlers
4. Implement the artifact list endpoint in `api_*_service_impl.go`
5. Update `plugin.go` to include the new artifact repository in `initServices`

## Filtering Support

All list endpoints automatically support `filterQuery` parameter with SQL-like syntax:

```
?filterQuery=transportType='http'
?filterQuery=toolCount>5 AND transportType='stdio'
?filterQuery=name LIKE '%server%'
```

Supported operators: `=`, `!=`, `>`, `<`, `>=`, `<=`, `LIKE`, `ILIKE`, `IN`, `AND`, `OR`.

Results can be ordered with `orderBy` and `sortOrder` parameters:

```
?orderBy=name&sortOrder=DESC
```
