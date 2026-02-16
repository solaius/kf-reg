# Scope, Personas, and Requirements

_Last updated: 2026-02-16_

## Scope
Phase 3 delivers a working “catalog of catalogs” experience across:
- Plugins (catalog types)
- Sources (configured providers feeding a plugin)
- Entities (discovered assets served by each plugin)
- Diagnostics (health, last refresh, error summaries)

## User stories

### Ops for AI stories
- As an Ops user, I can list installed plugins and see their versions, supported entity types, and health
- As an Ops user, I can list sources for a plugin and see status, last sync, entity counts, and errors
- As an Ops user, I can add, edit, enable, disable, or delete a source and those changes persist
- As an Ops user, I can refresh a plugin or an individual source with rate limits and audit logs
- As an Ops user, I can validate configuration before applying (dry run)
- As an Ops user, I can view per-source diagnostics when ingestion fails

### AI Engineer stories
- As an AI Engineer, I can browse MCP servers and filter by:
  - local vs remote
  - transport
  - license
  - verification / security labels
  - provider category
- As an AI Engineer, I can open an MCP server detail page and see:
  - description and capabilities
  - local container image (if local)
  - remote endpoint (if remote)
  - transports and auth hints
  - labels, verification, and supported tools or restrictions
- As an AI Engineer, I can copy relevant connection info from UI and CLI output

## Cross-cutting requirements

### R1: No placeholder data in default runtime mode
Mocks can remain for unit tests and optional dev-mode flags
The default dev + E2E setup must run against a real catalog-server and real provider data

### R2: Persistence
Source configuration changes must survive:
- catalog-server restart
- bff restart
- ui refresh

### R3: Backward compatibility
Existing model catalog API paths and semantics remain unchanged
New management endpoints are additive

### R4: Security and RBAC
Role-based access:
- viewer: read-only
- operator: can mutate sources and trigger refresh
All mutation endpoints enforce RBAC consistently

### R5: Observability
At minimum:
- structured logs on all mutations and refreshes
- per-source error summaries
- a stable “diagnostics” API consumed by UI and CLI
