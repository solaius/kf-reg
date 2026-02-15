# MCP Catalog Configuration Guide

This document explains how to configure MCP sources and servers for the MCP Catalog.

## Overview

The MCP Catalog uses a two-level configuration:

1. **Source Configuration** (`mcp-sources.yaml`) - Defines where to load MCP servers from
2. **Server Definitions** (`mcp-servers.yaml`) - Defines the actual MCP servers

## Source Configuration

### Basic Structure

```yaml
# mcp-sources.yaml
catalogs:
  - name: "Display Name"
    id: unique-source-id
    type: yaml                    # Provider type
    enabled: true                 # Enable/disable source
    labels:
      - label1
      - label2
    properties:
      yamlCatalogPath: ./servers.yaml  # Path to server definitions

# Optional: Named queries for pre-defined filters
namedQueries:
  verified-only:
    verifiedSource:
      operator: "="
      value: true
```

### Source Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier for the source |
| `name` | Yes | Human-readable display name |
| `type` | Yes | Provider type (`yaml` currently supported) |
| `enabled` | No | Enable/disable source (default: `true`) |
| `labels` | No | Array of labels for filtering |
| `properties` | Yes | Provider-specific configuration |
| `includedServers` | No | Glob patterns for servers to include |
| `excludedServers` | No | Glob patterns for servers to exclude |

### Server Filtering

Filter which servers are loaded from a source using glob patterns:

```yaml
catalogs:
  - id: org-mcp
    name: "Organization MCP Servers"
    type: yaml
    properties:
      yamlCatalogPath: all-servers.yaml

    # Only include servers matching these patterns
    includedServers:
      - "github-*"        # All GitHub-related servers
      - "slack-*"         # All Slack-related servers
      - "jira-*"          # All Jira-related servers

    # Exclude servers matching these patterns (takes precedence)
    excludedServers:
      - "*-deprecated"    # Exclude deprecated servers
      - "*-experimental"  # Exclude experimental servers
```

**Pattern Syntax:**
- `*` matches any sequence of characters
- Patterns are case-insensitive
- If `includedServers` is empty, all servers are included by default
- Exclusions always take precedence over inclusions

### Multi-Path Configuration

Configure multiple source paths for layered configuration:

```bash
# Base configuration
export MR_CATALOG_MCP_SOURCES_PATH=/etc/catalog/base-sources.yaml

# Override configuration (comma-separated)
export MR_CATALOG_MCP_SOURCES_PATH=/etc/catalog/base-sources.yaml,/etc/catalog/overrides.yaml
```

**Merge Behavior:**
- Sources with the same `id` are merged
- Later paths override earlier paths
- Non-nil fields in later sources override earlier values
- `enabled`, `labels`, and filter patterns are replaced (not merged)

**Example:**

```yaml
# base-sources.yaml
catalogs:
  - id: community
    name: "Community Servers"
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: community.yaml
```

```yaml
# overrides.yaml
catalogs:
  - id: community
    enabled: false  # Disable community source in this environment
```

## MCP Server Definitions

### Basic Server Structure

```yaml
# mcp-servers.yaml
source: "Source Display Name"

mcp_servers:
  - name: server-name
    provider: Provider Name
    description: Brief description of the server
    license: apache-2.0           # SPDX identifier
    license_link: https://...     # License URL
    version: "1.0.0"
    transports:
      - stdio                     # Or: http, sse
    tools:
      - name: tool_name
        description: What the tool does
        accessType: read_only     # Or: read_write, execute
        parameters:
          - name: param_name
            type: string
            description: Parameter description
            required: true
```

### Complete Server Example

```yaml
mcp_servers:
  - name: kubernetes-mcp
    provider: CNCF
    license: apache-2.0
    license_link: https://www.apache.org/licenses/LICENSE-2.0

    # Description (short, one-line)
    description: >-
      Interact with Kubernetes clusters through natural language.
      List pods, check deployments, view logs, and manage resources.

    # Readme (full markdown documentation)
    readme: |-
      # Kubernetes MCP Server

      **MCP Server Summary:**
      Comprehensive Kubernetes cluster management through MCP.

      ## Features
      - Pod management and listing
      - Deployment operations
      - Log retrieval

      ## Prerequisites
      - kubectl configured with cluster access
      - Appropriate RBAC permissions

    version: "1.2.0"
    transports:
      - stdio

    # Logo (base64 SVG or URL)
    logo: data:image/svg+xml;base64,PHN2Zy...

    # Links
    documentationUrl: https://docs.example.com/k8s-mcp
    repositoryUrl: https://github.com/org/kubernetes-mcp
    sourceCode: org/kubernetes-mcp

    publishedDate: "2025-01-12"

    # Tools exposed by this server
    tools:
      - name: get_pods
        description: List pods in a namespace
        accessType: read_only
        parameters:
          - name: namespace
            type: string
            description: Kubernetes namespace
            required: true

      - name: scale_deployment
        description: Scale a deployment
        accessType: read_write
        parameters:
          - name: namespace
            type: string
            description: Kubernetes namespace
            required: true
          - name: deployment_name
            type: string
            description: Name of the deployment
            required: true
          - name: replicas
            type: number
            description: Desired number of replicas
            required: true

    # Artifacts (OCI images for local deployment)
    artifacts:
      - uri: oci://ghcr.io/org/kubernetes-mcp:1.2.0
        createTimeSinceEpoch: "1736683200000"
        lastUpdateTimeSinceEpoch: "1736683200000"

    # Custom properties following Model Registry patterns
    customProperties:
      # Tags (MetadataStringValue with empty string_value)
      kubernetes:
        metadataType: MetadataStringValue
        string_value: ""
      containers:
        metadataType: MetadataStringValue
        string_value: ""
      orchestration:
        metadataType: MetadataStringValue
        string_value: ""

      # Security indicators (MetadataBoolValue)
      verifiedSource:
        metadataType: MetadataBoolValue
        bool_value: true
      secureEndpoint:
        metadataType: MetadataBoolValue
        bool_value: true
      sast:
        metadataType: MetadataBoolValue
        bool_value: true
      readOnlyTools:
        metadataType: MetadataBoolValue
        bool_value: false

    # Timestamps (epoch milliseconds)
    createTimeSinceEpoch: "1736683200000"
    lastUpdateTimeSinceEpoch: "1736683200000"
```

