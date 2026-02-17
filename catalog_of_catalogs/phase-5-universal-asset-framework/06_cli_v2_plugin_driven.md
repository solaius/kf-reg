# CLI v2: Plugin-driven Catalog CLI

## Purpose
Provide a consistent CLI surface that:
- discovers plugins/entities/actions at runtime
- offers consistent output formatting and pagination
- supports both Ops and AI Engineer workflows
- avoids plugin-specific code where possible

## CLI design (conceptual)
Base command: catalogctl (name placeholder)
- catalogctl plugins list
- catalogctl <plugin> <entityPlural> list
- catalogctl <plugin> <entityPlural> get <name>
- catalogctl <plugin> sources list
- catalogctl <plugin> sources validate/apply/enable/disable/refresh
- catalogctl <plugin> <entityPlural> action <actionId> [--params file|json] [--dry-run]

## Discovery mechanism
- CLI calls GET /api/plugins (or /api/capabilities)
- Generates command tree dynamically

## Output modes
- --output json|yaml|table
- Table defaults come from capabilities.columns

## Acceptance Criteria
- AC1: CLI can list plugins and entities from a fresh server
- AC2: CLI can list/get for model and mcp using generic paths
- AC3: CLI can execute actions defined in capabilities (refresh, deprecate)
- AC4: A new plugin appears with commands without CLI changes
- AC5: Integration tests validate table formatting, pagination, and error handling

## Definition of Done
- CLI v2 shipped with plugin discovery
- No per-plugin code required for common flows
- Golden tests exist for plugins list, entity list/get, action execution

## Verification plan
- Unit: command generation from capabilities
- Integration: run CLI against docker stack and compare outputs to golden files
