# 06_claude_code_execution_guide

**Date**: 2026-02-16  
**Audience**: Claude Code (implementation agent)  
**Goal**: Execute Phase 4 in a safe order, finishing each step with verifiable proof

## Execution order

1. Persistence and writable mounts
2. Validation and rollback
3. Apply to refresh feedback loop
4. Health endpoints and probes
5. End-to-end verification

## Working agreements

- Follow PROGRAMMING_GUIDELINES.md for code structure, logging, and tests
- Prefer small PRs that keep the build green
- Add tests as part of each change
- Do not change existing Model Catalog public API paths
- Keep feature flags or config switches for behavior differences between file mode and k8s mode

## Step 1 checklist (persistence)

- Implement SourceConfigStore interface
- Implement FileSourceConfigStore with atomic write plus history
- Implement K8sSourceConfigStore using fake client tests
  - Use a single ConfigMap per plugin (not per-source ConfigMaps)
  - Reconcile ConfigMap into in-memory CatalogSourcesConfig deterministically
- Wire store selection by env var
- Update docker-compose mount to allow writeback only where needed
- Implement SecretRef type and RedactSensitiveProperties for sensitive value handling
- Implement SecurityWarningsLayer (warning-only) for inline credential detection

Proof:
- Run docker compose, edit YAML, restart, change persists
- Sensitive values are redacted in API responses

## Step 2 checklist (validation and rollback)

- Add validate endpoint, wire to store
- Apply enforces validation
- Unknown fields must fail at both framework level (StrictFieldsLayer) and plugin level (ProviderLayer)
- MCP plugin ValidateSource uses KnownFields(true) with mcpServerStrictEntry struct for content blocks
- Implement revision list and rollback endpoints
- UI: validate button and error presentation

Proof:
- invalid YAML rejected, rollback restores previous
- unknown fields in MCP server entries produce validation errors

## Step 3 checklist (apply and refresh feedback)

- Implement refresh endpoints if not already present
- Apply supports refreshAfterApply option
- Persist last refresh status, counts, timestamp to database (catalog_refresh_status table via GORM)
- Refresh status must survive server restarts (loaded from DB, not in-memory)
- UI updates counts immediately after save

Proof:
- Save updates entity count without navigation
- After server restart, refresh status is preserved and visible in ListSources

## Step 4 checklist (health endpoints and probes)

- Add /livez and /readyz
- Add small healthcheck binary to image for docker healthcheck
- Update compose healthcheck to hit /readyz
- Add Kubernetes probe config

Proof:
- stopping DB makes readyz fail and compose shows unhealthy

## Completion promise

Keep iterating until all acceptance criteria in 01-05 are met and the full verification plan passes without manual workarounds.
