# BFF Integration

## Overview

The Backend for Frontend (BFF) acts as a **thin proxy layer** between the React frontend and the catalog-server. It forwards catalog API requests, resolves plugin base paths dynamically, wraps responses in a standard `Envelope` structure, and handles error propagation. The BFF does not contain catalog business logic; it delegates entirely to the catalog-server.

**Location:** `clients/ui/bff/internal/api/`

```
+-------------------+         +-------------------+         +-------------------+
|                   |  HTTP   |                   |  HTTP   |                   |
|  React Frontend   | ------> |       BFF         | ------> |  Catalog Server   |
|  (PatternFly)     |         |  (Go/httprouter)  |         |  (Go/chi)         |
|                   |         |                   |         |                   |
+-------------------+         +-------------------+         +-------------------+
       :9000                        :4000                         :8080

                              BFF responsibilities:
                              - Attach HTTP client via middleware
                              - Resolve plugin basePath from /api/plugins
                              - Wrap responses in Envelope{Data}
                              - Propagate HTTP errors (4xx, 5xx)
                              - Passthrough query params (pagination, filters)
```

## Request Flow

Every catalog request follows the same pattern through the BFF:

```
Frontend HTTP request
        |
        v
AttachModelCatalogRESTClient middleware
  - Resolve catalog-server base URL
  - Create HTTP client with TLS config
  - Inject client into request context
        |
        v
Handler function
  1. Extract client from context
  2. Extract path params (plugin_name, source_id, etc.)
  3. For management routes: ResolvePluginBasePath(pluginName)
     - GET /api/plugins -> find plugin -> return basePath
  4. Call repository method (builds upstream URL, sends request)
  5. Wrap result in Envelope{Data: ...}
  6. WriteJSON response
        |
        v
Frontend receives Envelope response
```

## Plugin Discovery

**Handler:** `GetAllCatalogPluginsHandler` in `catalog_plugins_handler.go`

The plugin discovery endpoint is a direct passthrough. The BFF fetches the plugin list from the catalog-server and wraps it in an envelope.

| BFF Path | Catalog-Server Path | Response Type |
|----------|---------------------|---------------|
| `GET /api/v1/model_catalog/plugins` | `GET /api/plugins` | `CatalogPluginListEnvelope` |

The repository layer (`catalog_plugins.go`) calls `GET /api/plugins` on the catalog-server and deserializes the response into a `CatalogPluginList` model. The plugin list includes each plugin's `basePath`, which is used by all subsequent management operations.

## Capabilities Proxy

**Handler:** `GetPluginCapabilitiesHandler` in `catalog_capabilities_handler.go`

The capabilities endpoint proxies the V2 capabilities document for a specific plugin. The response is forwarded as raw JSON (`json.RawMessage`) to avoid coupling the BFF to the capabilities schema.

| BFF Path | Catalog-Server Path | Response Type |
|----------|---------------------|---------------|
| `GET /api/v1/catalog/:plugin_name/capabilities` | `GET /api/plugins/{name}/capabilities` | `Envelope[json.RawMessage]` |

The repository layer (`catalog_entities.go`) constructs the upstream path using the format `/api/plugins/%s/capabilities` and returns the raw response bytes. The BFF wraps them in an envelope without deserialization.

## Entity Proxy

**Handlers:** `GetCatalogEntityListHandler`, `GetCatalogEntityHandler`, `PostCatalogEntityActionHandler`, `PostCatalogSourceActionHandler` in `catalog_entity_handler.go`

The generic entity proxy dynamically routes to any plugin's entity endpoints. Instead of maintaining per-plugin handler code, the BFF constructs the upstream URL using the plugin name convention.

### URL Construction

The repository layer (`catalog_entities.go`) uses format strings to build upstream paths:

```
Entity list:   /api/{plugin}_catalog/v1alpha1/{entityPlural}
Entity get:    /api/{plugin}_catalog/v1alpha1/{entityPlural}/{entityName}
Entity action: /api/{plugin}_catalog/v1alpha1/management/entities/{entityName}:action
Source action: /api/{plugin}_catalog/v1alpha1/management/sources/{sourceId}:action
```

### Query Parameter Passthrough

For entity list requests, all query parameters from the frontend are forwarded to the catalog-server. This includes pagination (`pageSize`, `nextPageToken`), filtering (`filterQuery`), and sorting (`orderBy`, `sortOrder`) parameters. The `UrlWithPageParams` helper appends these parameters to the constructed path.

### Response Handling

