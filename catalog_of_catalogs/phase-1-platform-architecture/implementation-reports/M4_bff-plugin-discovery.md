# M4: BFF Plugin Discovery Handlers

**Date**: 2026-02-15
**Status**: Complete
**Phase**: Phase 1: Platform Architecture

## Summary

This milestone adds a plugin discovery endpoint to the BFF (Backend for Frontend) layer, enabling the React frontend to discover all registered catalog plugins at runtime. The BFF proxies the catalog server's `GET /api/plugins` endpoint and exposes it at `GET /api/v1/model_catalog/plugins`, following the same middleware chain as other catalog routes.

## Motivation

- The frontend needs to know which catalog plugins are available (e.g., model catalog, MCP catalog) so it can build navigation, render plugin-specific views, and check capabilities dynamically.
- Without a discovery endpoint in the BFF, the frontend would need to hardcode plugin information or bypass the BFF entirely.
- **FR2** (Plugin discovery): Frontends can discover plugins and their capabilities via a standardized API.
- **AC6** (UI integration seam, partial): This provides the backend plumbing for frontend plugin discovery; the actual React components are a separate milestone.

## What Changed

### Files Created

| File | Purpose |
|------|---------|
| `clients/ui/bff/internal/api/catalog_plugins_handler.go` | HTTP handler for `GET /api/v1/model_catalog/plugins` |
| `clients/ui/bff/internal/api/catalog_plugins_handler_test.go` | Ginkgo/Gomega test for the handler |
| `clients/ui/bff/internal/models/catalog_plugin.go` | Go structs for `CatalogPlugin`, `CatalogPluginList`, `CatalogPluginCapabilities`, and `CatalogPluginStatus` |
| `clients/ui/bff/internal/repositories/catalog_plugins.go` | Repository layer that calls `GET /plugins` on the catalog server |

### Files Modified

| File | Change |
|------|--------|
| `clients/ui/bff/internal/api/app.go` | Added `CatalogPluginListPath` constant and registered the `GET` route with catalog middleware |
| `clients/ui/bff/internal/mocks/static_data_mock.go` | Added `GetCatalogPluginListMock()` returning two mock plugins (model, mcp) |
| `clients/ui/bff/internal/mocks/model_catalog_client_mock.go` | Added `GetAllCatalogPlugins` method to the mock client |
| `clients/ui/bff/internal/repositories/model_catalog_client.go` | Added `CatalogPluginsInterface` to the `ModelCatalogClientInterface` composite interface |

## How It Works

### Model Structs

The `CatalogPlugin` struct models the response from the catalog server's plugin endpoint, including capability flags and status information:

```go
type CatalogPlugin struct {
    Name         string                     `json:"name"`
    Version      string                     `json:"version"`
    Description  string                     `json:"description"`
    BasePath     string                     `json:"basePath"`
    Healthy      bool                       `json:"healthy"`
    EntityKinds  []string                   `json:"entityKinds,omitempty"`
    Capabilities *CatalogPluginCapabilities `json:"capabilities,omitempty"`
    Status       *CatalogPluginStatus       `json:"status,omitempty"`
}

type CatalogPluginList struct {
    Plugins []CatalogPlugin `json:"plugins"`
    Count   int             `json:"count"`
}
```

`CatalogPluginCapabilities` describes what operations a plugin supports:

```go
type CatalogPluginCapabilities struct {
    EntityKinds  []string `json:"entityKinds,omitempty"`
    ListEntities bool     `json:"listEntities"`
    GetEntity    bool     `json:"getEntity"`
    ListSources  bool     `json:"listSources"`
    Artifacts    bool     `json:"artifacts"`
}
```

### Repository Layer

The repository defines an interface and implementation that fetches plugins from the catalog server via the shared HTTP client:

```go
type CatalogPluginsInterface interface {
    GetAllCatalogPlugins(client httpclient.HTTPClientInterface) (*models.CatalogPluginList, error)
}

func (a CatalogPlugins) GetAllCatalogPlugins(client httpclient.HTTPClientInterface) (*models.CatalogPluginList, error) {
    responseData, err := client.GET(pluginsPath)
    if err != nil {
        return nil, fmt.Errorf("error fetching plugins: %w", err)
    }

    var pluginList models.CatalogPluginList
    if err := json.Unmarshal(responseData, &pluginList); err != nil {
        return nil, fmt.Errorf("error decoding response data: %w", err)
    }

    return &pluginList, nil
}
```

