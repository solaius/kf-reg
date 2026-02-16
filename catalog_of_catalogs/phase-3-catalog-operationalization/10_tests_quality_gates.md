# Tests and Quality Gates

_Last updated: 2026-02-16_

## Goals
Increase confidence that Phase 3 delivered real wiring and persistence

## Required test layers
- Unit tests
  - ConfigStore behavior
  - RBAC middleware
  - per-source diagnostics mapping
- Integration tests
  - catalog-server management endpoints with FileConfigStore
  - source refresh for YAML provider
- E2E tests
  - compose stack with UI and CLI smoke tests

## Required CI checks
- golangci-lint passes
- OpenAPI merge and validation passes
- generated code is in sync (no drift)
- tests pass across supported DB backends if relevant

## Definition of Done
- All new code paths covered by tests
- CI enforces generated code sync and OpenAPI validation
- No mock-only tests for behaviors that are required in real mode

## Acceptance Criteria
- AC1: A failing persistence behavior produces a red test
- AC2: OpenAPI drift fails CI
- AC3: E2E smoke tests run in CI or are runnable locally with the same commands
