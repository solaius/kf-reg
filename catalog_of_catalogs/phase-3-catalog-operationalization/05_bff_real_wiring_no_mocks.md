# BFF: Real Wiring, No Mocks in the Default Path

_Last updated: 2026-02-16_

## Problem
Phase 2 added mock clients to enable rapid UI iteration
Phase 3 must make the default runtime path use real integrations

## Goals
- Default mode is real catalog-server integration
- Mock modes remain for unit tests and optional dev flags only
- Remove placeholder data from responses in default mode

## Requirements
- BFF must proxy all management endpoints and MCP entity endpoints to catalog-server
- BFF must not fabricate status, counts, or diagnostics in real mode
- If catalog-server is unavailable, BFF returns a clear error and UI surfaces it

## Modes
- Real mode (default)
  - Requires catalog-server base URL
  - No mock clients enabled
- Mock mode (explicit)
  - gated behind flags (existing flags remain)
  - used only for UI dev without server

## Definition of Done
- `go run ./cmd/` without mock flags runs in real mode and UI renders using real server responses
- Mock data is not used unless explicitly enabled by flags
- Error handling is consistent and user-visible

## Acceptance Criteria
- AC1: Starting BFF in default mode requires catalog-server URL and fails fast if missing
- AC2: UI management page shows real sources and statuses from server
- AC3: Disabling mock flags removes any hardcoded source lists from the UI path