The `pluginsPath` constant is `/plugins`, which is appended to the catalog server's base URL by the HTTP client.

### Handler

The handler extracts the catalog HTTP client from the request context (injected by the `AttachModelCatalogRESTClient` middleware) and delegates to the repository:

```go
func (app *App) GetAllCatalogPluginsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
        return
    }

    catalogPlugins, err := app.repositories.ModelCatalogClient.GetAllCatalogPlugins(client)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    pluginsList := CatalogPluginListEnvelope{Data: catalogPlugins}
    err = app.WriteJSON(w, http.StatusOK, pluginsList, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### Route Registration

The route is registered in `app.go` alongside other catalog routes, using the same `AttachNamespace` and `AttachModelCatalogRESTClient` middleware:

```go
CatalogPluginListPath = ApiPathPrefix + "/model_catalog/plugins"

// In Routes():
apiRouter.GET(CatalogPluginListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAllCatalogPluginsHandler)))
```

### Interface Composition

The `CatalogPluginsInterface` is composed into the `ModelCatalogClientInterface`, which the mock and real clients both implement:

```go
type ModelCatalogClientInterface interface {
    CatalogSourcesInterface
    CatalogModelsInterface
    CatalogSourcePreviewInterface
    CatalogPluginsInterface
}
```

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Proxy through BFF rather than direct frontend-to-catalog calls | Keeps the catalog server internal; BFF handles auth context, namespace injection, and TLS | Direct calls from frontend (rejected -- exposes internal service, no auth middleware) |
| Reuse existing `AttachModelCatalogRESTClient` middleware | The plugin endpoint is on the same catalog server, so the same HTTP client and base URL apply | Separate HTTP client for plugin discovery (rejected -- unnecessary complexity) |
| Flat capability flags instead of nested feature map | Simple boolean fields are easy to consume in TypeScript and require no deserialization logic | Feature map with string keys (rejected -- harder to type-check in frontend) |
| Separate `CatalogPluginsInterface` composed into aggregate | Follows the existing pattern (e.g., `CatalogSourcesInterface`, `CatalogModelsInterface`) for clean separation of concerns | Single monolithic interface (rejected -- harder to test and mock) |

## Testing

- **Unit test**: `catalog_plugins_handler_test.go` uses Ginkgo/Gomega to test the full handler chain with mocked Kubernetes and catalog clients.
- **Test coverage**: Verifies HTTP 200 response, correct plugin count, and that each plugin's `Name`, `BasePath`, and `Healthy` fields match the mock data.
- **Mock data**: `GetCatalogPluginListMock()` returns two plugins (model and mcp) with full capabilities, matching the real server's response format.

```go
actual, rs, err := setupApiTest[CatalogPluginListEnvelope](
    http.MethodGet,
    "/api/v1/model_catalog/plugins?namespace=kubeflow",
    nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow",
)
Expect(rs.StatusCode).To(Equal(http.StatusOK))
Expect(actual.Data.Count).To(Equal(expected.Data.Count))
Expect(actual.Data.Plugins[0].Name).To(Equal(expected.Data.Plugins[0].Name))
```

To run:

```bash
cd clients/ui/bff && go test ./internal/api/ -run TestGetAllCatalogPluginsHandler
```

## Verification

```bash
# Start the BFF in mock mode
cd clients/ui/bff
make dev-bff

# Query the plugin discovery endpoint
curl -s http://localhost:4000/api/v1/model_catalog/plugins?namespace=kubeflow | jq .

# Expected: JSON envelope with data.plugins array containing "model" and "mcp"

# Run the unit test
cd clients/ui/bff && go test ./internal/api/ -v -run CatalogPlugins
```

## Dependencies & Impact

- **Upstream**: Depends on the catalog server's `GET /api/plugins` endpoint (implemented in M1 plugin framework).
- **Downstream**: Enables AC6 (frontend plugin UI) -- the React frontend can now call this endpoint to build dynamic navigation and plugin-specific views. Also enables M5 (developer documentation) to reference this endpoint as the integration seam.
- **Backward compatibility**: Additive change only. No existing routes or interfaces are modified.

## Open Items

- The endpoint does not yet support filtering by plugin health status or name.
- No caching of plugin metadata in the BFF; every request proxies through to the catalog server. This is acceptable at current scale but may need caching for high-traffic deployments.
- Frontend React components that consume this endpoint are not yet implemented (tracked separately).
