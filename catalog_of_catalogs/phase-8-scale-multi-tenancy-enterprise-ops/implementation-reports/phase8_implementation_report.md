# Phase 8: Scale, Multi-Tenancy, and Enterprise Ops -- Implementation Report

**Date:** 2026-02-17
**Branch:** `plugin-catalog-gen`
**Status:** Complete

## Summary

Phase 8 transforms the catalog-server from a single-user development tool into a multi-tenant, authorization-enforced, auditable platform service ready for organization-scale deployments. The phase introduces six new packages (`pkg/tenancy`, `pkg/authz`, `pkg/audit`, `pkg/jobs`, `pkg/cache`, `pkg/ha`) that layer cleanly onto the existing plugin framework without breaking backward compatibility.

## Milestones

### M8.1: Tenancy Plumbing

Created the `pkg/tenancy` package providing context, middleware, and resolver for multi-tenant operation.

**Key files:**

| File | Purpose |
|------|---------|
| `pkg/tenancy/config.go` | `TenancyMode` type: `single` (backward compat) or `namespace` (multi-tenant) |
| `pkg/tenancy/context.go` | `TenantContext` struct (Namespace, User, Groups) with context get/set functions |
| `pkg/tenancy/middleware.go` | HTTP middleware that resolves tenant context via `TenantResolver` and injects into request context |
| `pkg/tenancy/resolver.go` | `SingleTenantResolver` (always "default") and `NamespaceTenantResolver` (reads `?namespace=` or `X-Namespace` header, validates K8s DNS label format) |

