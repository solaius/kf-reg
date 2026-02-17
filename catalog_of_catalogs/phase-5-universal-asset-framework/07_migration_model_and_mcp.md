# Phase 5 Migration: Model and MCP plugin updates

## Purpose
Ensure existing plugins adopt Phase 5 standards so the universal UI/CLI works.

## Required updates
1) Capabilities
- Provide PluginCapabilities docs for model and mcp

2) Universal asset fields
- Ensure required universal metadata/status fields are present or projectable

3) Actions
- Implement baseline asset actions: tag, annotate, deprecate
- Implement baseline source actions: refresh, enable/disable (if applicable)

4) UI hints
- Provide display fields and sections
- Ensure filterFields align to server-side filterQuery

## Backward compatibility rules
- Existing API paths remain unchanged
- New action endpoints are additive
- New fields are additive

## Acceptance Criteria
- AC1: Model and MCP render fully in universal UI
- AC2: Model and MCP operate fully in CLI v2
- AC3: No regressions in existing UI flows
- AC4: Contract tests prove no breaking changes

## Definition of Done
- Capabilities returned for model and mcp
- Actions implemented and tested
- UI/CLI use only capabilities

## Verification plan
- Regression: existing model catalog calls still work
- Universal: generic UI + CLI tests pass for model and mcp
