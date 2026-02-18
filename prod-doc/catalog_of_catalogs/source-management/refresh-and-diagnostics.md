# Refresh and Diagnostics

## Overview

The refresh and diagnostics subsystem provides **on-demand data reload** and **runtime health introspection** for catalog plugins. Plugins opt into these capabilities by implementing the `RefreshProvider` and `DiagnosticsProvider` interfaces from the plugin framework.

Refresh results are persisted to the database so that status information survives server restarts. A per-source rate limiter prevents refresh storms caused by rapid API calls or automated tooling.

**Location:** `pkg/catalog/plugin/`

## Refresh Flow

```
Operator calls POST /management/refresh/{sourceId}
        |
        v
+-----------------------------------------------+
|         RefreshRateLimiter.Allow()             |
|   key = "pluginName:sourceId"                 |
|   30-second cooldown per source               |
+---------------------+-------------------------+
                      |
          +-----------+-----------+
          |                       |
     [allowed]              [rate limited]
          |                       |
          v                       v
+-----------------+      +------------------+
| RefreshProvider  |      | 429 Too Many     |
| .Refresh(ctx,   |      | Requests         |
|   sourceID)     |      | Retry-After: N   |
+---------+-------+      +------------------+
          |
          v
+-------------------------------------------------+
|         Server.saveRefreshStatus()              |
|   Upsert into catalog_refresh_status table      |
|   (persists across restarts)                    |
+-------------------------------------------------+
          |
          v
+------------------+
| 200 OK           |
| RefreshResult    |
| JSON response    |
+------------------+
```

## RefreshProvider Interface

Plugins implement this optional interface to support on-demand data reloading.

```go
// pkg/catalog/plugin/plugin.go
type RefreshProvider interface {
    // Refresh triggers a reload of a specific source.
    Refresh(ctx context.Context, sourceID string) (*RefreshResult, error)

    // RefreshAll triggers a reload of all sources.
    RefreshAll(ctx context.Context) (*RefreshResult, error)
}
```

When a plugin implements `RefreshProvider`, the framework automatically mounts the refresh endpoints under the plugin's management router.

## RefreshResult

```go
// pkg/catalog/plugin/management_types.go
type RefreshResult struct {
    SourceID        string        `json:"sourceId,omitempty"`
    EntitiesLoaded  int           `json:"entitiesLoaded"`
    EntitiesRemoved int           `json:"entitiesRemoved"`
    Duration        time.Duration `json:"duration"`
    Error           string        `json:"error,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `SourceID` | `string` | Source that was refreshed; empty for refresh-all operations |
| `EntitiesLoaded` | `int` | Number of entities loaded during the refresh |
| `EntitiesRemoved` | `int` | Number of entities removed (stale entries cleaned up) |
| `Duration` | `time.Duration` | Wall-clock time the refresh operation took |
| `Error` | `string` | Error message if the refresh failed; empty on success |

Example JSON response:

```json
{
  "sourceId": "hf-popular",
  "entitiesLoaded": 42,
  "entitiesRemoved": 3,
  "duration": 1250000000,
  "error": ""
}
```

## Refresh Status Persistence

Refresh results are persisted to the `catalog_refresh_status` database table so that status information survives server restarts. The `RefreshStatusRecord` GORM model maps to this table.

```go
// pkg/catalog/plugin/refresh_status.go
type RefreshStatusRecord struct {
    SourceID           string     `gorm:"primaryKey;column:source_id"`
    PluginName         string     `gorm:"column:plugin_name;index"`
    LastRefreshTime    *time.Time `gorm:"column:last_refresh_time"`
    LastRefreshStatus  string     `gorm:"column:last_refresh_status"`  // "success", "error"
    LastRefreshSummary string     `gorm:"column:last_refresh_summary"` // e.g. "Loaded 6 entities"
    LastError          string     `gorm:"column:last_error"`
    EntitiesLoaded     int        `gorm:"column:entities_loaded"`
    EntitiesRemoved    int        `gorm:"column:entities_removed"`
    DurationMs         int64      `gorm:"column:duration_ms"`
    UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (RefreshStatusRecord) TableName() string {
    return "catalog_refresh_status"
}
```

### Database Schema

