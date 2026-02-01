# Files Changed in MCP Catalog Implementation

This document provides a comprehensive inventory of all files added or modified for the MCP Catalog feature.

## Commit Summary

| Commit | Description | Files | Lines |
|--------|-------------|-------|-------|
| `21a39286` | Main MCP implementation (Phases 1-3) | 99 | +13,931 / -2,471 |
| `8d7b5096` | Free-form keyword search | 5 | +180 / -30 |
| `08c62fae` | Add MCP schemas to source OpenAPI | 3 | +450 / -20 |
| `933b2bd9` | Regenerate OpenAPI client SDK | 12 | +2,100 / -200 |

## Files by Category

### OpenAPI Specifications

```
api/openapi/catalog.yaml                          # MCP server schemas added
api/openapi/src/catalog.yaml                      # Source OpenAPI file
```

**Key additions:**
- `McpServer` schema
- `McpTool` schema
- `McpToolParameter` schema
- `McpEndpoints` schema
- `McpSecurityIndicator` schema
- `McpArtifact` schema
- `McpTransportType` enum
- `McpToolAccessType` enum
- `McpDeploymentMode` enum

### Catalog Service Backend

#### Core Implementation

```
catalog/internal/mcp/db_mcp_catalog.go            # Database-backed MCP catalog
catalog/internal/mcp/yaml_mcp_catalog.go          # YAML file provider
catalog/internal/catalog/mcp_loader.go            # MCP server loader
catalog/internal/catalog/mcp_source_merge.go      # Source merging logic
catalog/internal/catalog/mcp_server_filter.go     # Server inclusion/exclusion
```

#### Database Models

```
catalog/internal/db/models/mcp_server.go          # McpServer entity
catalog/internal/db/models/mcp_server_tool.go     # McpServerTool entity (future)
catalog/internal/db/service/mcp_server_repo.go    # Repository implementation
```

#### Filter System

```
catalog/internal/db/filter/mcp_entity_mappings.go # MCP field mappings
```

#### Common Utilities

```
catalog/internal/common/asset_detector.go         # YAML asset type detection
```

### Generated Code

```
catalog/pkg/openapi/model_mcp_server.go           # Generated MCP server model
catalog/pkg/openapi/model_mcp_tool.go             # Generated MCP tool model
catalog/pkg/openapi/model_mcp_endpoints.go        # Generated endpoints model
catalog/pkg/openapi/model_mcp_security_indicator.go
catalog/pkg/openapi/model_mcp_transport_type.go
catalog/pkg/openapi/model_mcp_tool_access_type.go
catalog/pkg/openapi/model_mcp_deployment_mode.go
catalog/pkg/openapi/model_mcp_artifact.go
catalog/pkg/openapi/api_mcp_catalog_service.go    # Generated API handler
```

### BFF Layer

#### Handlers

```
clients/ui/bff/internal/api/mcp_server_handler.go  # MCP API handlers
clients/ui/bff/internal/api/routes.go              # Route registration (modified)
clients/ui/bff/internal/api/app.go                 # App setup (modified)
```

#### Models

```
clients/ui/bff/internal/models/mcp_server.go       # BFF MCP server model
clients/ui/bff/internal/models/mcp_catalog_source.go
clients/ui/bff/internal/models/filter_options.go
```

#### Repositories

```
clients/ui/bff/internal/repositories/model_catalog_client.go  # Modified for MCP
```

### Frontend UI

#### Pages and Screens

```
clients/ui/frontend/src/app/pages/mcpCatalog/
├── McpCatalogRoutes.tsx                           # Route definitions
├── McpCatalogCoreLoader.tsx                       # Data loading wrapper
├── EmptyMcpCatalogState.tsx                       # Empty state component
└── screens/
    ├── McpCatalog.tsx                             # Main catalog page
    ├── McpCatalogGalleryView.tsx                  # Gallery layout
    ├── McpCatalogAllServersView.tsx               # List all view
    └── McpServerDetailsPage.tsx                   # Server details
    └── McpServerDetailsView.tsx                   # Details content
```

#### Components

