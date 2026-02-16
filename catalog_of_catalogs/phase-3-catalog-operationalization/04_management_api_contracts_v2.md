# Management API Contracts v2

_Last updated: 2026-02-16_

## Goals
- Stabilize management plane APIs for UI and CLI consumption
- Ensure consistent envelope shapes and error models
- Ensure RBAC enforcement on all mutation endpoints

## Contract-first requirement
All new or changed endpoints must be defined in OpenAPI first, then generated, then implemented

## Endpoints (target)
- GET /api/plugins
  - list installed plugins with health summary
- GET /api/plugins/{plugin}/sources
  - list sources with status summary and entity counts
- POST /api/plugins/{plugin}/sources:validate
  - validate a source config without applying
- POST /api/plugins/{plugin}/sources:apply
  - create or update a source config
- POST /api/plugins/{plugin}/sources/{sourceId}:enable
- POST /api/plugins/{plugin}/sources/{sourceId}:disable
- DELETE /api/plugins/{plugin}/sources/{sourceId}
- POST /api/plugins/{plugin}:refresh
- POST /api/plugins/{plugin}/sources/{sourceId}:refresh
- GET /api/plugins/{plugin}/diagnostics
- GET /api/plugins/{plugin}/sources/{sourceId}/diagnostics

## Status model requirements
Each source returns:
- enabled
- lastRefreshTime
- lastRefreshStatus (success, failed, running)
- entityCount
- errorSummary (short)
Diagnostics endpoint returns:
- errors with code and message
- provider info
- lastSuccessfulRefresh
- ingestion metrics (optional)

## RBAC
- viewer: can call GET endpoints only
- operator: can call POST and DELETE endpoints
Return 403 for insufficient role

## Definition of Done
- OpenAPI updated and merged successfully
- Generated server stubs compile with no manual edits in generated files
- UI and CLI use only contract-defined fields
- Backward-compatible behavior preserved for existing endpoints

## Acceptance Criteria
- AC1: All management endpoints appear in merged OpenAPI and Swagger UI loads
- AC2: UI builds without using out-of-contract fields
- AC3: CLI commands succeed against a real server and show expected fields
- AC4: Role gating enforced: viewer cannot mutate, operator can
