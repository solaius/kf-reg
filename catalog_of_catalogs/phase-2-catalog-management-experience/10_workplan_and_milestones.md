# Workplan and Milestones

This workplan is intentionally outcome based. Tasks are grouped by thin slices that produce usable functionality.

## Milestone 2.1: Management plane MVP

Outcomes

- API contract for sources management and validation
- Server implementation for list, validate, apply, enable, disable, refresh trigger
- Status endpoint exposes per source refresh state

Deliverable demo

- Operator adds a new MCP YAML source, validates, applies, refreshes, and sees assets

## Milestone 2.2: CLI MVP

Outcomes

- CLI can list plugins, list and get entities, list sources, validate and apply config, trigger refresh
- JSON output for scripting
- Integration tests against a local dev deployment

Deliverable demo

- Same ops flow as UI demo but fully from CLI

## Milestone 2.3: UI MVP

Outcomes

- Plugin switcher driven by discovery endpoint
- Generic list and detail rendering
- Sources management screens
- Status and diagnostics screens

Deliverable demo

- Viewer browses assets and filters
- Operator manages sources and refresh

## Milestone 2.4: Extensibility and hardening

Outcomes

- Documented pattern for plugin UI hints
- Documented pattern for plugin CLI hints
- Hook or template approach for custom loader behavior
- Test coverage for schema changes and backward compatibility

Deliverable demo

- Add a new generated plugin type and see it appear in UI and CLI with generic rendering

## Milestone 2.5: Release readiness

Outcomes

- Full docs for operators and developers
- Upgrade and backward compatibility checks
- End to end tests in CI
- Security review for management endpoints

## Dependencies and assumptions

- Plugin architecture PR is merged or at least stable enough to build on
- BFF and UI can rely on plugin discovery endpoints
- Schema merge pipeline is stable

