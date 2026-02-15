# Skill: validate_openapi_and_contracts

## When to use
Use this skill when you add or change any API paths or schemas, including plugin OpenAPI specs.

## Repo rules to obey
- Contract-first: the OpenAPI spec is the source of truth
- Generated code must be regenerated and committed
- Changes must be additive and must not break existing Model Catalog paths and schemas

## Steps
1. Update source OpenAPI files
   - Core specs under api/openapi/src
   - Plugin specs under catalog/plugins/<name>/api/openapi/src (if applicable)
2. Regenerate and merge
   - Run the repo merge scripts or make targets that produce merged OpenAPI output
   - Ensure the unified catalog spec includes all plugin paths and schemas
3. Validate
   - Run the repo OpenAPI validation target
   - Confirm operationIds are unique
   - Confirm schema names do not collide after prefixing
4. Backwards compatibility checks
   - Diff existing Model Catalog paths and schemas for unintended changes
   - Confirm no behavior changes for existing endpoints
5. Update tests if needed
   - Ensure contract changes are exercised by at least one test (unit or integration)

## Validation commands
- make gen
- make openapi/validate (or the catalog OpenAPI validation target if different)
- make test

## Output
- What changed (paths, schemas) and why
- Proof: commands executed and results
- A short compatibility note stating what was preserved
