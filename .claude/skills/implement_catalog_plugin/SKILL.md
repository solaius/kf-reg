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
7. Implement Phase 5 interfaces (required for new plugins)
   - `CapabilitiesV2Provider`: return full `PluginCapabilitiesV2` with entities, sources, actions, field metadata, UI hints
   - `AssetMapperProvider`: implement `AssetMapper` to project native entities to `AssetResource` universal envelope
   - `ActionProvider`: implement actions (at minimum: tag, annotate, deprecate via overlay store; refresh for sources)
   - Verify capabilities endpoint returns valid JSON: `GET /api/plugins/{name}/capabilities`
8. Add tests
   - Unit tests for parsing, mapping, and provider behavior
   - Integration tests that exercise sources.yaml -> ingest -> list/get
   - If you add filtering, include tests for filterQuery behavior
   - Test action execution and dry-run semantics
   - Run conformance suite: `CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1`
9. Validate end-to-end
   - Verify /api/plugins lists the plugin and its base path
   - Verify /api/plugins/{name}/capabilities returns complete V2 capabilities
   - Verify OpenAPI merge produces a unified spec without collisions
   - Verify generic UI renders the plugin without plugin-specific code
   - Verify CLI can list/get entities without CLI code changes

## Key Phase 5 interfaces

```go
// Required interfaces for capabilities-driven plugins
type CapabilitiesV2Provider interface {
    GetCapabilitiesV2() PluginCapabilitiesV2
}

type AssetMapperProvider interface {
    GetAssetMapper() AssetMapper
}

type ActionProvider interface {
    HandleAction(ctx context.Context, scope ActionScope, targetID string, req ActionRequest) (*ActionResult, error)
    ListActions(scope ActionScope) []ActionDefinition
}
```

## Validation
Use the repo loop:
- make gen
- make lint
- make test
- make openapi/validate (or the catalog OpenAPI validation target if different)
- CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1

If you touch DB schema or migrations, ensure the repo DB schema checks pass.

## Output
- A short summary of what you added
- The exact commands you ran and their results
- Links to the changed OpenAPI files and the generated outputs
- Conformance suite results
