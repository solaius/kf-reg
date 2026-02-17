# Claude Code execution guide for Phase 6

## Goal
Implement Phase 6 in a way that is:
- schema-first
- plugin-driven
- provider reusable
- test-driven
- safe and observable

## Operating constraints
- Do not hardcode plugin-specific UI pages or CLI commands for standard flows
- All new plugins must conform to the universal asset contract and action model
- All new plugins must pass the conformance suite
- No secrets in repo or catalogs, only secret references

## Step-by-step execution plan
### Step 1: Deep research per plugin
For each plugin spec:
- Complete the "Deep research outputs" section
- Produce a short "research summary" artifact:
  - sources consulted
  - key fields discovered
  - schema draft adjustments

Deliverable
- Update the plugin spec doc with the research summary and the final schema draft

### Step 2: Finalize catalog.yaml per plugin
- Translate schema draft into catalog.yaml fields
- Identify:
  - filterable fields
  - artifact fields
  - linkable references
- Decide provider support flags (YAML, Git, HTTP, OCI)

Deliverable
- catalog.yaml committed for each plugin

### Step 3: Generate plugin scaffolding and wire into server
- Run catalog-gen for each plugin
- Ensure OpenAPI merge and validation passes
- Ensure database migrations are generated and applied

Deliverable
- Generated code committed
- Server compiles and runs locally

### Step 4: Implement providers
- YAML provider must work for all plugins
- Implement HTTP provider in shared provider library
- Implement Git provider in shared provider library
- Implement OCI provider in shared provider library for at least one plugin

Deliverable
- Provider contract tests
- Integration tests for at least one plugin per provider type

### Step 5: Implement action flows
For each plugin:
- validate, apply, refresh must be implemented and wired
- Default policy: validate must pass before apply unless forced and audited

Deliverable
- Actions conformance tests

### Step 6: Seed real catalogs and prove exit criteria
- Create at least four additional asset catalogs from real sources
- Ensure UI and CLI can list, filter, and manage those assets

Deliverable
- E2E test script or documented manual runbook that reproduces exit criteria

### Step 7: Audit Model, MCP, Knowledge Sources schemas
- Implement required schema deltas
- Ensure conformance suite passes

Deliverable
- Updated schemas and migrations
- Regression tests

### Step 8: Documentation
- Update developer docs:
  - how to add a plugin
  - how to add a provider
  - how to write catalogs for each asset type

Deliverable
- Docs committed and referenced in README or developer guide

## Definition of done
- Exit criteria for Phase 6 met
- Conformance suite passes for all shipped plugins and providers
- At least one new plugin can be added with:
  - catalog.yaml
  - fixtures
  - no UI or CLI code changes

## Optional: coding guidelines
If a repo-level programming guidelines file exists, treat it as authoritative for style and review expectations.
