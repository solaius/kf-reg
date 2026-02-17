# Universal Action Model

## Purpose
Standardize “verbs” (actions) so:
- UI can render an action bar and forms dynamically
- CLI can offer consistent subcommands across plugins
- Plugins can opt into a shared action set without re-implementing UX/CLI

## Design principles
- Discovery-first: UI/CLI learn what actions exist from capabilities
- Schema-driven: each action declares its input/output schema
- Safe: action execution must support validation, dry-run, and auditing
- Additive: existing endpoints remain unchanged; actions are additive

## Action scopes
1) Source actions (operate on a plugin source)
- validate
- apply
- enable
- disable
- refresh

2) Asset actions (operate on a single asset)
- tag
- deprecate
- promote
- annotate
- link

## Action definition (capabilities shape)
An action is described as:
- id: string (stable, lowercase, e.g., refresh, deprecate)
- displayName: string
- description: string
- scope: enum [source, asset]
- endpoint: string (relative path)
- inputSchemaRef: OpenAPI component ref (or embedded JSON schema)
- outputSchemaRef: OpenAPI component ref (or embedded JSON schema)
- supportsDryRun: bool
- supportsValidateOnly: bool
- idempotency: enum [idempotent, non_idempotent]
- authz:
  - requiredRoles: list[string] (or permissions)
  - auditCategory: string

## Standard endpoints
### Source actions
POST /api/<plugin>_catalog/<ver>/sources/{sourceId}:action

### Asset actions
POST /api/<plugin>_catalog/<ver>/<entities>/{name}:action

Body:
{
  "action": "refresh",
  "dryRun": false,
  "params": { ... }
}

## Async actions
Refresh may become async for large sources.
Standardize:
- Response may include jobId
- Standard job query endpoint:
  GET /api/catalog_management/v1alpha1/jobs/{jobId}

## Acceptance Criteria
- AC1: Capabilities expose at least baseline source actions for every plugin that supports sources
- AC2: Model and MCP expose and implement: refresh (source), tag (asset), deprecate (asset), annotate (asset)
- AC3: CLI can execute any action solely from capabilities + OpenAPI, without plugin-specific code
- AC4: UI can render action forms from schemas and execute successfully
- AC5: Dry-run works where declared

## Definition of Done
- Action schemas exist and are validated in OpenAPI
- Actions appear in plugin capabilities and are discoverable via /api/plugins
- Contract tests validate actions for model/mcp/knowledge sources
- Audit log events emitted for action executions

## Verification plan
- Unit: action handler routing, schema validation
- Integration: execute actions against a running server with seeded data
- UI: e2e test executes refresh and deprecate for mcp and verifies UI state changes
- CLI: golden test for action execution outputs; error cases and dry-run
