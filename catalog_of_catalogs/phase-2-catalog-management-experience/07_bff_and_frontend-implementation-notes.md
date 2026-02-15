# BFF and Frontend Implementation Notes

This file is guidance for implementation planning. It is not a design document for final code.

## BFF scope in Phase 2

- Continue to use BFF as the UI adapter
- BFF fetches plugin discovery and translates it into UI friendly structures
- BFF proxies or aggregates catalog server APIs when needed for auth and multi tenant concerns
- BFF exposes management plane endpoints to UI with RBAC enforcement and clear error mapping

## Required BFF endpoints

- GET /api/v1/catalog/plugins
- GET /api/v1/catalog/plugins/{plugin}/capabilities
- GET /api/v1/catalog/{plugin}/entities
- GET /api/v1/catalog/{plugin}/entities/{id}
- GET /api/v1/catalog/{plugin}/sources
- POST /api/v1/catalog/{plugin}/sources/validate
- POST /api/v1/catalog/{plugin}/sources/apply
- POST /api/v1/catalog/{plugin}/refresh

Exact naming can differ, but the semantic coverage is required

## Frontend scope in Phase 2

- Implement plugin switcher and routing
- Implement generic list and detail renderer
- Implement sources management views
- Implement status and diagnostics views
- Provide feature gating based on plugin capabilities and RBAC

## Implementation constraints

- Follow PROGRAMMING_GUIDELINES.md
- Prefer generated API clients derived from OpenAPI
- Keep the BFF as a thin layer
  - auth
  - mapping
  - aggregation
- Avoid duplicating business logic from catalog-server in the UI stack
