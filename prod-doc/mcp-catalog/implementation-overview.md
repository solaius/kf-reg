# MCP Catalog Implementation Overview

This document provides a high-level summary of the MCP Catalog feature implementation.

## What is MCP Catalog?

The MCP (Model Context Protocol) Catalog extends the Kubeflow Model Registry to support discovery, management, and governance of MCP servers. MCP is a standardized protocol that allows AI agents and applications to interact with external tools, data sources, and services in a consistent way.

## Feature Summary

### What Was Built

The MCP Catalog implementation adds:

1. **Full-Stack MCP Server Discovery**
   - React-based gallery UI for browsing MCP servers
   - Backend API for listing, filtering, and retrieving MCP servers
   - Database-backed storage for MCP server metadata

2. **YAML-Based Configuration**
   - Define MCP servers in YAML files
   - Configure sources with inclusion/exclusion filters
   - Support for multiple source files with merge priority

3. **Rich Metadata Support**
   - Server properties (name, provider, license, version)
   - Tool definitions with parameters
   - Security indicators (verified source, SAST, secure endpoint)
   - Deployment modes (local vs remote)
   - Transport types (stdio, http, sse)

4. **Advanced Filtering**
   - Free-form text search across multiple fields
   - Filter query DSL for precise filtering
   - Named queries for pre-defined filter sets
   - Filter options API for dynamic UI

## Implementation Phases

### Phase 1: UI Gallery View

Created the frontend components for MCP server discovery:

- **McpCatalogGalleryView**: Card-based gallery layout
- **McpServerDetailsPage**: Individual server details
- **McpCatalogFilters**: Sidebar filter components
- **McpSecurityIndicators**: Trust badge displays
- **McpToolsList**: Tool listing with parameters

### Phase 2: Database-Backed YAML Source

Implemented backend persistence and data loading:

- **McpServer Entity**: Database model for MCP servers
- **McpServerRepository**: CRUD operations
- **McpLoader**: YAML file parsing and database sync
- **DbMcpCatalogProvider**: Database-backed API provider

### Phase 3: Source Filtering & Merge

Added source-level configuration and multi-file support:

- **Source Inclusion/Exclusion**: Glob patterns for server filtering
- **Multi-Path Merge**: Later sources override earlier ones
- **Named Queries**: Pre-defined filter configurations
- **Hot-Reload**: File watcher for config changes

## Key Components

### Backend Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `McpLoader` | `catalog/internal/catalog/mcp_loader.go` | Loads MCP servers from YAML |
| `DbMcpCatalogProvider` | `catalog/internal/mcp/db_mcp_catalog.go` | Database-backed catalog provider |
| `YamlMcpProvider` | `catalog/internal/mcp/yaml_mcp_catalog.go` | YAML file provider |
| `McpServerRepository` | `catalog/internal/db/models/mcp_server.go` | Repository interface |

### Frontend Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `McpCatalog` | `clients/ui/frontend/src/app/pages/mcpCatalog/` | Main catalog page |
| `McpCatalogGalleryView` | `screens/McpCatalogGalleryView.tsx` | Gallery layout |
| `McpServerDetailsView` | `screens/McpServerDetailsView.tsx` | Server details |
| `McpCatalogFilters` | `components/McpCatalogFilters.tsx` | Filter sidebar |

### BFF Handlers

| Handler | Location | Purpose |
|---------|----------|---------|
| `GetAllMcpServersHandler` | `bff/internal/api/mcp_server_handler.go` | List MCP servers |
| `GetMcpServerHandler` | Same file | Get single server |
| `GetMcpFilterOptionsHandler` | Same file | Get filter options |
| `GetAllMcpSourcesHandler` | Same file | List sources |

## Lines of Code

The main implementation commit added:

- **Total Files Changed**: 99
- **Lines Added**: +13,931
- **Lines Removed**: -2,471
- **Net Change**: +11,460

### Breakdown by Area

| Area | Files | Lines Added |
|------|-------|-------------|
| Catalog Backend | 15 | ~4,500 |
| Frontend UI | 20 | ~3,000 |
| BFF Layer | 8 | ~1,200 |
| OpenAPI Specs | 3 | ~2,000 |
| Test Files | 12 | ~1,800 |
| Manifests/Config | 10 | ~1,400 |

## Key Design Decisions

### 1. Asset Type Detection

The implementation auto-detects asset type from YAML content:

```go
// common/asset_detector.go
func DetectYamlAssetType(source SourceProperties, reldir string) (AssetType, error) {
    // Check for "mcp_servers" key -> AssetTypeMcpServers
    // Check for "models" key -> AssetTypeModels
    // Otherwise -> AssetTypeModels (default)
}
```

### 2. Unified Source Collection

MCP sources are merged into the shared `SourceCollection` for a unified `/sources` API:

```go
// McpLoader merges into shared collection
l.sources.MergeWithNamedQueries(path, catalogSources, l.namedQueries)
```

### 3. License Display Names

SPDX license identifiers are converted to user-friendly display names:

```go
var spdxToDisplayName = map[string]string{
    "apache-2.0": "Apache 2.0",
    "mit":        "MIT",
    // ...
}
```

### 4. Security Indicators

Security trust signals are stored as boolean custom properties:

- `verifiedSource`: Source code is from a verified publisher
- `secureEndpoint`: Uses HTTPS/TLS
- `sast`: Has static analysis security testing
- `readOnlyTools`: All tools are read-only

## Testing Approach

### Unit Tests

- Repository tests with mock databases
- Provider tests with test YAML files
- Converter tests for data transformation

### Integration Tests

- End-to-end API tests
- Database migration tests
- Hot-reload behavior tests

### Frontend Tests

- Component unit tests with Jest
- E2E tests with Cypress
- Accessibility testing

---

[Back to MCP Catalog Index](./README.md) | [Next: Files Changed](./files-changed.md)