All entity responses are forwarded as `json.RawMessage` wrapped in an envelope. This allows the BFF to proxy any entity type without knowing its schema, which is essential for the generic UI components that discover entity shapes at runtime via capabilities.

| BFF Path | Catalog-Server Path | Method |
|----------|---------------------|--------|
| `/api/v1/catalog/:plugin/entities/:entityPlural` | `/api/{plugin}_catalog/v1alpha1/{entityPlural}` | GET |
| `/api/v1/catalog/:plugin/entities/:entityPlural/:entityName` | `/api/{plugin}_catalog/v1alpha1/{entityPlural}/{entityName}` | GET |
| `/api/v1/catalog/:plugin/entities/:entityPlural/:entityName/action` | `/api/{plugin}_catalog/v1alpha1/management/entities/{entityName}:action` | POST |
| `/api/v1/catalog/:plugin/sources/:sourceId/action` | `/api/{plugin}_catalog/v1alpha1/management/sources/{sourceId}:action` | POST |

## Management Proxy

**Handlers:** `catalog_management_handler.go`

The management proxy handles all source lifecycle and operational endpoints. Each handler follows the same pattern:

1. Extract the `plugin_name` path parameter
2. Call `ResolvePluginBasePath(pluginName)` to discover the plugin's upstream base path
3. Construct the full management endpoint URL by joining `basePath` with the management sub-path
4. Forward the request and wrap the response in a typed envelope

### Base Path Resolution

The `ResolvePluginBasePath` method fetches `GET /api/plugins` from the catalog-server, iterates the plugin list, and returns the `basePath` field for the matching plugin name. This is cached per-request (not globally) to ensure consistency with the current server state.

```
ResolvePluginBasePath("mcp")
    |
    v
GET /api/plugins
    |
    v
Find plugin where name == "mcp"
    |
    v
Return basePath: "/api/mcp_catalog/v1alpha1"
```

### Management Upstream Paths

All management paths are constructed by joining the resolved `basePath` with management sub-paths defined as constants in `catalog_management.go`:

```
mgmtPrefix          = "/management"
mgmtSourcesPath     = "/management/sources"
mgmtRefreshPath     = "/management/refresh"
mgmtDiagnosticsPath = "/management/diagnostics"
mgmtValidatePath    = "/management/validate-source"
mgmtApplyPath       = "/management/apply-source"
mgmtEnableSuffix    = "/enable"
mgmtRevisionsPath   = "/revisions"
```

### Source CRUD

| Operation | BFF Handler | Upstream Path |
|-----------|-------------|---------------|
| List sources | `GetPluginSourcesHandler` | `GET {basePath}/management/sources` |
| Apply source config | `ApplyPluginSourceConfigHandler` | `POST {basePath}/management/apply-source` |
| Enable/disable source | `EnablePluginSourceHandler` | `POST {basePath}/management/sources/{id}/enable` |
| Delete source | `DeletePluginSourceHandler` | `DELETE {basePath}/management/sources/{id}` |

### Validation

| Operation | BFF Handler | Upstream Path |
|-----------|-------------|---------------|
| Validate new config | `ValidatePluginSourceConfigHandler` | `POST {basePath}/management/validate-source` |
| Validate existing source | `ValidatePluginSourceHandler` | `POST {basePath}/management/sources/{id}:validate` |

### Refresh

| Operation | BFF Handler | Upstream Path |
|-----------|-------------|---------------|
| Refresh all sources | `RefreshPluginHandler` | `POST {basePath}/management/refresh` |
| Refresh single source | `RefreshPluginSourceHandler` | `POST {basePath}/management/refresh/{id}` |

### Revisions and Rollback

| Operation | BFF Handler | Upstream Path |
|-----------|-------------|---------------|
| List revisions | `GetPluginSourceRevisionsHandler` | `GET {basePath}/management/sources/{id}/revisions` |
| Rollback source | `RollbackPluginSourceHandler` | `POST {basePath}/management/sources/{id}:rollback` |

### Diagnostics

| Operation | BFF Handler | Upstream Path |
|-----------|-------------|---------------|
| Get diagnostics | `GetPluginDiagnosticsHandler` | `GET {basePath}/management/diagnostics` |

### Request Body Handling

Management POST handlers that accept request bodies (apply, validate, enable, rollback) unwrap the frontend's `{data: ...}` envelope before forwarding to the catalog-server:

```go
var requestBody struct {
    Data models.SourceConfigPayload `json:"data"`
}
json.NewDecoder(r.Body).Decode(&requestBody)
// Forward requestBody.Data to catalog-server
```