### Remote MCP Servers

For servers hosted externally (cloud-based MCP):

```yaml
mcp_servers:
  - name: openai-assistants-mcp
    provider: OpenAI
    description: Access OpenAI Assistants API

    # Deployment mode
    deploymentMode: remote

    # Network endpoints (instead of artifacts)
    endpoints:
      http: https://api.openai.com/v1/mcp
      sse: https://api.openai.com/v1/mcp/stream

    # No transports field needed - derived from endpoints
    # No artifacts - remote servers don't need local images

    tools:
      - name: chat
        description: Send a message to an assistant
        accessType: read_write
        parameters:
          - name: message
            type: string
            required: true
```

### Tool Revocation

Mark tools as revoked to warn users:

```yaml
tools:
  - name: dangerous_tool
    description: This tool had a security vulnerability
    accessType: execute
    revoked: true
    revokedReason: "CVE-2025-1234: Remote code execution vulnerability"
```

## Named Queries

Pre-define filter combinations for common use cases:

```yaml
# In mcp-sources.yaml
namedQueries:
  verified-servers:
    verifiedSource:
      operator: "="
      value: true

  read-only-servers:
    readOnlyTools:
      operator: "="
      value: true

  cloud-ai:
    deploymentMode:
      operator: "="
      value: "remote"
    provider:
      operator: IN
      value:
        - OpenAI
        - Anthropic
        - Google
```

**Usage in API:**

```bash
GET /api/v1/mcp_catalog/mcp_servers?namedQuery=verified-servers
```

## License Configuration

Use SPDX identifiers for licenses. The UI automatically converts them to display names:

| SPDX Identifier | Display Name |
|-----------------|--------------|
| `apache-2.0` | Apache 2.0 |
| `mit` | MIT |
| `gpl-3.0` | GPL 3.0 |
| `bsd-3-clause` | BSD 3-Clause |
| `elastic-2.0` | Elastic 2.0 |

## Security Indicators

| Indicator | Description | Meaning |
|-----------|-------------|---------|
| `verifiedSource` | Source code verification | Code is from a verified publisher |
| `secureEndpoint` | Transport security | Uses HTTPS/TLS for communication |
| `sast` | Static analysis | Has been scanned with SAST tools |
| `readOnlyTools` | Tool safety | All tools are read-only (no mutations) |

## Kubernetes Deployment

### ConfigMap for Sources

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-catalog-sources
data:
  mcp-sources.yaml: |
    catalogs:
      - id: org-mcp
        name: "Organization MCP Servers"
        type: yaml
        enabled: true
        properties:
          yamlCatalogPath: mcp-servers.yaml

  mcp-servers.yaml: |
    source: Organization
    mcp_servers:
      - name: internal-mcp
        provider: My Org
        # ... server definition
```

### Mount Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: catalog
          env:
            - name: MR_CATALOG_MCP_SOURCES_PATH
              value: /etc/catalog/mcp-sources.yaml
          volumeMounts:
            - name: mcp-config
              mountPath: /etc/catalog
      volumes:
        - name: mcp-config
          configMap:
            name: mcp-catalog-sources
```

## Hot Reload

The MCP Loader watches configuration files for changes:

- File modifications trigger automatic reload
- Sources are re-parsed and merged
- Servers are upserted (update if exists, create if new)
- Orphaned servers (from removed sources) are deleted

**Note:** Hot reload works with file-based sources. ConfigMap updates require pod restart unless using a sidecar like `reloader`.

---

[Back to MCP Catalog Index](./README.md) | [Previous: Architecture](./architecture.md) | [Next: Data Models](./data-models.md)
