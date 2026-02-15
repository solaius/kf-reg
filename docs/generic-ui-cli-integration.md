# Generic UI/CLI Integration Guide

This document explains how frontends and CLI tools can discover and interact with catalog plugins dynamically using the plugin discovery API.

## Plugin Discovery: GET /api/plugins

The catalog server exposes a `GET /api/plugins` endpoint that returns metadata about all registered plugins. This is the entry point for any client that needs to discover available catalog types.

### Response Format

```json
{
  "plugins": [
    {
      "name": "model",
      "version": "v1alpha1",
      "description": "Model catalog for ML models",
      "basePath": "/api/model_catalog/v1alpha1",
      "healthy": true,
      "entityKinds": ["CatalogModel"],
      "capabilities": {
        "entityKinds": ["CatalogModel"],
        "listEntities": true,
        "getEntity": true,
        "listSources": true,
        "artifacts": true
      },
      "status": {
        "enabled": true,
        "initialized": true,
        "serving": true
      }
    },
    {
      "name": "mcp",
      "version": "v1alpha1",
      "description": "McpServer catalog",
      "basePath": "/api/mcp_catalog/v1alpha1",
      "healthy": true,
      "entityKinds": ["McpServer"],
      "capabilities": {
        "entityKinds": ["McpServer"],
        "listEntities": true,
        "getEntity": true,
        "listSources": true,
        "artifacts": false
      }
    }
  ],
  "count": 2
}
```

### Metadata Fields

| Field | Description |
|-------|-------------|
| `name` | Plugin identifier (e.g., `model`, `mcp`) |
| `version` | API version string (e.g., `v1alpha1`) |
| `description` | Human-readable description |
| `basePath` | API base path for this plugin's endpoints |
| `healthy` | Whether the plugin is functioning correctly |
| `entityKinds` | List of entity types managed by this plugin |
| `capabilities` | Detailed capability flags (optional, requires plugin to implement `CapabilitiesProvider`) |
| `status` | Detailed status info (optional, requires plugin to implement `StatusProvider`) |

### Capability Flags

When a plugin implements `CapabilitiesProvider`, its capabilities indicate which API operations are available:

| Capability | Description |
|------------|-------------|
| `listEntities` | Plugin supports listing entities with pagination |
| `getEntity` | Plugin supports fetching a single entity by name |
| `listSources` | Plugin supports listing data sources |
| `artifacts` | Plugin supports artifact operations |
| `entityKinds` | Which entity types this plugin manages |

## BFF Proxy Layer

The BFF (Backend for Frontend) proxies the plugin discovery endpoint to the frontend via:

```
GET /api/v1/model_catalog/plugins
```

This follows the same middleware chain as other catalog endpoints (`AttachNamespace` and `AttachModelCatalogRESTClient`), ensuring proper namespace context and HTTP client setup.

### Handler: `GetAllCatalogPluginsHandler`

Location: `clients/ui/bff/internal/api/catalog_plugins_handler.go`

The handler retrieves the catalog HTTP client from the request context and delegates to the repository layer, which calls the catalog server's `/api/plugins` endpoint.

## Frontend Extension Points

To add a plugin-specific view to the frontend:

1. **Discover plugins at startup**: Call `GET /api/v1/model_catalog/plugins` when the app loads.

2. **Build navigation dynamically**: For each plugin in the response, create a navigation entry using the plugin's `name` and `description`.

3. **Construct API URLs**: Use the plugin's `basePath` to construct entity API calls. The standard patterns are:
   - List entities: `GET {basePath}/{entityKindPlural}`
   - Get entity: `GET {basePath}/{entityKindPlural}/{name}`
   - List sources: `GET {basePath}/sources`

4. **Check capabilities**: Use the `capabilities` field to determine which UI components to render. For example, only show an artifacts tab if `capabilities.artifacts` is `true`.

5. **Handle unhealthy plugins**: Check the `healthy` flag and optionally `status.lastError` to show degraded state in the UI.

### Example: Dynamic Plugin Cards

```typescript
// Fetch plugins
const response = await fetch('/api/v1/model_catalog/plugins?namespace=kubeflow');
const { data } = await response.json();

// Render a card for each plugin
data.plugins.forEach(plugin => {
  if (plugin.healthy) {
    renderPluginCard({
      title: plugin.description,
      basePath: plugin.basePath,
      entityKinds: plugin.entityKinds,
      hasArtifacts: plugin.capabilities?.artifacts ?? false,
    });
  }
});
```

## CLI Integration Pattern

A CLI tool can use plugin discovery to provide generic commands that work across all catalog types:

### 1. Discover available plugins

```bash
curl -s http://localhost:8080/api/plugins | jq '.plugins[] | {name, basePath, healthy}'
```

### 2. List entities from a specific plugin

Use the `basePath` from discovery to construct the entity list URL:

```bash
# Get the basePath for the MCP plugin
BASE=$(curl -s http://localhost:8080/api/plugins | jq -r '.plugins[] | select(.name=="mcp") | .basePath')

# List MCP servers
curl -s "http://localhost:8080${BASE}/mcpservers"
```

### 3. Generic entity listing

A CLI can iterate over all plugins and list all entities:

```bash
for plugin in $(curl -s http://localhost:8080/api/plugins | jq -c '.plugins[]'); do
  name=$(echo $plugin | jq -r '.name')
  basePath=$(echo $plugin | jq -r '.basePath')
  healthy=$(echo $plugin | jq -r '.healthy')

  if [ "$healthy" = "true" ]; then
    echo "=== $name ==="
    curl -s "http://localhost:8080${basePath}/sources" | jq '.items[].name'
  fi
done
```

### 4. Filtering with filterQuery

All plugin list endpoints support the `filterQuery` parameter:

```bash
# Find MCP servers using HTTP transport
curl -s "http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers?filterQuery=transportType='http'"

# Find models with a specific license
curl -s "http://localhost:8080/api/model_catalog/v1alpha1/models?filterQuery=license='Apache-2.0'"
```

## Health Monitoring

The `/api/plugins` endpoint also serves as a health check for individual plugins. A monitoring system can poll this endpoint and alert on:

- `healthy: false` for any plugin
- `status.lastError` containing error details
- `count` changing unexpectedly (plugin failed to load)

Failed plugins that could not initialize still appear in the response with `healthy: false` and `status.lastError` populated, so operators can diagnose issues without checking server logs.