## BFF Route Table

The complete route table for catalog-related BFF endpoints, showing the mapping between BFF paths and catalog-server paths.

### Plugin Discovery and Capabilities

| Method | BFF Path | Proxied To | Namespace |
|--------|----------|------------|-----------|
| GET | `/api/v1/model_catalog/plugins` | `/api/plugins` | Optional |
| GET | `/api/v1/catalog/:plugin_name/capabilities` | `/api/plugins/{name}/capabilities` | Optional |

### Generic Entity Browsing

| Method | BFF Path | Proxied To | Namespace |
|--------|----------|------------|-----------|
| GET | `/api/v1/catalog/:plugin/entities/:entityPlural` | `/api/{plugin}_catalog/v1alpha1/{entityPlural}` | Optional |
| GET | `/api/v1/catalog/:plugin/entities/:entityPlural/:entityName` | `/api/{plugin}_catalog/v1alpha1/{entityPlural}/{entityName}` | Optional |
| POST | `/api/v1/catalog/:plugin/entities/:entityPlural/:entityName/action` | `/api/{plugin}_catalog/v1alpha1/management/entities/{entityName}:action` | Optional |
| POST | `/api/v1/catalog/:plugin/sources/:sourceId/action` | `/api/{plugin}_catalog/v1alpha1/management/sources/{sourceId}:action` | Optional |

### Plugin Management

| Method | BFF Path | Proxied To | Namespace |
|--------|----------|------------|-----------|
| GET | `/api/v1/catalog/:plugin/sources` | `{basePath}/management/sources` | Required |
| POST | `/api/v1/catalog/:plugin/validate-source` | `{basePath}/management/validate-source` | Required |
| POST | `/api/v1/catalog/:plugin/apply-source` | `{basePath}/management/apply-source` | Required |
| POST | `/api/v1/catalog/:plugin/sources/:id/enable` | `{basePath}/management/sources/{id}/enable` | Required |
| DELETE | `/api/v1/catalog/:plugin/sources/:id` | `{basePath}/management/sources/{id}` | Required |
| POST | `/api/v1/catalog/:plugin/refresh` | `{basePath}/management/refresh` | Required |
| POST | `/api/v1/catalog/:plugin/refresh/:id` | `{basePath}/management/refresh/{id}` | Required |
| GET | `/api/v1/catalog/:plugin/diagnostics` | `{basePath}/management/diagnostics` | Required |
| POST | `/api/v1/catalog/:plugin/sources/:id/validate` | `{basePath}/management/sources/{id}:validate` | Required |
| GET | `/api/v1/catalog/:plugin/sources/:id/revisions` | `{basePath}/management/sources/{id}/revisions` | Required |
| POST | `/api/v1/catalog/:plugin/sources/:id/rollback` | `{basePath}/management/sources/{id}:rollback` | Required |

### Legacy Model Catalog Browsing

| Method | BFF Path | Proxied To | Namespace |
|--------|----------|------------|-----------|
| GET | `/api/v1/model_catalog/models` | `/api/model_catalog/v1alpha1/models` | Required |
| GET | `/api/v1/model_catalog/sources` | `/api/model_catalog/v1alpha1/sources` | Required |
| GET | `/api/v1/model_catalog/models/filter_options` | `/api/model_catalog/v1alpha1/models/filter_options` | Required |
| GET | `/api/v1/model_catalog/sources/:id/models/*name` | `/api/model_catalog/v1alpha1/sources/{id}/models/{name}` | Required |
| GET | `/api/v1/mcp_catalog/mcpservers` | `{basePath}/mcpservers` | Required |
| GET | `/api/v1/mcp_catalog/mcpservers/:name` | `{basePath}/mcpservers/{name}` | Required |

## Configuration

### CATALOG_SERVER_BASE_URL

The primary configuration for the BFF is the catalog-server base URL. This can be set via:

- **Environment variable:** `CATALOG_SERVER_BASE_URL`
- **CLI flag:** `--catalog-server-url`

When set, the BFF sends all catalog requests directly to this URL. When not set, the BFF falls back to Kubernetes service discovery, resolving the catalog-server address via the cluster's service registry.

```go
// clients/ui/bff/internal/api/middleware.go
if app.config.CatalogServerURL != "" {
    modelCatalogBaseURL = app.config.CatalogServerURL
} else {
    // Fall back to Kubernetes service discovery
    // ...
}
```

### Middleware Chain

Each catalog route is wrapped in a middleware chain that varies by route category:

