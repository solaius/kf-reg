# MCP Catalog Data Models

This document covers the data models used in the MCP Catalog feature.

## Entity Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    MCP Catalog Entities                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐         ┌─────────────┐                    │
│  │  McpSource  │───1:N──>│  McpServer  │                    │
│  └─────────────┘         └──────┬──────┘                    │
│                                 │                            │
│                                 │ 1:N                        │
│                                 ▼                            │
│                          ┌─────────────┐                    │
│                          │   McpTool   │                    │
│                          └─────────────┘                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## McpSource

Represents a catalog source that provides MCP servers.

### Go Definition

```go
// catalog/internal/mcp/yaml_mcp_catalog.go
type McpSource struct {
    Id              string         `json:"id"`
    Name            string         `json:"name"`
    Type            string         `json:"type"`
    Enabled         *bool          `json:"enabled,omitempty"`
    Labels          []string       `json:"labels,omitempty"`
    Properties      map[string]any `json:"properties,omitempty"`
    Origin          string         `json:"-" yaml:"-"`
    IncludedServers []string       `json:"includedServers,omitempty"`
    ExcludedServers []string       `json:"excludedServers,omitempty"`
}
```

### OpenAPI Schema

```yaml
McpCatalogSource:
  type: object
  properties:
    id:
      type: string
      description: Unique source identifier
    name:
      type: string
      description: Display name
    labels:
      type: array
      items:
        type: string
    enabled:
      type: boolean
      default: true
    assetType:
      type: string
      enum: [models, mcp_servers]
```

## McpServer

Represents an MCP server with its metadata, tools, and artifacts.

### Go Database Entity

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

type McpServerImpl = models.BaseEntity[McpServerAttributes]
```

### OpenAPI Schema

```yaml
McpServer:
  type: object
  required:
    - name
  properties:
    id:
      type: string
      description: Unique server identifier
    name:
      type: string
      description: Server name
    sourceId:
      type: string
      description: Source this server came from
    description:
      type: string
      description: Brief description
    readme:
      type: string
      description: Full markdown documentation
    logo:
      type: string
      description: Base64 SVG or image URL
    provider:
      type: string
      description: Provider/organization name
    version:
      type: string
      description: Server version
    license:
      type: string
      description: License display name
    licenseLink:
      type: string
      format: uri
      description: License URL
    transports:
      type: array
      items:
        $ref: '#/components/schemas/McpTransportType'
    deploymentMode:
      $ref: '#/components/schemas/McpDeploymentMode'
    endpoints:
      $ref: '#/components/schemas/McpEndpoints'
    tools:
      type: array
      items:
        $ref: '#/components/schemas/McpTool'
    artifacts:
      type: array
      items:
        $ref: '#/components/schemas/McpArtifact'
    securityIndicators:
      $ref: '#/components/schemas/McpSecurityIndicator'
    tags:
      type: array
      items:
        type: string
    documentationUrl:
      type: string
      format: uri
    repositoryUrl:
      type: string
      format: uri
    sourceCode:
      type: string
      description: Repository path (e.g., org/repo)
    publishedDate:
      type: string
      description: Publication date
    lastUpdated:
      type: string
      format: date-time
    customProperties:
      type: object
      additionalProperties:
        $ref: '#/components/schemas/MetadataValue'
```

### TypeScript Interface

```typescript
// clients/ui/frontend/src/app/api/types/mcpCatalog.ts
interface McpServer {
  id: string;
  name: string;
  sourceId?: string;
  description?: string;
  readme?: string;
  logo?: string;
  provider?: string;
  version?: string;
  license?: string;
  licenseLink?: string;
  transports: McpTransportType[];
  deploymentMode?: McpDeploymentMode;
  endpoints?: McpEndpoints;
  tools?: McpTool[];
  artifacts?: McpArtifact[];
  securityIndicators?: McpSecurityIndicator;
  tags?: string[];
  documentationUrl?: string;
  repositoryUrl?: string;
  sourceCode?: string;
  publishedDate?: string;
  lastUpdated?: string;
  customProperties?: Record<string, MetadataValue>;
}
```

## McpTool

Represents a tool exposed by an MCP server.

### YAML Definition

```yaml
tools:
  - name: get_pods
    description: List pods in a namespace
    accessType: read_only
    parameters:
      - name: namespace
        type: string
        description: Kubernetes namespace
        required: true
    revoked: false
    revokedReason: ""
```

### OpenAPI Schema

```yaml
McpTool:
  type: object
  required:
    - name
    - accessType
  properties:
    name:
      type: string
      description: Tool name
    description:
      type: string
      description: Tool description
    accessType:
      $ref: '#/components/schemas/McpToolAccessType'
    parameters:
      type: array
      items:
        $ref: '#/components/schemas/McpToolParameter'
    revoked:
      type: boolean
      default: false
      description: Whether the tool is revoked
    revokedReason:
      type: string
      description: Reason for revocation
