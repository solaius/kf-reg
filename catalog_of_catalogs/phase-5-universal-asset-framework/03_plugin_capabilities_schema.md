# Plugin Capabilities Schema (Drives UI + CLI)

## Purpose
Provide a discovery document that enables:
- UI to render plugin navigation, entity list/detail, actions, and filters
- CLI to generate a consistent command surface at runtime

This mirrors the Kubernetes pattern of publishing:
- a discovery summary (what exists)
- a schema spec (OpenAPI) citeturn2view3

## Endpoint(s)
- GET /api/plugins  (existing) returns plugin list
- GET /api/plugins/{plugin}/capabilities  returns full capabilities document
- Optional: GET /api/capabilities (aggregated) for caching

## CapabilitiesDocument (logical)
- server:
  - version, build, baseUrl
- plugins: array of PluginCapabilities

### PluginCapabilities
- name: string (plugin id, e.g., model, mcp)
- displayName: string
- description: string
- api:
  - group: string (e.g., model_catalog)
  - version: string (e.g., v1alpha1)
  - basePath: string (e.g., /api/mcp_catalog/v1alpha1)
  - openapiSpecUrl: string
- entities: array of EntityCapabilities
- sources: SourceCapabilities (optional; if plugin supports sources)
- actions: list of ActionDefinition (optional)

### EntityCapabilities
- kind, plural, displayName, description
- endpoints: list/get/artifacts
- fields:
  - identityFields
  - displayFields
  - detailSections
  - filterFields
- uiHints:
  - icon, defaultSort, columns
- actions (asset-scoped)

## UI hints strategy
Do not try to build the UI solely from OpenAPI.
Instead:
- OpenAPI remains schema system of record
- Capabilities provide curated UI hints (columns, sections, safe filters)

## Acceptance Criteria
- AC1: /api/plugins/{plugin}/capabilities exists and is stable
- AC2: UI navigation and views are fully driven by capabilities for model, mcp, and knowledge sources
- AC3: CLI v2 uses capabilities to generate commands for model, mcp, and knowledge sources
- AC4: Adding a new plugin with generator-produced capabilities causes it to appear in UI/CLI without code changes
- AC5: Capabilities are versioned (schemaVersion)

## Definition of Done
- Capabilities JSON schema exists and is validated in CI
- Server returns capabilities; BFF passes through or caches them
- UI and CLI have no plugin-specific switch statements for supported plugins
- Contract tests validate capabilities for model/mcp/knowledge sources

## Verification plan
- Unit: capabilities builder, overrides, JSON schema validation
- Integration: server returns capabilities for each plugin
- UI: generic rendering e2e for each plugin
- CLI: dynamic command generation and outputs verified
