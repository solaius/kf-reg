# API contract expectations

This file defines API-level expectations to support generic UI and CLI consumers and to keep plugin behavior consistent.

## Design principles
- Contract-first, OpenAPI is the source of truth
- Consistent patterns across plugins
- Additive evolution only
- Clear separation between catalog and registry responsibilities

## Base path conventions
- Each plugin owns a base path
- Example pattern: /api/<asset>_catalog/v1alpha1/...
- The model plugin base path stays unchanged
- Avoid mixing multiple asset types under one base path

## Required endpoints per plugin
Minimum set
- List entities
- Get entity
- List sources
- Get source (optional but recommended if sources are addressable resources)

Server-level endpoints
- List plugins
- Health and readiness

Optional, per plugin
- List artifacts for an entity
- Get artifact
- List supported provider types (if the plugin supports multiple providers)

## Resource naming and identifiers
- Entities must have a stable name within a source
- The API should expose enough fields to create a stable reference
- If global uniqueness is needed, it is achieved by combining plugin, source, and name

## Query parameters
### filterQuery
- A single string parameter used across list endpoints
- The grammar should be documented and consistent
- Plugins must explicitly map filterable fields to DB representation
- If custom properties are filterable, document which keys are supported

### Pagination and ordering
- pageSize
- nextPageToken
- orderBy (field name)
- sortOrder (ASC, DESC)

Pagination must be stable and consistent across plugins to support generic clients.

## Response shapes
### List responses
- Use a shared list envelope
- Include nextPageToken if more results exist
- Include items array with consistent baseline fields

### Error responses
- Use standard HTTP status codes
- Return JSON error bodies with enough information to debug:
  - code
  - message
  - details (optional)
  - pluginName and sourceId where applicable

## API versioning
- Keep API version in the path
- Prefer adding fields and endpoints rather than changing existing ones
- If a plugin needs a new version, publish it side-by-side under a new versioned path

## Compatibility checks
- Any change touching the model plugin must include a compatibility test
- Any change touching shared schemas must be validated against existing clients where feasible

