# 02 Plugin conformance suite

## Objective

Define the contract that every “supported” plugin must satisfy.
Conformance is the gate that makes ecosystem scale possible.

## Supported plugin bar (MUST)

A supported plugin MUST:

1. Expose capabilities + UI hints endpoints
2. Implement list/get endpoints with:
   - consistent pagination envelope
   - filterQuery support (or explicit “not supported” declared in capabilities)
   - stable ordering
3. Implement sources management verbs (as applicable):
   - validate, apply, enable, disable, refresh
4. Emit audit events for state-changing operations
5. Enforce tenancy + RBAC consistently
6. Provide OpenAPI that merges cleanly into the unified catalog spec

## Conformance harness design

### Where tests live

- In a shared repo folder (e.g., `pkg/catalog/conformance/`)
- Plugin repos import the harness and provide:
  - plugin name
  - base URL
  - fixture data sources
  - expected capabilities

### How tests run

- `make conformance` in the plugin directory runs:
  1. spin up catalog-server with plugin enabled
  2. load fixture sources for the plugin
  3. run conformance suite:
     - API contract tests
     - negative tests
     - multi-tenant isolation tests (if enabled)
     - RBAC allow/deny matrix tests

### Required fixture format

Each plugin must provide fixture sources:
- minimal dataset
- medium dataset (for pagination)
- invalid dataset (for validation tests)

## Test categories

### A. Capability contract
- capabilities endpoint returns valid schema
- UI hints schema validates

### B. List/get contract
- pagination works (no duplicates, stable ordering)
- filterQuery behavior matches contract
- get-by-id/name works

### C. Sources verbs
- validate returns errors for invalid sources
- apply persists expected sources state
- refresh updates catalog results

### D. Security
- tenant pre-filter cannot be bypassed
- RBAC deny paths return 403 and consistent error envelope

### E. Observability
- audit events emitted for each management action
- job status endpoints work if refresh is async

### F. OpenAPI merge
- plugin openapi compiles
- unified openapi generation succeeds

## Definition of Done

- Conformance suite exists and is runnable in CI
- Each core plugin (model, mcp, knowledge sources) passes conformance
- Conformance output produces an artifact report (json + human readable)

## Acceptance Criteria

- A third-party team’s plugin is accepted as “supported” only by passing conformance + required governance checks
- Failing conformance produces actionable errors (not “mysterious 500”)

## Verification plan

- Add a toy plugin and intentionally break:
  - pagination
  - tenant filter
  - audit emission
  Ensure conformance catches each failure.
