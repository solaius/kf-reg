# MCP Catalog Architecture

This document covers the architecture and component design of the MCP Catalog feature.

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           MCP Catalog System                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────┐                                                          │
│  │  React Frontend │                                                         │
│  │  (/mcp-catalog) │                                                         │
│  └────────┬───────┘                                                          │
│           │ HTTP                                                             │
│           ▼                                                                  │
│  ┌────────────────┐     ┌─────────────────┐     ┌─────────────────┐         │
│  │      BFF       │────>│ Catalog Service │────>│    Database     │         │
│  │   (Go/chi)     │     │    (Go/GORM)    │     │   (MySQL/PG)    │         │
│  └────────────────┘     └────────┬────────┘     └─────────────────┘         │
│                                  │                                           │
│                                  ▼                                           │
│                         ┌─────────────────┐                                  │
│                         │   MCP Loader    │                                  │
│                         │   (fsnotify)    │                                  │
│                         └────────┬────────┘                                  │
│                                  │                                           │
│                                  ▼                                           │
│                         ┌─────────────────┐                                  │
│                         │  YAML Sources   │                                  │
│                         │ (ConfigMaps)    │                                  │
│                         └─────────────────┘                                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. MCP Loader

The `McpLoader` is the central orchestrator for loading MCP servers from configuration files.

```go
// catalog/internal/catalog/mcp_loader.go
type McpLoader struct {
    paths         []string                           // Config file paths
    services      service.Services                   // Database services
    sources       *SourceCollection                  // Shared source collection
    handlers      []McpLoaderEventHandler            // Post-load callbacks
    loadedSources map[string]bool                    // Loaded source tracker
    namedQueries  map[string]map[string]FieldFilter  // Merged named queries
}
```

**Responsibilities:**
- Parse source configuration YAML files
- Merge sources from multiple paths with priority
- Invoke registered providers for each source
- Apply server inclusion/exclusion filters
- Persist servers to database
- Hot-reload on file changes

**Lifecycle:**

```
Start()
    │
    ├─> readAndMergeSources()         # Parse all config files
    ├─> readAndMergeNamedQueries()    # Merge named query definitions
    ├─> removeMcpServersFromMissingSources()  # Clean up deleted sources
    ├─> mergeMcpSourcesIntoCollection()       # Unified sources API
    ├─> loadAllMcpServers()           # Load from all enabled sources
    │       │
    │       └─> readProviderRecords() # Call provider functions
    │               │
    │               └─> providerFunc(source) ──> records channel
    │
    └─> StartWatcher()                # Watch for file changes
            │
            └─> reloadAll()           # Full reload on change
```

### 2. MCP Provider Pattern

Providers are functions that emit MCP server records from a data source.

```go
// catalog/internal/mcp/yaml_mcp_catalog.go
type McpServerProviderFunc func(
    ctx context.Context,
    source *McpSource,
    reldir string,
) (<-chan McpServerProviderRecord, error)
```

**Provider Registration:**

```go
var RegisteredMcpProviders = map[string]McpServerProviderFunc{
    "yaml": NewYamlMcpProvider,
    // Future: "hf", "github", etc.
}
```

**YAML Provider Flow:**

```
NewYamlMcpProvider(ctx, source, reldir)
    │
    ├─> DetectYamlAssetType()   # Verify file contains mcp_servers
    ├─> Read YAML file
    ├─> Parse yamlMcpCatalog structure
    └─> Emit McpServerProviderRecord for each server
```

### 3. Database-Backed Catalog Provider

The `DbMcpCatalogProvider` implements the API for querying MCP servers from the database.

```go
// catalog/internal/mcp/db_mcp_catalog.go
type DbMcpCatalogProvider struct {
    repository         dbmodels.McpServerRepository
    namedQueryResolver NamedQueryResolver
}
```

**API Methods:**

```go
// List with filtering
ListMcpServers(ctx, name, q, filterQuery, namedQuery) ([]McpServer, error)

// Get by ID or name
GetMcpServer(ctx, serverId) (*McpServer, error)

// Available filter options
GetFilterOptions(ctx) (*FilterOptionsList, error)
```

**Query Resolution Flow:**

```
ListMcpServers(name, q, filterQuery, namedQuery)
    │
    ├─> Resolve named query to filter conditions
    ├─> Transform license display names to SPDX
    ├─> Build McpServerListOptions
    ├─> repository.List(options)
    └─> Convert DB entities to API models
```

### 4. McpServer Entity

```go
// catalog/internal/db/models/mcp_server.go
type McpServer interface {
    models.Entity[McpServerAttributes]
}

type McpServerAttributes struct {
    Name                     *string
    ExternalID               *string
    CreateTimeSinceEpoch     *int64
    LastUpdateTimeSinceEpoch *int64
}

// Properties stored in property table:
// - description, logo, provider, version, license
// - transports (JSON array), tools (JSON array)
// - deploymentMode, endpoints (JSON object)
// - verifiedSource, secureEndpoint, sast, readOnlyTools (bools)
// - tags (JSON array)
```

### 5. Source Filtering

Server inclusion/exclusion patterns filter which servers are loaded from each source.

```go
// catalog/internal/catalog/mcp_server_filter.go
type McpServerFilter struct {
    includePatterns []*regexp.Regexp
    excludePatterns []*regexp.Regexp
}

func (f *McpServerFilter) Allows(serverName string) bool {
    // 1. Check exclusions first - if matches any, reject
    // 2. If no include patterns, allow all not excluded
    // 3. If include patterns exist, must match at least one
}
```

