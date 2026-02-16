# Phase 3: Catalog Operationalization

_Last updated: 2026-02-16_

## What Phase 3 is
Phase 3 turns the Phase 2 UX and management surfaces into a real, end-to-end working system

Key theme: remove placeholders and mock data from the “happy path” and make the catalog management and MCP catalog experiences operate against a real catalog-server, with persisted configuration and repeatable verification

## Phase 3 outcomes
- A working catalog management plane that persists plugin + source configuration and survives restarts
- A working MCP catalog with real data sources (local images and remote endpoints) loaded via providers, not hardcoded mocks
- A CLI that can manage plugins and sources and validate the system end to end
- A UI that reflects live plugin, source, entity, and health state and supports the Ops for AI and AI Engineer personas
- Automated verification that proves all of the above in CI (or reproducible local E2E)

## Primary personas
- Ops for AI
  - Configure and govern what sources are available
  - Diagnose load errors and health
  - Trigger refresh and manage enablement
- AI Engineer
  - Browse and search assets
  - Inspect details to decide what to use
  - Understand whether an asset is local vs remote and what it requires

## Non-goals for Phase 3
- Full “registry” lifecycling for assets (versioning, promotions, governance workflows)
- Deployment of assets into runtime environments (beyond describing local vs remote MCP and showing image or endpoint metadata)
- Multi-tenant org governance policy engine (keep RBAC scoped to roles and basic gates)

## Milestones
- M3.1 Persistent config and reconciliation
- M3.2 Remove mocks from the default path, add E2E runtime
- M3.3 MCP catalog real data and provider validation
- M3.4 UI: live management + live browsing for Ops and Engineers
- M3.5 CLI: live management + diagnostics
- M3.6 Tests, CI, and doc hardening
