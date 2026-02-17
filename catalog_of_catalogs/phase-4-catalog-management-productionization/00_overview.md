# Phase 4: Catalog Management Productionization Specs

**Date**: 2026-02-16  
**Scope**: Four hardening steps that make Catalog Management safe, persistent, and truly operational across dev (Docker) and cluster (Kubernetes/OpenShift)

## Context

Phase 3 got us to a working end-to-end stack:
- catalog-server runs with PostgreSQL and loads real MCP data from YAML
- BFF points to real catalog-server endpoints
- UI can view and edit YAML content for a source (write-back to file) but still has gaps around persistence, validation, refresh feedback, and health checks

This Phase 4 is about removing the remaining "prototype behavior" and making the management experience dependable.

## Outcomes (what Phase 4 delivers)

- Operators can edit a source safely without breaking the catalog
- Changes persist correctly in both Docker dev and cluster deployments
- After saving, the system refreshes data and the UI immediately reflects reality (entity counts, status, last refresh)
- Platform health checks use real HTTP endpoints and real readiness semantics

## Deliverables

1. Writable persistence layer for management edits (dev and cluster)
2. Validation plus safety plus rollback for YAML edits and source changes
3. Apply to refresh to UI feedback loop with accurate counts and status
4. Real HTTP health endpoints and probes for Docker and Kubernetes

## Review Clarifications (Design Decisions)

The following design decisions were confirmed during review and are binding for the implementation.

### C1: K8s ConfigMap persistence uses a single ConfigMap

The K8s ConfigMap store (`pkg/catalog/plugin/k8s_config_store.go`) stores all source configuration in a single ConfigMap (one per plugin), not one ConfigMap per source. The catalog-server must reconcile the ConfigMap content into an in-memory `CatalogSourcesConfig` view deterministically on startup and after every Save.

### C2: Refresh status is persisted to the database

Refresh metadata (`lastRefreshTime`, `lastRefreshStatus`, `lastRefreshSummary`, `lastError`, entity counts, duration) is persisted to a `catalog_refresh_status` database table via GORM, not stored in-memory. This ensures refresh status survives server restarts. The table is auto-migrated on startup. See `pkg/catalog/plugin/refresh_status.go`.

### C3: SecretRef framework for sensitive values

Sensitive values must be referenced via `SecretRef` (Name, Namespace, Key), never inlined as plain strings. The `RedactSensitiveProperties()` function in `pkg/catalog/plugin/redact.go` redacts properties with sensitive key patterns (password, token, secret, apikey, api_key, credential) before returning data via the API. The `SecurityWarningsLayer` in `pkg/catalog/plugin/validator.go` produces warnings (not errors) when sensitive values are inlined, guiding operators toward SecretRef usage without blocking saves.

### C4: Strict plugin content validation rejects unknown fields

Unknown fields in both `sources.yaml` and plugin-specific config blocks must produce validation errors. The MCP plugin's `ValidateSource()` in `catalog/plugins/mcp/management.go` uses `yaml.NewDecoder` with `KnownFields(true)` and a strict struct (`mcpServerStrictEntry` with all 19 known fields) to detect unknown fields. Plugin-specific validation is executed via the `ProviderLayer` in the multi-layer validator pipeline.

## Non-goals (explicitly out of scope)

- New asset types beyond MCP and Model
- Replacing the plugin architecture or catalog-gen scaffolding
- Advanced long-running job orchestration (a simple async job is OK if needed)
- Multi-tenant RBAC beyond the current management-plane roles

## Definition of Done (Phase 4)

- All acceptance criteria in the step documents are met
- E2E verification runs in Docker compose with real editing plus refresh plus UI updates
- Cluster mode persists config via Kubernetes API (no attempts to write to ConfigMap volume mounts)
- Health probes are HTTP-based and reflect actual readiness
- Changes follow repo coding guidelines in PROGRAMMING_GUIDELINES.md
