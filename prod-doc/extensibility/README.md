# Extensibility Documentation

This section covers how to extend the Kubeflow Model Registry to support new asset types.

## Overview

The Model Registry architecture is designed to be extensible. While it currently focuses on ML models and MCP servers, the patterns established can support additional AI asset types such as prompts, knowledge bases, guardrails, and agents.

## Documentation

| Document | Description |
|----------|-------------|
| [Asset Type Framework](./asset-type-framework.md) | Core framework for new asset types |
| [Adding New Assets](./adding-new-assets.md) | Step-by-step implementation guide |
| [Proposed Assets](./proposed-assets.md) | Future AI asset type proposals |

## Current Asset Types

### Model Registry (Core)

| Entity | Description |
|--------|-------------|
| RegisteredModel | Logical model container |
| ModelVersion | Specific version of a model |
| ModelArtifact | Physical model files |
| InferenceService | Deployed model instance |
| ServingEnvironment | Deployment environment |

### Model Catalog

| Entity | Description |
|--------|-------------|
| CatalogModel | Curated model metadata |
| CatalogArtifact | Artifact references |
| CatalogSource | Data source configuration |

### MCP Catalog

| Entity | Description |
|--------|-------------|
| McpServer | MCP server definition |
| McpTool | Tools provided by MCP server |
| McpSource | MCP catalog source |

## Extension Points

```
┌─────────────────────────────────────────────────────────────────┐
│                      Extension Points                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    OpenAPI Specification                     │ │
│  │   Define new entities, endpoints, and operations            │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                   Database Models                            │ │
│  │   GORM entities with migrations                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                  Repository Layer                            │ │
│  │   Generic repository implementations                         │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                   Service Layer                              │ │
│  │   Business logic and orchestration                           │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                   API Handlers                               │ │
│  │   HTTP endpoints from OpenAPI generation                     │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                  Frontend Components                         │ │
│  │   React pages, tables, forms                                 │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Extensibility Patterns

### Pattern 1: New Core Entity

For entities that need versioning and artifact management:
- Extends MLMD-style Context/Artifact pattern
- Full CRUD operations
- Property system for custom metadata

### Pattern 2: Catalog Extension

For curated collections from external sources:
- APIProvider interface
- YAML/Database source providers
- Hot-reload configuration

### Pattern 3: Registry Integration

For entities that relate to existing models:
- Links to RegisteredModel/ModelVersion
- Shared serving infrastructure
- Common deployment patterns

## Quick Reference

### Adding a New Entity (High-Level)

1. **Define OpenAPI spec** - `api/openapi/`
2. **Generate code** - `make gen/openapi`
3. **Add database model** - `internal/db/models/`
4. **Create migrations** - `internal/datastore/embedmd/*/migrations/`
5. **Implement repository** - `internal/db/service/`
6. **Add service layer** - `internal/core/`
7. **Update frontend** - `clients/ui/frontend/`
8. **Update BFF** - `clients/ui/bff/`

### Key Files to Modify

| Area | Files |
|------|-------|
| API Spec | `api/openapi/model-registry.yaml` or `api/openapi/catalog.yaml` |
| Database | `internal/db/models/*.go` |
| Repository | `internal/db/service/*.go` |
| Service | `internal/core/*.go` |
| Handlers | `internal/server/openapi/*.go` |
| Frontend | `clients/ui/frontend/src/app/pages/` |
| BFF | `clients/ui/bff/internal/api/*_handler.go` |

---

[Back to Main Index](../README.md)