| Route Category | Middleware Chain |
|---------------|-----------------|
| Generic entity/capabilities | `AttachOptionalNamespace` -> `AttachModelCatalogRESTClient` -> Handler |
| Plugin management | `AttachNamespace` -> `AttachModelCatalogRESTClient` -> Handler |
| Legacy model catalog | `AttachNamespace` -> `AttachModelCatalogRESTClient` -> Handler |
| Plugin list | `AttachOptionalNamespace` -> `AttachModelCatalogRESTClient` -> Handler |

The `AttachOptionalNamespace` middleware allows global catalog browsing without requiring a namespace header, while `AttachNamespace` enforces that a namespace is present (used for management operations that may need RBAC scoping).

### Health Check

The BFF exposes a `/healthcheck` endpoint that validates configuration readiness. It checks that the BFF is properly configured but does not verify connectivity to the catalog-server.

### TLS Configuration

When `BundlePaths` are configured, the BFF loads CA bundles for secure communication with the catalog-server. The `rootCAs` pool is attached to the HTTP client created by the `AttachModelCatalogRESTClient` middleware.

## Error Handling

All handlers follow a consistent error handling pattern. Errors from the catalog-server HTTP client are checked for `HTTPError` type to preserve the original status code:

```go
if err != nil {
    var httpErr *httpclient.HTTPError
    if errors.As(err, &httpErr) {
        app.errorResponse(w, r, httpErr)    // Preserve upstream status code
    } else {
        app.serverErrorResponse(w, r, err)  // 500 Internal Server Error
    }
    return
}
```

This ensures that 404, 400, 429, and other catalog-server error codes are forwarded to the frontend rather than being flattened to 500.

## Envelope Pattern

All BFF responses use a generic `Envelope` wrapper:

```go
type Envelope[T any, M any] struct {
    Data T `json:"data"`
    Meta M `json:"meta,omitempty"`
}
```

For catalog endpoints, the `Meta` type is typically `None` (an empty struct). The `Data` field contains the actual response payload. Typed envelope aliases are defined per handler file:

```
CatalogPluginListEnvelope    = Envelope[*models.CatalogPluginList, None]
SourceInfoListEnvelope       = Envelope[*models.SourceInfoList, None]
RefreshResultEnvelope        = Envelope[*models.RefreshResult, None]
ValidationResultEnvelope     = Envelope[*models.ValidationResult, None]
...
```

For generic entity and capabilities endpoints, responses use `Envelope[json.RawMessage, None]` to avoid deserializing plugin-specific payloads.

## Key Files

| File | Purpose |
|------|---------|
| `clients/ui/bff/internal/api/app.go` | Route constants, route registration, middleware wiring |
| `clients/ui/bff/internal/api/catalog_plugins_handler.go` | Plugin discovery handler (`GetAllCatalogPluginsHandler`) |
| `clients/ui/bff/internal/api/catalog_capabilities_handler.go` | V2 capabilities proxy (`GetPluginCapabilitiesHandler`) |
| `clients/ui/bff/internal/api/catalog_entity_handler.go` | Generic entity list/get/action handlers |
| `clients/ui/bff/internal/api/catalog_management_handler.go` | Management proxy (sources, validate, refresh, rollback, diagnostics) |
| `clients/ui/bff/internal/api/catalog_models_handler.go` | Legacy model catalog list handler |
| `clients/ui/bff/internal/api/catalog_sources_handler.go` | Legacy source/model browsing handlers |
| `clients/ui/bff/internal/api/catalog_filters_handler.go` | Model filter options handler |
| `clients/ui/bff/internal/api/catalog_source_preview_handler.go` | Source preview handler (settings page) |
| `clients/ui/bff/internal/api/mcp_catalog_handler.go` | MCP server list/get browsing handlers |
| `clients/ui/bff/internal/api/middleware.go` | `AttachModelCatalogRESTClient` middleware (URL resolution, HTTP client creation) |
| `clients/ui/bff/internal/repositories/catalog_plugins.go` | Plugin list repository (`GET /api/plugins`) |
| `clients/ui/bff/internal/repositories/catalog_management.go` | Management repository (sources, refresh, diagnostics, revisions, rollback) |
| `clients/ui/bff/internal/repositories/catalog_entities.go` | Generic entity repository (capabilities, entity list/get, actions) |
| `clients/ui/bff/cmd/main.go` | BFF entry point, `CATALOG_SERVER_BASE_URL` flag parsing |

---

[Back to Clients](./README.md) | [Next: Generic UI](./generic-ui.md)
