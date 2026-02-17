# 05_verification_plan_and_acceptance_matrix

**Date**: 2026-02-16

## Test plan overview

This plan ensures the four steps work together and we do not regress model catalog behavior.

### A. Unit tests

- SourceConfigStore interface
  - file store apply and get
  - file store revision list and rollback
  - k8s store apply and get using fake client
  - k8s store uses a single ConfigMap for all sources (not per-source ConfigMaps)
  - k8s store reconciles ConfigMap into in-memory config deterministically
- YAML validation
  - syntax failures
  - unknown field failures in top-level source config (strict)
  - unknown field failures in plugin-specific content (MCP: mcpServerStrictEntry with KnownFields)
  - plugin semantic failures
  - SecurityWarningsLayer produces warnings (not errors) for inline credentials
  - SecretRef map values do not trigger warnings
- Sensitive value handling
  - RedactSensitiveProperties replaces plain string values for sensitive keys
  - SecretRef-like map values are not redacted
  - Sensitive key patterns match case-insensitively
- Refresh status persistence
  - RefreshStatusRecord is saved to DB after refresh
  - Refresh status is loaded and enriches ListSources responses
  - Refresh status survives simulated server restart (re-read from DB)
- Health endpoints
  - livez always 200 when server running
  - readyz fails when DB is down
  - readyz succeeds when dependencies are up

### B. Integration tests (Docker compose)

Pre-req:
- docker compose -f docker-compose.catalog.yaml up

Cases:
1. Baseline load
   - MCP source loads and entity count is non-zero
2. Edit and apply valid YAML
   - entity count changes as expected
3. Edit invalid YAML
   - apply rejected, no data corruption
4. Edit YAML with unknown fields in MCP server entries
   - apply rejected with validation error identifying the unknown field
5. Rollback
   - previous entity count restored
6. Healthchecks
   - catalog-server marked unhealthy if DB container is stopped
   - recovers when DB restarts (depending on restart policy)
7. Refresh status persistence
   - refresh a source, verify status in ListSources response
   - restart catalog-server container, verify refresh status is preserved from DB
8. Sensitive value handling
   - source with inline credential property returns redacted value on Get
   - source with SecretRef property returns the SecretRef object unredacted

### C. UI verification

- Validate action shows errors and warnings
- Save triggers refresh and the page updates counts and last refresh time
- Rollback (if exposed in UI) works

### D. Cluster verification (k8s store mode)

- Catalog-server serviceaccount can update ConfigMaps
- Apply stores changes in ConfigMap via API
- Restart pods and confirm changes persist
- Probes work

## Acceptance matrix

| Step | Key checks |
|------|------------|
| Persistence | changes persist across restarts in both Docker and cluster; K8s uses single ConfigMap per plugin; sensitive values redacted on Get |
| Validation | invalid input blocked; unknown fields rejected at both framework and plugin level; inline credentials produce warnings suggesting SecretRef |
| Apply and refresh | counts and status update automatically; refresh status persisted to DB and survives restarts |
| Health checks | HTTP-based livez and readyz, correct readiness gating |