**Pattern Syntax:**
- `*` matches any sequence of characters
- Patterns are case-insensitive
- Exclusions take precedence over inclusions

### 6. Source Merge Strategy

When multiple config paths are specified, sources are merged with later paths having priority.

```go
// catalog/internal/catalog/mcp_source_merge.go
func MergeMcpSourcesFromPaths(paths []string, readFunc func(string) ([]McpSource, error)) (map[string]McpSource, error) {
    merged := make(map[string]McpSource)

    for _, path := range paths {
        sources, _ := readFunc(path)
        for _, source := range sources {
            source.Origin = path  // Track origin for relative paths
            if existing, ok := merged[source.Id]; ok {
                merged[source.Id] = MergeMcpSource(existing, source)
            } else {
                merged[source.Id] = source
            }
        }
    }
    return merged
}
```

**Merge Rules:**
- Non-nil fields from later sources override earlier
- `Enabled` flag can be overridden
- Labels are replaced entirely (not merged)
- `IncludedServers` and `ExcludedServers` are replaced

## Data Flow

### Loading Flow

```
1. Service Startup
   │
   └─> McpLoader.Start()
       │
       ├─> Parse /etc/catalog/mcp-sources.yaml
       ├─> Parse /etc/catalog/custom-sources.yaml (override)
       │
       ├─> For each enabled source:
       │   ├─> Get provider for source.Type
       │   ├─> Call provider(source) -> channel of records
       │   ├─> Apply server filter (included/excluded)
       │   ├─> Set source_id property
       │   └─> Save to database via McpServerRepository
       │
       └─> Start file watcher for hot-reload

2. On File Change
   │
   └─> McpLoader.reloadAll()
       │
       ├─> Re-parse all config files
       ├─> Remove orphaned servers
       └─> Reload all servers (upsert)
```

### Query Flow

```
1. API Request: GET /api/v1/mcp_catalog/mcp_servers?q=kubernetes
   │
   ├─> BFF Handler receives request
   │   └─> Extracts query parameters
   │
   ├─> ModelCatalogClient.GetAllMcpServers()
   │   └─> HTTP GET to Catalog Service
   │
   ├─> Catalog API Handler
   │   └─> DbMcpCatalogProvider.ListMcpServers()
   │       │
   │       ├─> Build list options from params
   │       ├─> Apply text search filter
   │       ├─> Apply filter query (if present)
   │       ├─> Resolve named query (if present)
   │       │
   │       └─> McpServerRepository.List(options)
   │           │
   │           └─> GORM query with:
   │               - Text search on name, description
   │               - Filter conditions from filterQuery
   │               - Pagination
   │
   └─> Convert to API response
       └─> Return JSON with servers array
```

## Database Schema

### Entity Relationship

```
┌─────────────────┐         ┌─────────────────┐
│   mcp_server    │──1:N───>│ mcp_property    │
├─────────────────┤         ├─────────────────┤
│ id (PK)         │         │ entity_id (FK)  │
│ name            │         │ name            │
│ external_id     │         │ string_value    │
│ create_time     │         │ bool_value      │
│ last_update_time│         │ int_value       │
└─────────────────┘         └─────────────────┘
```

### Property Storage

Properties follow the Model Registry pattern using a generic property table:

| Property Name | Type | Description |
|---------------|------|-------------|
| `source_id` | string | Source identifier |
| `description` | string | Server description |
| `logo` | string | Base64 SVG or URL |
| `provider` | string | Provider name |
| `version` | string | Server version |
| `license` | string | SPDX identifier |
| `transports` | string (JSON) | `["stdio", "http"]` |
| `tools` | string (JSON) | Tool definitions array |
| `deploymentMode` | string | "local" or "remote" |
| `endpoints` | string (JSON) | `{"http": "...", "sse": "..."}` |
| `tags` | string (JSON) | Tag array |
| `verifiedSource` | bool | Security indicator |
| `secureEndpoint` | bool | Security indicator |
| `sast` | bool | Security indicator |
| `readOnlyTools` | bool | Security indicator |

## Security Model

### Security Indicators

```go
type McpSecurityIndicator struct {
    VerifiedSource *bool  // Source code from verified publisher
    SecureEndpoint *bool  // Uses HTTPS/TLS
    Sast           *bool  // Has static analysis security testing
    ReadOnlyTools  *bool  // All tools are read-only
}
```

### Trust Display

The UI displays trust badges based on security indicators:

```typescript
// McpSecurityIndicators.tsx
const indicators = [
  { key: 'verifiedSource', icon: <CheckCircleIcon />, label: 'Verified Source' },
  { key: 'secureEndpoint', icon: <LockIcon />, label: 'Secure Endpoint' },
  { key: 'sast', icon: <ShieldIcon />, label: 'SAST Scanned' },
  { key: 'readOnlyTools', icon: <EyeIcon />, label: 'Read-Only Tools' },
];
```

## Deployment Modes

### Local Deployment

```yaml
deploymentMode: local
transports:
  - stdio
artifacts:
  - uri: oci://ghcr.io/org/mcp-server:1.0.0
```

- Server runs locally via stdio
- Requires artifact (OCI image)
- No network endpoints

### Remote Deployment

```yaml
deploymentMode: remote
endpoints:
  http: https://api.example.com/mcp
  sse: https://api.example.com/mcp/events
```

- Server hosted externally
- Network endpoints required
- No artifacts needed

---

[Back to MCP Catalog Index](./README.md) | [Previous: Files Changed](./files-changed.md) | [Next: Configuration Guide](./configuration-guide.md)