| Column | Type | Description |
|--------|------|-------------|
| `source_id` | `string` (PK) | Unique source identifier; `_all` for refresh-all operations |
| `plugin_name` | `string` (indexed) | Plugin that owns the source |
| `last_refresh_time` | `timestamp` | When the last refresh completed |
| `last_refresh_status` | `string` | `"success"` or `"error"` |
| `last_refresh_summary` | `string` | Human-readable summary (e.g., `"Loaded 6 entities"`) |
| `last_error` | `string` | Error message from the last failed refresh |
| `entities_loaded` | `int` | Entity count from the last refresh |
| `entities_removed` | `int` | Removed entity count from the last refresh |
| `duration_ms` | `int64` | Refresh duration in milliseconds |
| `updated_at` | `timestamp` | Auto-updated on each write |

### Persistence Operations

The `Server` exposes internal methods for managing refresh status records:

```
saveRefreshStatus(pluginName, sourceID, result)     Upsert a record after refresh
getRefreshStatus(pluginName, sourceID)               Load a single record
listRefreshStatuses(pluginName)                      Load all records for a plugin
deleteRefreshStatus(pluginName, sourceID)             Remove record when source is deleted
deleteAllRefreshStatuses(pluginName)                  Remove all records for a plugin
```

When the sources list endpoint is called, persisted refresh statuses are merged into the returned `SourceInfo` objects. The persisted values fill in any gaps left by the in-memory status (e.g., after a server restart when in-memory state has been lost).

### Summary Formatting

The `formatRefreshSummary` helper produces human-readable text:

| Condition | Summary |
|-----------|---------|
| Error present | `"Refresh failed"` |
| Removals > 0 | `"Loaded N entities, removed M"` |
| Success | `"Loaded N entities"` |

## Rate Limiting

The `RefreshRateLimiter` prevents refresh storms by enforcing a minimum interval between refresh calls for the same source. Each `(plugin, sourceID)` pair gets an independent token bucket.

```go
// pkg/catalog/plugin/rate_limiter.go
type RefreshRateLimiter struct {
    mu       sync.Mutex
    buckets  map[string]*bucket
    interval time.Duration   // minimum time between refreshes (default: 30s)
}

func NewRefreshRateLimiter(interval time.Duration) *RefreshRateLimiter
func (rl *RefreshRateLimiter) Allow(key string) (bool, time.Duration)
func (rl *RefreshRateLimiter) Reset()
```

### Key Construction

```go
// Per-source key:    "mcp:my-source-id"
func RefreshKey(pluginName, sourceID string) string {
    return pluginName + ":" + sourceID
}

// Refresh-all key:   "mcp:*"
func RefreshAllKey(pluginName string) string {
    return pluginName + ":*"
}
```

### Behavior

```
Time 0s    POST /management/refresh/src-1   --> 200 OK (bucket created)
Time 10s   POST /management/refresh/src-1   --> 429 Too Many Requests
                                                 Retry-After: 20
Time 30s   POST /management/refresh/src-1   --> 200 OK (cooldown expired)
Time 30s   POST /management/refresh/src-2   --> 200 OK (independent bucket)
```

When a request is rate limited, the handler returns HTTP 429 with a `Retry-After` header indicating how many seconds remain before the next allowed attempt:

```go
func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
    seconds := int(math.Ceil(retryAfter.Seconds()))
    w.Header().Set("Retry-After", strconv.Itoa(seconds))
    writeError(w, http.StatusTooManyRequests,
        fmt.Sprintf("rate limited, retry after %d seconds", seconds), nil)
}
```

## DiagnosticsProvider Interface

Plugins implement this optional interface to expose runtime health and per-source status.

```go
// pkg/catalog/plugin/plugin.go
type DiagnosticsProvider interface {
    // Diagnostics returns diagnostic information about the plugin.
    Diagnostics(ctx context.Context) (*PluginDiagnostics, error)
}
```

The diagnostics endpoint is read-only and available to all roles (including viewers).

## PluginDiagnostics

```go
// pkg/catalog/plugin/management_types.go
type PluginDiagnostics struct {
    PluginName  string             `json:"pluginName"`
    Sources     []SourceDiagnostic `json:"sources"`
    LastRefresh *time.Time         `json:"lastRefresh,omitempty"`
    Errors      []DiagnosticError  `json:"errors,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `PluginName` | `string` | Name of the plugin reporting diagnostics |
| `Sources` | `[]SourceDiagnostic` | Per-source health and status details |
| `LastRefresh` | `*time.Time` | When any source in this plugin was last refreshed |
| `Errors` | `[]DiagnosticError` | Active plugin-level or source-level errors |

## SourceDiagnostic

