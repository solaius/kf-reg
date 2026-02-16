# Open Questions, Risks, and Decisions

_Last updated: 2026-02-16_

## Decisions needed early
- Persistence store baseline for upstream
  - file-only vs ConfigMap support
- Scope of partial refresh
  - YAML-only vs all providers

## Risks
- Provider reload semantics differ across plugins, making source-level refresh inconsistent
- ConfigMap write permissions may be undesirable in some deployments
- UI requires stable status fields and may block on server implementation details

## Mitigations
- Implement source-level refresh for YAML provider as Phase 3 baseline
- Keep ConfigStore interface so deployments can choose FileConfigStore
- Keep management API stable and additive

## Out of scope but important backlog
- Slack MCP listing once a supported enterprise local or approved remote integration exists
- Asset deployment flows (install local MCP into namespace)
- Deeper policy and governance

## Definition of Done
- All open questions either decided or explicitly deferred with documented rationale
- Risks have owners and mitigation plans
