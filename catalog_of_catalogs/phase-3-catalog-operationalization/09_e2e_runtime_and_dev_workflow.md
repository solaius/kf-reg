# E2E Runtime and Developer Workflow

_Last updated: 2026-02-16_

## Goals
Provide a repeatable way to run and verify the system end to end with real wiring

## E2E setup requirements
- A Docker Compose (or equivalent) setup that brings up:
  - catalog-server with model and mcp plugins
  - a persistent config store (file or ConfigMap simulator)
  - BFF in real mode
  - frontend UI
- A single “make e2e” command that:
  - starts the stack
  - runs a small smoke test suite
  - prints URLs for UI and API

## Smoke tests
At minimum:
- plugins list includes model and mcp
- sources list includes configured sources
- MCP list returns entries
- apply source then restart then sources still present
- RBAC: viewer gets 403 for mutation endpoints

## Definition of Done
- New contributor can run E2E locally without editing code
- Smoke tests pass reliably

## Acceptance Criteria
- AC1: `make e2e` brings up UI and both catalog pages work in a browser
- AC2: `make e2e-test` runs smoke tests and exits 0
- AC3: Restarting services does not lose applied configuration
