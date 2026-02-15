# Skill: implement_catalog_plugin

## When to use
Use this skill when adding a new asset-type plugin or extending an existing plugin.

## Repo rules to obey
- Contract-first OpenAPI: update the spec first, then regenerate server stubs and clients
- Never edit generated files directly
- Keep API changes additive and preserve existing Model Catalog paths and behavior
- Follow repo error handling (sentinel errors) and logging conventions

## Steps
1. Define the asset type
   - Entity name, optional artifact types, and the minimal fields needed for list and detail
   - Decide what is stored as first-class typed fields vs customProperties
2. Define the API contract
   - Add or update the plugin OpenAPI spec
   - Ensure it composes shared schemas (BaseResource, BaseResourceList, MetadataValue)
   - Ensure list endpoints support page tokens and filterQuery
3. Regenerate code
   - Run the repo standard generators (make gen) and any plugin-specific OpenAPI generation targets
   - Confirm generated code is committed and in sync with the spec
4. Implement persistence and query
   - Follow the repo database patterns (GORM, migrations, repository patterns)
   - Keep migrations idempotent and run once per plugin
   - Add filter mappings so filterQuery maps to DB columns or property rows
5. Implement ingestion
   - Implement at least one provider type (YAML first)
   - Ingest into the shared DB and make API reads DB-backed
   - Implement hot reload or watchers only if required and clearly bounded
6. Wire plugin lifecycle
   - Init: read plugin config from sources.yaml
   - Migrations: apply schema for the plugin
   - RegisterRoutes: mount under the plugin base path
   - Start: start background loaders/watchers if used
   - Healthy: readiness checks must reflect ingest health
7. Add tests
   - Unit tests for parsing, mapping, and provider behavior
   - Integration tests that exercise sources.yaml -> ingest -> list/get
   - If you add filtering, include tests for filterQuery behavior
8. Validate end-to-end
   - Verify /api/plugins lists the plugin and its base path
   - Verify OpenAPI merge produces a unified spec without collisions

## Validation
Use the repo loop:
- make gen
- make lint
- make test
- make openapi/validate (or the catalog OpenAPI validation target if different)

If you touch DB schema or migrations, ensure the repo DB schema checks pass.

## Output
- A short summary of what you added
- The exact commands you ran and their results
- Links to the changed OpenAPI files and the generated outputs
