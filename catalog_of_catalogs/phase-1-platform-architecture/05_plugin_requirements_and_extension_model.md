# Plugin requirements and extension model

## Plugin contract expectations
A plugin should be the unit of isolation per asset type. Plugins own:
- API base path and route registration
- DB tables and migration steps
- Providers and ingestion logic
- OpenAPI paths and entity schemas
- Conventions for identifiers and references

The server owns:
- HTTP server lifecycle
- shared DB connection and migration orchestration
- unified configuration loading
- unified OpenAPI merge tooling
- common helper libraries (filtering, pagination, source definitions)

## Required plugin capabilities
- Provide list and get for primary entities
- Provide list and get for sources relevant to that plugin
- Provide health indicator reflecting ingestion and serving readiness
- Provide filter mappings so filterQuery behaves predictably

## Extension points
### Provider types
Provider types should be pluggable within a plugin:
- YAML file provider
- HTTP provider
- Registry-backed provider
- Git provider

### Artifact types
A plugin may define artifacts for its entities
- Artifacts must have stable identifiers
- Artifact list endpoints follow the same query and pagination patterns

### Custom logic
Plugins may require custom loading logic, for example:
- reading performance metrics from a directory
- resolving artifacts from a remote store
- synthesizing computed fields

The framework should allow this without forcing every plugin to fork the loader.

Requirements for custom logic support
- Hooks or template overrides for generated boilerplate
- A stable internal API so plugins can evolve without breaking the server

## Backward compatible naming
Some plugins may need to read config keys that differ from their internal plugin name for compatibility
- Example: plugin name model reads from config section models
Support for this should be explicit and tested.

