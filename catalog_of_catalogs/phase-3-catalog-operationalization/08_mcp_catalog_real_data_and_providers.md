# MCP Catalog: Real Data and Provider Wiring

_Last updated: 2026-02-16_

## Goals
- Load MCP server entries from real provider sources, not mocks
- Support both local and remote MCP entries in the schema
- Ensure the MCP gallery and detail pages are driven by real data

## MCP entries in scope
Local:
- Kubernetes
- OpenShift
- Ansible
- Postgres (use image `quay.io/mcp-servers/edb-postgres-mcp`)
Remote:
- GitHub
- Jira

Out of scope for Phase 3:
- Slack (treat as remote and defer until a supported enterprise local option exists)

## Data model requirements
Each MCP entry should include at minimum:
- name
- description
- deploymentMode: local | remote
- transports: list (http-streaming, sse, stdio if used)
- license
- labels (category)
- verification flags (verifiedSource, sast, partner)
- connection info:
  - local: image URI
  - remote: base URL
- optional auth hint (oauth, token, none)

## Provider wiring
- YAML provider must load these entries from a YAML file referenced by sources config
- Source status and diagnostics must reflect YAML parsing errors and validation errors

## Definition of Done
- MCP list and get endpoints return the scoped MCP servers loaded from provider data
- No MCP entries are hardcoded in the server or BFF for real mode
- Source diagnostics show actionable error messages on invalid YAML

## Acceptance Criteria
- AC1: MCP source configured via persisted sources config loads at least 6 MCP entries
- AC2: UI gallery shows these 6 entries with correct local vs remote badges
- AC3: Editing the YAML to add one entry and refreshing the source updates the list without restart
- AC4: Broken YAML produces a failed status and diagnostics contain parse error details
