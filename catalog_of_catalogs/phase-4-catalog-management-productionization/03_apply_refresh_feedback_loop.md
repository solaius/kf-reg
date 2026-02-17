# 03_apply_refresh_feedback_loop

**Date**: 2026-02-16  
**Owner**: catalog-server plus BFF plus UI plus CLI  
**Goal**: After an operator saves changes, the system refreshes and the UI and CLI immediately reflects the true catalog state

## Problem statement

After Save, the operator currently has to navigate away and back to see updated entity counts. Refresh behavior is not consistently surfaced, and there is no clear last refresh feedback loop.

## Requirements

### R1: Make refresh a first-class management action

Define and implement refresh operations:
- Refresh a single source
- Refresh all sources for a plugin

Endpoints (example shape, align to existing Phase 2 contracts):
- POST /api/catalog-management/v1alpha1/plugins/{plugin}/sources/{sourceId}:refresh
- POST /api/catalog-management/v1alpha1/plugins/{plugin}:refresh

Response must include:
- refresh status (success or failure)
- counts loaded (entities, artifacts if relevant)
- timing (startedAt, finishedAt)
- any diagnostics or partial failure detail

### R2: Apply should optionally trigger refresh

Enhance ApplySource to accept an option:
- applyOptions.refreshAfterApply: true|false

Default behavior recommendation:
- UI sets refreshAfterApply=true
- CLI can choose either mode

### R3: Persist and expose source status fields

**Design decision (confirmed in review):** Refresh metadata is persisted to the database, not stored in-memory. This ensures refresh status survives server restarts.

**Database table:** `catalog_refresh_status` (GORM model: `RefreshStatusRecord` in `pkg/catalog/plugin/refresh_status.go`)

| Column | Type | Description |
|--------|------|-------------|
| source_id | string (PK) | Source identifier |
| plugin_name | string (indexed) | Plugin that owns the source |
| last_refresh_time | timestamp | When the last refresh occurred |
| last_refresh_status | string | "success" or "error" |
| last_refresh_summary | string | Human-readable summary (e.g., "Loaded 6 entities") |
| last_error | string | Error message if refresh failed |
| entities_loaded | int | Number of entities loaded |
| entities_removed | int | Number of entities removed |
| duration_ms | int64 | Refresh duration in milliseconds |
| updated_at | timestamp | Auto-updated by GORM |

Behavior:
- Table is auto-migrated on server startup
- A record is upserted after every refresh and apply+refresh operation via `saveRefreshStatus()`
- Records are loaded in `ListSources` to enrich `SourceStatus` fields returned to callers
- If no DB is configured, refresh status operations are no-ops

The UI sources table should show:
- status
- last refreshed
- entity count

### R4: UX feedback loop

UI behavior:
- On Save:
  - show in-page progress state
  - on success, show a toast and update the sidebar details (counts, last refresh)
  - on failure, show errors with a View diagnostics affordance

PatternFly toast guidance should be followed.

### R5: BFF cache invalidation (if applicable)

If the BFF caches plugin or sources responses, it must invalidate cache on:
- Apply
- Refresh
- Rollback

## Implementation notes

### Sync vs async refresh
Two acceptable options:

Option A: Synchronous refresh for dev and small catalogs
- Apply returns only after refresh completes
- Easier to implement
- Might time out for large catalogs

Option B: Async refresh job
- Apply returns a job id
- UI polls GET /jobs/{id} until complete
- Scales better

Choose Option A for Phase 4 unless timeouts are already observed.

### Diagnostics integration
If a refresh fails:
- store per-source failure reason
- provide a diagnostics endpoint or include diagnostics in refresh response

## Acceptance criteria

- Saving a change results in an updated entity count without manual navigation
- A refresh can be triggered explicitly from UI and CLI
- Last refresh status and time are visible in the UI sources list and source details
- Failures are actionable: the UI shows enough detail to troubleshoot without checking server logs first
- After server restart, refresh status is preserved (loaded from `catalog_refresh_status` DB table)
- Refresh status includes entity counts (loaded and removed) and duration

## Definition of Done

- refresh endpoints implemented and wired into MCP plugin and model plugin where relevant
- Apply supports refreshAfterApply option
- Persist last refresh status, counts, timestamp
- UI updates counts and status after save
- CLI supports refresh and displays summary

## References

- Model Registry architecture docs (API server and CLI patterns)  
  https://www.kubeflow.org/docs/components/model-registry/reference/architecture/
