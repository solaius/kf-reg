# Architecture and Data Flow for Phase 3

_Last updated: 2026-02-16_

## Components
- catalog-server
  - hosts plugin entity APIs
  - hosts plugin management APIs (plugins, sources, refresh, diagnostics)
  - runs plugin loaders and watchers
- BFF
  - proxies UI requests to catalog-server
  - optionally integrates with Kubernetes for auth context and future expansion
- Frontend UI
  - catalog management pages
  - MCP catalog browsing pages
- CLI
  - talks to management APIs and entity APIs

## Phase 3 target runtime
The target runtime is “real server” mode:
- UI and CLI call the BFF (or call catalog-server directly for local testing)
- BFF proxies to catalog-server without mock clients
- catalog-server loads sources from a persisted config store

## Data flow: list entities
UI -> BFF -> catalog-server plugin entity endpoint -> response

## Data flow: manage sources
UI or CLI -> BFF -> catalog-server management endpoint -> persisted config store updated -> plugin reload triggered -> status and diagnostics updated -> UI refresh shows new state

## Persistence design principle
Configuration must have a single source of truth that:
- can be read at startup
- can be written by the server on mutation
- can be observed for drift
- can be reconciled if the in-memory state diverges

Phase 3 defines and implements this persistence as a first-class requirement

## Interfaces (high-level)
- ConfigStore
  - Load: returns full config snapshot (plugins + sources)
  - Save: persists a new snapshot with optimistic concurrency
  - Watch (optional): informs server of external config changes for reconciliation
- SourceRefresher
  - RefreshPlugin(pluginName)
  - RefreshSource(pluginName, sourceID)
- DiagnosticsProvider
  - plugin summary
  - source summary
  - source error details