```go
// pkg/catalog/plugin/management_types.go
type SourceDiagnostic struct {
    ID                  string         `json:"id"`
    Name                string         `json:"name"`
    State               string         `json:"state"`
    EntityCount         int            `json:"entityCount"`
    LastRefreshTime     *time.Time     `json:"lastRefreshTime,omitempty"`
    LastRefreshDuration *time.Duration `json:"lastRefreshDuration,omitempty"`
    Error               string         `json:"error,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Source identifier |
| `Name` | `string` | Human-readable source name |
| `State` | `string` | Current state: `available`, `error`, `disabled`, `loading` |
| `EntityCount` | `int` | Number of entities currently loaded from this source |
| `LastRefreshTime` | `*time.Time` | When this source was last refreshed |
| `LastRefreshDuration` | `*time.Duration` | How long the last refresh took |
| `Error` | `string` | Last error message for this source, if any |

## DiagnosticError

```go
// pkg/catalog/plugin/management_types.go
type DiagnosticError struct {
    Source  string    `json:"source,omitempty"`
    Message string    `json:"message"`
    Time    time.Time `json:"time"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Source` | `string` | Source ID where the error occurred; empty for plugin-level errors |
| `Message` | `string` | Error description |
| `Time` | `time.Time` | When the error occurred |

### Example Diagnostics Response

```json
{
  "pluginName": "mcp",
  "sources": [
    {
      "id": "filesystem",
      "name": "Filesystem MCP Server",
      "state": "available",
      "entityCount": 1,
      "lastRefreshTime": "2026-02-17T10:30:00Z",
      "lastRefreshDuration": 45000000
    },
    {
      "id": "broken-src",
      "name": "Broken Source",
      "state": "error",
      "entityCount": 0,
      "error": "connection refused"
    }
  ],
  "lastRefresh": "2026-02-17T10:30:00Z",
  "errors": [
    {
      "source": "broken-src",
      "message": "connection refused",
      "time": "2026-02-17T10:25:00Z"
    }
  ]
}
```

## Management Endpoints

All refresh and diagnostics endpoints are mounted under the plugin's management router at `{basePath}/management/`.

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| `POST` | `/management/refresh` | `operator` | Refresh all sources for the plugin |
| `POST` | `/management/refresh/{sourceId}` | `operator` | Refresh a specific source by ID |
| `GET` | `/management/diagnostics` | `viewer` | Retrieve plugin diagnostics and per-source health |

### Route Mounting

The management router automatically registers refresh and diagnostics routes when the plugin implements the corresponding interfaces:

```go
// pkg/catalog/plugin/management_handlers.go
func managementRouter(p CatalogPlugin, roleExtractor RoleExtractor, srv *Server) chi.Router {
    r := chi.NewRouter()

    // Refresh (requires RefreshProvider)
    if rp, ok := p.(RefreshProvider); ok {
        r.Post("/refresh", refreshAllHandler(rp, rl, pluginName, srv))
        r.Post("/refresh/{sourceId}", refreshSourceHandler(rp, rl, pluginName, srv))
    }

    // Diagnostics (read-only, available to viewers)
    if dp, ok := p.(DiagnosticsProvider); ok {
        r.Get("/diagnostics", diagnosticsHandler(dp))
    }

    return r
}
```

### Endpoint Details

**POST /management/refresh** -- Triggers `RefreshAll()` on the plugin. Rate-limited using the key `pluginName:*`. The refresh result is persisted with the synthetic source ID `_all`.

**POST /management/refresh/{sourceId}** -- Triggers `Refresh(ctx, sourceID)` for a single source. Rate-limited using the key `pluginName:sourceId`. Returns 429 with `Retry-After` header if the cooldown has not elapsed.

**GET /management/diagnostics** -- Calls `Diagnostics()` on the plugin. Read-only endpoint, no role restriction beyond viewer. Returns the full `PluginDiagnostics` response including per-source state, entity counts, and active errors.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/management_types.go` | `RefreshResult`, `PluginDiagnostics`, `SourceDiagnostic`, `DiagnosticError` type definitions |
| `pkg/catalog/plugin/management_handlers.go` | HTTP handlers for refresh and diagnostics endpoints, rate-limit integration, status persistence calls |
| `pkg/catalog/plugin/rate_limiter.go` | `RefreshRateLimiter` with per-source token buckets and 30-second default cooldown |
| `pkg/catalog/plugin/refresh_status.go` | `RefreshStatusRecord` GORM model, `catalog_refresh_status` table, save/get/list/delete operations |
| `pkg/catalog/plugin/plugin.go` | `RefreshProvider` and `DiagnosticsProvider` interface definitions |

---

[Back to Source Management](./README.md) | [Prev: Validation Pipeline](./validation-pipeline.md)
