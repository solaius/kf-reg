# Testing, acceptance criteria, and release approach

## Test strategy
- Unit tests for framework utilities (loader, provider registry, filtering)
- Plugin contract tests verifying lifecycle methods and health behavior
- Integration tests for:
  - sources.yaml parsing
  - migrations for multiple plugins
  - end-to-end ingestion to DB to API
- OpenAPI validation tests ensuring merged spec is correct and deterministic

## Acceptance criteria
- AC1: catalog-server can start with model plugin plus one new plugin enabled
- AC2: the new plugin supports ingestion from a YAML source and serves list and get endpoints
- AC3: /api/plugins lists both plugins and reports health
- AC4: Merged OpenAPI spec includes paths and schemas from both plugins without collisions
- AC5: Existing model catalog API endpoints remain unchanged in behavior
- AC6: Docs exist describing how to add a new plugin and how to wire generic UI and CLI support

## Release approach
- Ship incremental PRs that keep main branch buildable and testable
- Prefer feature-flag or config-gated behavior over large refactors
- Ensure any new generated code is committed and validated in CI

## Definition of Done
- All acceptance criteria met
- Documentation updated and aligned with current repository patterns
- CI green across supported DB backends

Date of this spec pack: 2026-02-15
