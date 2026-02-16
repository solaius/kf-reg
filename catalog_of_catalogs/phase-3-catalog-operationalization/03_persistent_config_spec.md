# Persistent Source Configuration Spec

_Last updated: 2026-02-16_

## Problem to solve
Phase 2 mutation endpoints update in-memory configuration only
After restart, changes are lost and source status can become inconsistent

## Goal
Implement persistent configuration with safe concurrency and deterministic behavior

## Persistence format
Persist the same “sources.yaml” shape used by the plugin system
Persisted object includes:
- plugin name
- sources
  - id
  - type
  - enabled
  - properties (provider-specific)
  - include/exclude patterns (if supported)
- optional per-plugin config block

## ConfigStore implementations

### S1: FileConfigStore (local dev)
- Reads a YAML file on startup
- Writes back on mutation using atomic write (write temp, fsync, rename)
- Supports optional watch (fsnotify) to trigger reconcile if file changes

### S2: KubernetesConfigMapStore (cluster mode)
- Reads a ConfigMap key (default `sources.yaml`)
- Writes updates by patching the ConfigMap data with resourceVersion checks
- Optional: stores per-source YAML catalogs as additional keys in the same ConfigMap so that new sources do not require new volume mounts
- Emits Kubernetes Events for auditability

## Concurrency
- Use optimistic concurrency
- If the stored snapshot changed since last read, mutation returns 409 conflict with retry guidance

## Reconciliation
- At startup: load persisted snapshot and initialize plugins
- Periodic reconcile loop (default 30s) verifies:
  - in-memory config equals persisted snapshot hash
  - if drift detected, prefer persisted snapshot and re-init affected plugins

## Partial refresh
Implement source-level refresh:
- refresh only the specified source for the plugin
- update source status and diagnostics
- do not require full plugin reload

If source-level refresh is too invasive for all providers, implement:
- plugin-level refresh as baseline
- source-level refresh for YAML provider in Phase 3
- document provider-by-provider support

## Rate limiting
Protect refresh endpoints with a simple token bucket:
- per plugin and per source
- default: max 1 refresh per source per 30s
Return 429 with retry-after when exceeded

## Definition of Done
- Config changes persist across restarts in FileConfigStore and KubernetesConfigMapStore modes
- Conflicts are handled via 409 with no silent overwrites
- Source-level refresh works for YAML provider and updates diagnostics without full server restart
- Refresh rate limits enforced and tested
- Audit logs written for all mutations

## Acceptance Criteria
- AC1: Add a source via CLI, restart server, source remains present and enabled
- AC2: Toggle enablement via UI, restart server, enablement remains correct
- AC3: Two concurrent updates produce a conflict for the second writer
- AC4: Refresh a single YAML source and observe entity counts change without affecting other sources
- AC5: Refresh spam is blocked with 429 and retry-after
