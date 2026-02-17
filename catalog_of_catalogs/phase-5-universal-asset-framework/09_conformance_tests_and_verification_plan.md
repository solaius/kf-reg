# Phase 5 Verification Plan + Plugin Conformance Suite

## Purpose
Make Phase 5 test-driven and prevent future plugins from breaking the universal UI/CLI.

## Test layers
1) Unit tests
2) Integration tests (docker stack)
3) UI e2e tests
4) CLI integration tests (golden)

## Plugin conformance requirements
Every plugin MUST:
- expose capabilities
- identify entity kinds and endpoints
- declare filter fields and columns
- declare supported actions (or none)
- include OpenAPI schemas for entities and action inputs/outputs
- include minimal sample data for CI

## Conformance test suite
Shared suite runs against each plugin:
- Capabilities schema checks
- Endpoint reachability
- Universal required fields present
- Filters accepted (or documented)
- Actions execute and match supportsDryRun semantics

## Acceptance Criteria
- AC1: model, mcp, knowledge-sources pass conformance
- AC2: CI fails if plugin breaks universal contract
- AC3: golden files updated intentionally and reviewed

## Definition of Done
- Conformance suite implemented and wired into CI
- Documentation exists for plugin authors

## Verification artifacts
- CI produces per-plugin conformance report + UI e2e + CLI golden diffs
