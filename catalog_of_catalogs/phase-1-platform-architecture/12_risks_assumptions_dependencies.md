# Risks, assumptions, and dependencies

## Assumptions
- Plugin-based architecture PR is the base direction for this work
- Model Catalog remains the initial anchor plugin and cannot break
- Database persistence remains the source of truth for API responses
- FilterQuery and pagination semantics are shared across plugins

## Key dependencies
- Upstream review and merge of plugin architecture and catalog-gen tooling
- Existing OpenAPI merge tooling and validation
- DB migration patterns working across SQLite, MySQL, and PostgreSQL
- UI BFF and frontend patterns in the repo

## Risks
- OpenAPI merge collisions as plugin count grows
- Plugin lifecycle complexity and error handling causing brittle startup
- Generated code drift if deterministic generation is not enforced in CI
- Cross-asset linking becoming too tightly coupled too early
- Provider security patterns not standardized, leading to secret handling issues

## Mitigations
- Strict CI checks for OpenAPI merge determinism
- Contract tests for plugin lifecycle and server startup failure modes
- Clear docs and templates for provider auth and secret references
- Keep cross-asset linking minimal and additive in early versions

