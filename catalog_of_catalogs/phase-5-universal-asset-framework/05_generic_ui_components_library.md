# Generic UI Components Library (Phase 5)

## Purpose
Create a generic UI layer that renders any plugin/entity using:
- Capabilities document (primary driver)
- OpenAPI schemas (validation and type information)
- Standard endpoints (list/get/artifacts/actions)

## Core components
- ListView (columns + filters)
- FilterBar (capabilities-driven)
- DetailPage (sections + field widgets)
- ArtifactPanel (if supported)
- ActionBar (actions + dialogs)
- DiagnosticsPanel (status, last refresh, warnings/errors)

## Core screens
1) Catalog Home
2) Plugin Entity List
3) Entity Detail
4) Sources Management (per plugin)

## Persona differences
Ops for AI:
- Sources management, refresh, diagnostics, revisions/rollback, health
AI Engineer:
- Asset discovery, details, artifacts, linking, tagging

## Acceptance Criteria
- AC1: Plugin appears in nav based solely on capabilities
- AC2: List view works for model, mcp, and knowledge sources with no plugin-specific code
- AC3: Filters render from filterFields and produce filterQuery correctly
- AC4: Actions render from action definitions, validate, and execute
- AC5: Artifacts panel renders only when supported

## Definition of Done
- Generic components exist with component tests
- No plugin-specific branches required to render model/mcp/knowledge sources
- UI e2e tests cover list, get, filter, action execution, refresh status display

## Verification plan
- Component tests: render list and detail with mocked capabilities/data
- Integration tests: run UI against local docker stack seeded with sample data
- E2E: playwright/cypress tests for model+mcp+knowledge sources flows