```

### TypeScript Interface

```typescript
interface McpTool {
  name: string;
  description?: string;
  accessType: McpToolAccessType;
  parameters?: McpToolParameter[];
  revoked?: boolean;
  revokedReason?: string;
}
```

## McpToolParameter

Describes a parameter for an MCP tool.

### OpenAPI Schema

```yaml
McpToolParameter:
  type: object
  required:
    - name
    - type
    - required
  properties:
    name:
      type: string
      description: Parameter name
    type:
      type: string
      description: Parameter type (string, number, boolean, object)
    description:
      type: string
      description: Parameter description
    required:
      type: boolean
      description: Whether the parameter is required
```

## McpEndpoints

Network endpoints for remote MCP servers.

### OpenAPI Schema

```yaml
McpEndpoints:
  type: object
  properties:
    http:
      type: string
      format: uri
      description: HTTP endpoint URL
    sse:
      type: string
      format: uri
      description: Server-Sent Events endpoint URL
```

## McpArtifact

Represents a deployable artifact (OCI image) for local MCP servers.

### OpenAPI Schema

```yaml
McpArtifact:
  type: object
  required:
    - uri
  properties:
    uri:
      type: string
      description: Artifact URI (e.g., oci://ghcr.io/org/image:tag)
    createTimeSinceEpoch:
      type: string
      description: Creation timestamp (epoch milliseconds)
    lastUpdateTimeSinceEpoch:
      type: string
      description: Last update timestamp (epoch milliseconds)
```

## McpSecurityIndicator

Security trust signals for an MCP server.

### OpenAPI Schema

```yaml
McpSecurityIndicator:
  type: object
  properties:
    verifiedSource:
      type: boolean
      description: Source code is from a verified publisher
    secureEndpoint:
      type: boolean
      description: Uses HTTPS/TLS for communication
    sast:
      type: boolean
      description: Has been scanned with static analysis tools
    readOnlyTools:
      type: boolean
      description: All tools are read-only (no mutations)
```

## Enums

### McpTransportType

```yaml
McpTransportType:
  type: string
  enum:
    - stdio    # Standard input/output (local)
    - http     # HTTP/REST (remote)
    - sse      # Server-Sent Events (remote)
```

### McpToolAccessType

```yaml
McpToolAccessType:
  type: string
  enum:
    - read_only    # Tool only reads data
    - read_write   # Tool can read and modify data
    - execute      # Tool executes actions
```

### McpDeploymentMode

```yaml
McpDeploymentMode:
  type: string
  enum:
    - local    # Server runs locally via stdio
    - remote   # Server hosted externally via network
```

## Property Storage

Properties are stored in a generic property table following the Model Registry pattern:

### Database Schema

```sql
CREATE TABLE mcp_server_property (
    entity_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_custom_property BOOLEAN DEFAULT FALSE,
    string_value TEXT,
    bool_value BOOLEAN,
    int_value BIGINT,
    double_value DOUBLE,
    PRIMARY KEY (entity_id, name),
    FOREIGN KEY (entity_id) REFERENCES mcp_server(id)
);
```

### Property Mapping

| Property Name | Go Type | DB Column | Notes |
|---------------|---------|-----------|-------|
| `source_id` | string | `string_value` | Source identifier |
| `description` | string | `string_value` | Server description |
| `logo` | string | `string_value` | Base64 or URL |
| `provider` | string | `string_value` | Provider name |
| `version` | string | `string_value` | Version string |
| `license` | string | `string_value` | SPDX identifier |
| `license_link` | string | `string_value` | License URL |
| `transports` | []string | `string_value` | JSON array |
| `tools` | []McpTool | `string_value` | JSON array |
| `artifacts` | []McpArtifact | `string_value` | JSON array |
| `deploymentMode` | string | `string_value` | "local" or "remote" |
| `endpoints` | McpEndpoints | `string_value` | JSON object |
| `tags` | []string | `string_value` | JSON array |
| `documentationUrl` | string | `string_value` | URL |
| `repositoryUrl` | string | `string_value` | URL |
| `sourceCode` | string | `string_value` | Repository path |
| `readme` | string | `string_value` | Markdown content |
| `publishedDate` | string | `string_value` | Date string |
| `verifiedSource` | bool | `bool_value` | Security indicator |
| `secureEndpoint` | bool | `bool_value` | Security indicator |
| `sast` | bool | `bool_value` | Security indicator |
| `readOnlyTools` | bool | `bool_value` | Security indicator |

## List Options

```go
// catalog/internal/db/models/mcp_server.go
type McpServerListOptions struct {
    models.Pagination
    Name        *string    // Name filter
    SourceIDs   *[]string  // Filter by source IDs
    Query       *string    // Legacy name search
    TextSearch  *string    // Free-form text search
    FilterQuery *string    // Advanced filter DSL
    NamedQuery  *string    // Pre-defined query name
}
```

## Repository Interface

```go
// catalog/internal/db/models/mcp_server.go
type McpServerRepository interface {
    GetByID(id int32) (McpServer, error)
    GetByName(name string) (McpServer, error)
    List(listOptions McpServerListOptions) (*models.ListWrapper[McpServer], error)
    Save(server McpServer) (McpServer, error)
    DeleteBySource(sourceID string) error
    DeleteByID(id int32) error
    GetDistinctSourceIDs() ([]string, error)
}
```

---

[Back to MCP Catalog Index](./README.md) | [Previous: Configuration Guide](./configuration-guide.md) | [Next: Remaining Work](./remaining-work.md)