**Design decisions:**
- Namespace is resolved from query param first, then X-Namespace header (consistent with Kubeflow's proxy model)
- Validation enforces K8s DNS-1123 label rules (lowercase alphanumeric + hyphens, 1-63 chars)
- `SingleTenantResolver` always returns `"default"` namespace, making single-tenant mode a zero-config default

**Tests:** `context_test.go`, `middleware_test.go`, `resolver_test.go`

### M8.2: DB Migrations

Added `namespace` columns to all entity and governance tables to support namespace-scoped data isolation.

**Key changes:**
- Added `namespace` column (with `DEFAULT 'default'` and `NOT NULL`) to all relevant database tables
- Updated uniqueness constraints to include namespace as part of composite keys
- Backfill existing rows to `"default"` namespace during migration
- Two-step migration approach: add column with default, then add constraint

**Design decisions:**
- Non-nullable namespace with default `"default"` ensures backward compatibility for existing single-tenant data
- GORM `AutoMigrate` handles schema evolution; no manual SQL migration files needed

### M8.3: RBAC/Authorization

Created the `pkg/authz` package implementing Kubernetes SubjectAccessReview-based authorization.

**Key files:**

| File | Purpose |
|------|---------|
| `pkg/authz/types.go` | `Authorizer` interface, `AuthzRequest` struct, resource/verb constants, API group `catalog.kubeflow.org` |
| `pkg/authz/config.go` | `AuthzMode`: `none` (dev/backward compat) or `sar` (K8s SAR) |
| `pkg/authz/identity.go` | `IdentityMiddleware` extracts `X-Remote-User` and `X-Remote-Group` headers; defaults to `"anonymous"` |
| `pkg/authz/sar.go` | `SARAuthorizer` creates SubjectAccessReview against K8s API server |
| `pkg/authz/cache.go` | `CachedAuthorizer` wraps any Authorizer with a 10-second TTL in-memory cache |
| `pkg/authz/noop.go` | `NoopAuthorizer` always allows (used with `AuthzModeNone`) |
| `pkg/authz/middleware.go` | `RequirePermission` (per-route) and `AuthzMiddleware` (auto-mapping) middleware |
| `pkg/authz/mapper.go` | `MapRequest` maps HTTP method + URL path to `(resource, verb)` tuples for SAR checks |

**Resource mapping table:**

| Resource | Verbs |
|----------|-------|
| `plugins` | list |
| `capabilities` | get |
| `catalogsources` | get, list, create, update, delete |
| `assets` | get, list, create, update, delete |
| `actions` | list, execute |
| `jobs` | get, list, create |
| `approvals` | list, approve, get |
| `audit` | list, get |

**Design decisions:**
- Authorization is delegated to Kubernetes RBAC via SAR, not reimplemented
- Identity is extracted from standard auth-proxy headers (`X-Remote-User`, `X-Remote-Group`)
- SAR results are cached with a short TTL (default 10s) to reduce API server load
- `AuthzModeNone` provides full backward compatibility for development and single-tenant deployments
- URL-to-resource mapping uses path pattern matching from most specific to least specific

**Tests:** `cache_test.go`, `identity_test.go`, `mapper_test.go`, `middleware_test.go`, `sar_test.go`

### M8.4: Audit V2

Created the `pkg/audit` package providing enhanced audit logging with namespace, actor, and request ID tracking.

**Key files:**

| File | Purpose |
|------|---------|
| `pkg/audit/config.go` | `AuditConfig` (RetentionDays, LogDenied, Enabled) with env var loading |
| `pkg/audit/middleware.go` | `AuditMiddleware` captures audit events after handler completion |
| `pkg/audit/handlers.go` | `ListEventsHandler` and `GetEventHandler` for audit API |
| `pkg/audit/helpers.go` | Path extraction utilities (plugin, resource type, resource IDs, action verb) |
| `pkg/audit/retention.go` | `RetentionWorker` runs daily cleanup of expired audit events |
| `pkg/audit/router.go` | Chi router with optional authz middleware for audit endpoints |

**Audit event schema:** ID, Namespace, CorrelationID, EventType, Actor, RequestID, Plugin, ResourceType, ResourceIDs, Action, ActionVerb, Outcome (success/denied/failure), StatusCode, Reason, OldValue, NewValue, Metadata, CreatedAt

**API endpoints:**
- `GET /api/audit/v1alpha1/events` -- paginated, filterable (namespace, actor, plugin, action, eventType)
- `GET /api/audit/v1alpha1/events/{eventId}` -- single event by ID

**Design decisions:**
- Audit is best-effort: write failures are logged but do not fail the request
- Denied actions can optionally be recorded (controlled by `LogDenied` setting)
- Only management endpoints are audited; pure read endpoints are not
- Retention cleanup runs daily via a background worker
- CorrelationID from `X-Correlation-ID` header enables cross-service tracing

**Tests:** `config_test.go`, `handlers_test.go`, `helpers_test.go`, `middleware_test.go`, `retention_test.go`

### M8.5: Async Refresh Jobs

Created the `pkg/jobs` package implementing a database-backed job queue with worker pool for async refresh operations.

**Key files:**

| File | Purpose |
|------|---------|
| `pkg/jobs/config.go` | `JobConfig` (Concurrency, MaxRetries, PollInterval, ClaimTimeout, RetentionDays) |
| `pkg/jobs/models.go` | `RefreshJob` GORM model with state machine (queued -> running -> succeeded/failed/canceled) |
| `pkg/jobs/store.go` | `JobStore` with Enqueue, Claim (FOR UPDATE SKIP LOCKED), Complete, Fail, Cancel, List, Get |
| `pkg/jobs/worker.go` | `WorkerPool` with configurable concurrency, stuck job recovery, and old job cleanup |
| `pkg/jobs/handlers.go` | `GetJobHandler`, `ListJobsHandler`, `CancelJobHandler` HTTP handlers |
| `pkg/jobs/router.go` | Chi router with optional authz for job status API |

**Job state machine:** `queued` -> `running` -> `succeeded` | `failed` | `canceled`

**API endpoints:**
- `GET /api/jobs/v1alpha1/refresh` -- list jobs (filterable by namespace, plugin, sourceId, state, requestedBy)
- `GET /api/jobs/v1alpha1/refresh/{jobId}` -- get job by ID
- `POST /api/jobs/v1alpha1/refresh/{jobId}:cancel` -- cancel a queued job

**Design decisions:**
- `FOR UPDATE SKIP LOCKED` (PostgreSQL) ensures safe multi-replica job processing without duplicates
- Fallback to plain SELECT for non-PostgreSQL databases
- Idempotency key prevents duplicate job creation for the same refresh request
- Stuck job recovery: running jobs older than `ClaimTimeout` are re-queued automatically
- `PluginRefresher` interface avoids circular dependency between jobs and plugin packages

**Tests:** `config_test.go`, `handlers_test.go`, `models_test.go`, `store_test.go`, `worker_test.go`

### M8.6: Performance Hardening

Created the `pkg/cache` package providing in-memory LRU caching with TTL for discovery endpoints.

**Key files:**

| File | Purpose |
|------|---------|
| `pkg/cache/config.go` | `CacheConfig` (Enabled, DiscoveryTTL, CapabilitiesTTL, MaxSize) with env var loading |
| `pkg/cache/lru.go` | Thread-safe `LRUCache` with TTL-based expiration and oldest-entry eviction |
| `pkg/cache/middleware.go` | `CacheMiddleware` caches GET 200 responses; adds `X-Cache: HIT/MISS` header |
| `pkg/cache/invalidation.go` | `CacheManager` with separate discovery and capabilities caches, per-plugin invalidation |

**Design decisions:**
- Only safe, read-only, low-entropy endpoints are cached: `/api/plugins` and `/api/plugins/{name}/capabilities`
- Tenant-scoped list endpoints are NOT cached to avoid cross-tenant data leakage
- Invalidation is triggered on source apply/refresh completion
- `X-Cache` response header enables monitoring of cache effectiveness
- Separate TTLs for discovery (60s default) and capabilities (30s default)

**Tests:** `invalidation_test.go`, `lru_test.go`, `middleware_test.go`

### M8.7: HA Readiness

Created the `pkg/ha` package providing migration locking and Kubernetes Lease-based leader election.

**Key files:**

| File | Purpose |
|------|---------|
| `pkg/ha/config.go` | `HAConfig` (LeaderElectionEnabled, LeaseName/Namespace/Duration, MigrationLockEnabled, Identity) |
| `pkg/ha/migration_lock.go` | `MigrationLocker` interface, PostgreSQL advisory lock, table-based fallback for other DBs |
| `pkg/ha/leader_election.go` | `LeaderElector` using Kubernetes Lease resources with start/stop callbacks |

**Migration locking strategies:**

| Database | Strategy | Mechanism |
|----------|----------|-----------|
| PostgreSQL | `pgAdvisoryLock` | `pg_advisory_lock(CRC32('catalog-server-migration'))` |
| Other (MySQL, SQLite) | `fallbackMigrationLock` | `migration_lock` table with INSERT-or-fail and stale lock cleanup (5 min) |

**Leader election:**
- Uses standard Kubernetes `LeaseLock` from `k8s.io/client-go`
- Only the leader runs singleton background loops (config reconciliation, audit retention, job workers)
- Non-leaders serve HTTP normally
- Leadership loss triggers `OnStopLeading` callback for graceful handoff
- `ReleaseOnCancel: true` ensures fast failover on graceful shutdown

**Design decisions:**
- Migration locking is enabled by default (`MigrationLockEnabled: true`) -- safe for single and multi-replica
- Leader election is disabled by default (`LeaderElectionEnabled: false`) -- opt-in for HA deployments
- Identity defaults to `POD_NAME` env var or hostname
- `/readyz` fails if migrations are incomplete, preventing traffic to partially-migrated replicas

**Tests:** `config_test.go`, `leader_election_test.go`, `migration_lock_test.go`

### M8.8: UI/CLI/BFF

Extended the BFF layer, React UI, and CLI to support namespace selection and propagation.

**Key changes:**
- BFF proxies `?namespace=` query parameter and `X-Namespace` header to catalog server
- React UI adds namespace selector component (dropdown populated from `/api/tenancy/v1alpha1/namespaces`)
- Selected namespace is stored in browser session and propagated to all API calls
- CLI adds `--namespace` flag to all subcommands, defaults to current kube-context namespace
- catalogctl respects `CATALOG_NAMESPACE` env var as override

### M8.9: E2E, Load, and HA Tests

Comprehensive test suite validating acceptance criteria across all Phase 8 features.

**Test categories:**
- Tenant isolation: cross-namespace visibility and action prevention
- RBAC enforcement: positive (allow) and negative (deny) authorization flows
- Audit completeness: every management action produces exactly one audit event
- Job lifecycle: enqueue, claim, complete, fail, retry, cancel
- Cache behavior: hit/miss, invalidation on apply/refresh
- HA: migration locking under concurrent startup, leader election failover

## Architecture Summary

### Middleware Stack (request processing order)

```
CORS -> Tenancy -> Identity -> Authz -> Audit -> Cache -> Handler
```

1. **CORS** -- standard cross-origin headers
2. **Tenancy** -- resolves namespace from query/header, injects `TenantContext`
3. **Identity** -- extracts user/groups from `X-Remote-User`/`X-Remote-Group` headers
4. **Authz** -- maps request to resource/verb, calls SAR, denies with 403 if unauthorized
5. **Audit** -- captures response status code, writes audit event after handler completes
6. **Cache** -- serves cached responses for discovery/capabilities endpoints
7. **Handler** -- actual endpoint logic

### New Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `pkg/tenancy` | 4 + 3 tests | Tenant context, middleware, resolvers |
| `pkg/authz` | 8 + 5 tests | Authorization: SAR, identity, caching, middleware, mapper |
| `pkg/audit` | 6 + 5 tests | Audit events: middleware, handlers, retention, router |
| `pkg/jobs` | 6 + 5 tests | Async refresh: models, store, worker pool, handlers, router |
| `pkg/cache` | 4 + 3 tests | LRU caching: core, middleware, invalidation, config |
| `pkg/ha` | 3 + 3 tests | HA: migration lock, leader election, config |

### New API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/tenancy/v1alpha1/namespaces` | List namespaces available to user |
| GET | `/api/audit/v1alpha1/events` | List audit events (paginated, filtered) |
| GET | `/api/audit/v1alpha1/events/{id}` | Get audit event by ID |
| GET | `/api/jobs/v1alpha1/refresh` | List refresh jobs (paginated, filtered) |
| GET | `/api/jobs/v1alpha1/refresh/{id}` | Get refresh job by ID |
| POST | `/api/jobs/v1alpha1/refresh/{id}:cancel` | Cancel a queued job |

### New Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_TENANCY_MODE` | `single` | Tenancy mode: `single` or `namespace` |
| `CATALOG_AUTHZ_MODE` | `none` | Authorization mode: `none` or `sar` |
| `CATALOG_AUDIT_RETENTION_DAYS` | `90` | Days to keep audit events |
| `CATALOG_AUDIT_LOG_DENIED` | `true` | Whether to log 403 denied actions |
| `CATALOG_AUDIT_ENABLED` | `true` | Whether audit middleware is active |
| `CATALOG_JOB_CONCURRENCY` | `3` | Max concurrent refresh workers |
| `CATALOG_JOB_MAX_RETRIES` | `3` | Max retry attempts per job |
| `CATALOG_JOB_ENABLED` | `true` | Whether the job system is active |
| `CATALOG_CACHE_ENABLED` | `true` | Whether discovery caching is active |
| `CATALOG_CACHE_DISCOVERY_TTL` | `60` | Discovery cache TTL in seconds |
| `CATALOG_CACHE_CAPABILITIES_TTL` | `30` | Capabilities cache TTL in seconds |
| `CATALOG_CACHE_MAX_SIZE` | `1000` | Max entries per cache instance |
| `CATALOG_LEADER_ELECTION_ENABLED` | `false` | Enable K8s Lease-based leader election |
| `CATALOG_MIGRATION_LOCK_ENABLED` | `true` | Enable DB migration locking |
| `POD_NAME` | hostname | Instance identity for leader election |

## Known Limitations

1. **Namespace listing endpoint** -- `/api/tenancy/v1alpha1/namespaces` currently returns a static or config-driven list; dynamic namespace discovery from Kubernetes is not yet implemented
2. **SAR caching** -- the 10-second TTL means RBAC changes take up to 10 seconds to take effect
3. **Job cancellation** -- running jobs cannot be force-canceled; cancellation is cooperative (only queued jobs are immediately cancelable)
4. **Cross-namespace queries** -- no built-in aggregation across namespaces for non-ops users; each query is scoped to a single namespace
5. **Metrics/tracing** -- structured logging is in place, but Prometheus metrics and OpenTelemetry tracing integration are deferred to a future phase

## Backward Compatibility

- **Single-tenant mode** (`CATALOG_TENANCY_MODE=single`) is the default -- existing deployments continue to work unchanged
- **Authorization disabled** (`CATALOG_AUTHZ_MODE=none`) is the default -- no SAR calls unless explicitly enabled
- **All new columns** have `DEFAULT 'default'` so existing data is automatically backfilled
- **No breaking API changes** -- all new query parameters (`namespace`) are optional in single-tenant mode
- **Migration locking** is enabled by default and is safe for single-replica deployments (noop overhead)
