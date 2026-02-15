# MCP Catalog Documentation

This section covers the Model Context Protocol (MCP) Catalog feature implementation in the Kubeflow Model Registry.

## Overview

The MCP Catalog extends the Model Registry to support discovery and management of MCP servers - standardized interfaces that allow AI agents to interact with external tools, data sources, and services.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **MCP Server** | A service exposing tools through the MCP protocol |
| **MCP Tool** | An individual capability exposed by an MCP server |
| **Transport** | Communication protocol (stdio, http, sse) |
| **Deployment Mode** | Local (stdio) or Remote (http/sse) |
| **Security Indicators** | Trust signals (verified source, SAST, etc.) |

## Feature Status

**Branch:** `feature/mcp-catalog`
**Status:** In development (not yet merged to main)
**Commits:** 4 unique commits with +13,931 lines added

## Documentation

| Document | Description |
|----------|-------------|
| [Implementation Overview](./implementation-overview.md) | High-level summary of what was built |
| [Files Changed](./files-changed.md) | Complete inventory of modified files |
| [Architecture](./architecture.md) | MCP-specific architecture and components |
| [Configuration Guide](./configuration-guide.md) | How to configure MCP sources |
| [Data Models](./data-models.md) | McpServer, McpTool entities |
| [Remaining Work](./remaining-work.md) | Outstanding tasks and TODOs |
| [Step-by-Step Creation](./step-by-step-creation.md) | How the feature was implemented |

## API Endpoints

### MCP Server Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/mcp_catalog/mcp_servers` | List all MCP servers |
| GET | `/api/v1/mcp_catalog/mcp_servers/{id}` | Get MCP server by ID |
| GET | `/api/v1/mcp_catalog/filter_options` | Get available filter options |
| GET | `/api/v1/mcp_catalog/sources` | List MCP sources |

### Query Parameters

```bash
# Free-form text search
GET /mcp_servers?q=kubernetes

# Filter by provider
GET /mcp_servers?filterQuery=provider='CNCF'

# Use named query
GET /mcp_servers?namedQuery=verified-servers

# Combine filters
GET /mcp_servers?q=ai&filterQuery=deploymentMode='remote'
```

## Quick Start

### 1. Create MCP Source Configuration

```yaml
# mcp-sources.yaml
catalogs:
  - name: "Organization MCP Servers"
    id: org-mcp
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: mcp-servers.yaml
```

### 2. Define MCP Servers

```yaml
# mcp-servers.yaml
source: Organization
mcp_servers:
  - name: my-mcp-server
    provider: My Org
    description: Custom MCP server
    license: apache-2.0
    transports:
      - stdio
    tools:
      - name: my_tool
        description: Does something useful
        accessType: read_only
```

### 3. Configure Catalog Service

```bash
# Set MCP sources path
export MR_CATALOG_MCP_SOURCES_PATH=/path/to/mcp-sources.yaml

# Start catalog service
./catalog serve
```

## Component Relationships

```
┌─────────────────────────────────────────────────────────────────┐
│                    MCP Catalog Architecture                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐       ┌──────────────┐       ┌──────────────┐ │
│  │    React     │  ──>  │     BFF      │  ──>  │   Catalog    │ │
│  │   Frontend   │       │   Handlers   │       │   Service    │ │
│  └──────────────┘       └──────────────┘       └──────────────┘ │
│         │                      │                      │          │
│         ▼                      ▼                      ▼          │
│  ┌──────────────┐       ┌──────────────┐       ┌──────────────┐ │
│  │  MCP Catalog │       │  MCP Client  │       │  MCP Loader  │ │
│  │   Gallery    │       │  Repository  │       │              │ │
│  └──────────────┘       └──────────────┘       └──────────────┘ │
│                                                        │          │
│                                                        ▼          │
│                                                ┌──────────────┐  │
│                                                │  YAML/DB     │  │
│                                                │  Provider    │  │
│                                                └──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

[Back to Main Index](../README.md)