```
clients/ui/frontend/src/app/pages/mcpCatalog/components/
├── McpCatalogCard.tsx                             # Server card component
├── McpCatalogCardBody.tsx                         # Card body content
├── McpCatalogCategorySection.tsx                  # Category grouping
├── McpCatalogFilters.tsx                          # Filter sidebar
├── McpCatalogLabels.tsx                           # Tag labels display
├── McpCatalogSourceLabelBlocks.tsx                # Source labels
├── McpCatalogStringFilter.tsx                     # String filter input
├── McpSecurityIndicators.tsx                      # Security badges
└── McpToolsList.tsx                               # Tools list display
```

#### Context and Hooks

```
clients/ui/frontend/src/app/context/McpCatalogContext.tsx  # React context
clients/ui/frontend/src/app/hooks/useMcpCatalog.ts         # Custom hooks
```

#### API Integration

```
clients/ui/frontend/src/app/api/mcpCatalogService.ts       # API client
clients/ui/frontend/src/app/api/types/mcpCatalog.ts        # TypeScript types
```

#### Routes and Navigation

```
clients/ui/frontend/src/app/AppRoutes.tsx          # Modified for MCP routes
clients/ui/frontend/src/app/components/NavSidebar.tsx  # Nav links added
```

### Test Files

#### Backend Tests

```
catalog/internal/mcp/db_mcp_catalog_test.go
catalog/internal/mcp/yaml_mcp_catalog_test.go
catalog/internal/catalog/mcp_loader_test.go
catalog/internal/catalog/mcp_source_merge_test.go
catalog/internal/catalog/mcp_server_filter_test.go
catalog/internal/db/service/mcp_server_repo_test.go
```

#### Test Data

```
catalog/internal/catalog/testdata/dev-community-mcp-servers.yaml
catalog/internal/catalog/testdata/dev-organization-mcp-servers.yaml
catalog/internal/catalog/testdata/test-mcp-sources.yaml
```

#### Frontend Tests

```
clients/ui/frontend/src/app/pages/mcpCatalog/__tests__/
├── McpCatalogCard.test.tsx
├── McpCatalogFilters.test.tsx
├── McpSecurityIndicators.test.tsx
└── McpToolsList.test.tsx
```

### Manifests and Configuration

```
manifests/kustomize/options/catalog/overlays/demo/
├── dev-mcp-catalog-sources.yaml                   # Demo MCP sources config
├── dev-community-mcp-servers.yaml                 # Demo community servers
├── dev-organization-mcp-servers.yaml              # Demo org servers
└── kustomization.yaml                             # Modified for MCP files
```

### Documentation

```
docs/mcp-catalog.md                                # Feature documentation
docs/configuration/mcp-sources.md                  # Configuration guide
```

## File Statistics

### By File Type

| Extension | Files | Lines Added |
|-----------|-------|-------------|
| `.go` | 35 | ~7,500 |
| `.tsx` | 20 | ~2,800 |
| `.ts` | 8 | ~900 |
| `.yaml` | 12 | ~2,400 |
| `.md` | 4 | ~350 |

### By Directory

| Directory | Files | Purpose |
|-----------|-------|---------|
| `catalog/internal/mcp/` | 6 | MCP catalog implementation |
| `catalog/internal/catalog/` | 5 | Loader and source handling |
| `catalog/internal/db/` | 4 | Database models and repos |
| `catalog/pkg/openapi/` | 12 | Generated API code |
| `clients/ui/bff/` | 6 | BFF handlers and models |
| `clients/ui/frontend/` | 25 | React UI components |
| `manifests/` | 5 | Kubernetes manifests |

## New Dependencies

### Go Modules

```go
// No new external dependencies - uses existing libraries:
// - github.com/deckarep/golang-set/v2 (already in project)
// - k8s.io/apimachinery/pkg/util/yaml (already in project)
```

### NPM Packages

```json
// No new frontend dependencies - uses existing:
// - @patternfly/react-core
// - @patternfly/react-icons
// - react-router-dom
```

---

[Back to MCP Catalog Index](./README.md) | [Previous: Implementation Overview](./implementation-overview.md) | [Next: Architecture](./architecture.md)
