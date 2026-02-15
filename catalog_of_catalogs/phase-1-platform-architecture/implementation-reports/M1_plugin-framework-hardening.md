# M1: Plugin Framework Hardening

**Date**: 2026-02-15
**Status**: Complete
**Phase**: Phase 1: Platform Architecture

## Summary

This milestone hardened the plugin framework with failure isolation during initialization, optional capability and status interfaces for richer plugin metadata, and enhanced `/api/plugins` and `/readyz` endpoints that report per-plugin health and error details. These changes ensure the catalog server remains available even when individual plugins fail, and that operators can diagnose problems through the API.

## Motivation

- The original plugin server treated any `Init()` failure as fatal, bringing down all plugins when one misbehaved. Production deployments need graceful degradation.
- The `/api/plugins` endpoint returned only basic identity information (name, version, basePath). UIs and CLIs that discover plugins generically need to know what entity kinds a plugin serves and what operations it supports.
- The `/readyz` endpoint did not account for plugins that failed initialization, potentially reporting "ready" when a plugin was down.
- Satisfies **FR2** (Plugin discovery & metadata) -- richer `/api/plugins` response with capabilities.
- Satisfies **FR3** (Plugin lifecycle) -- graceful failure isolation so one broken plugin does not block others.
- Satisfies **AC3** (`/api/plugins` lists plugins and reports health).

## What Changed

### Files Created

_No new files were created; all changes were to existing files._

### Files Modified

| File | Change |
|------|--------|
| `pkg/catalog/plugin/plugin.go` | Added `PluginCapabilities` struct, `CapabilitiesProvider` interface, `PluginStatus` struct, and `StatusProvider` interface |
| `pkg/catalog/plugin/server.go` | Added `failedPlugin` struct and `failedPlugins` slice to `Server`; changed `Init()` to log-and-continue on plugin failure; updated `pluginsHandler` to include capabilities, status, and failed plugins; updated `readyHandler` to include failed plugins |
| `pkg/catalog/plugin/server_test.go` | Added `failingPlugin` and `capablePlugin` test doubles; added `TestServerInitPluginFailureIsolation`, `TestServerPluginsEndpointWithCapabilities`, and `TestServerReadyEndpointWithFailedPlugin` tests |

## How It Works

### Optional Capability and Status Interfaces

Plugins can optionally implement two new interfaces to advertise richer metadata. The server detects these via type assertion at runtime, so existing plugins require no changes.

```go
type PluginCapabilities struct {
    EntityKinds  []string `json:"entityKinds"`
    ListEntities bool     `json:"listEntities"`
    GetEntity    bool     `json:"getEntity"`
    ListSources  bool     `json:"listSources"`
    Artifacts    bool     `json:"artifacts"`
}

type CapabilitiesProvider interface {
    Capabilities() PluginCapabilities
}

type PluginStatus struct {
    Enabled     bool   `json:"enabled"`
    Initialized bool   `json:"initialized"`
    Serving     bool   `json:"serving"`
    LastError   string `json:"lastError,omitempty"`
}

type StatusProvider interface {
    Status() PluginStatus
}
```

### Failure Isolation in Init()

When a plugin's `Init()` returns an error, the server logs the failure and moves it to the `failedPlugins` list instead of aborting. Healthy plugins continue to initialize and serve traffic.

```go
if err := p.Init(ctx, pluginCfg); err != nil {
    s.logger.Error("plugin init failed, continuing with remaining plugins",
        "plugin", p.Name(), "error", err)
    s.failedPlugins = append(s.failedPlugins, failedPlugin{plugin: p, err: err})
    continue
}
s.plugins = append(s.plugins, p)
```

The `Server.Init()` method now always returns `nil` -- individual plugin failures are recorded, not propagated.

### Enhanced /api/plugins Endpoint

The `pluginsHandler` now returns a richer JSON payload. For healthy plugins it includes `capabilities` and `status` if the plugin implements the corresponding interfaces. For failed plugins it synthesizes a `PluginStatus` with `initialized: false` and the error message.

```json
{
  "plugins": [
    {
      "name": "models",
      "version": "v1alpha1",
      "basePath": "/api/models_catalog/v1alpha1",
      "healthy": true,
      "entityKinds": ["Model", "ModelVersion", "ModelArtifact"],
      "capabilities": { "listEntities": true, "getEntity": true, ... }
    },
    {
      "name": "broken",
      "version": "v1",
      "healthy": false,
      "status": { "enabled": true, "initialized": false, "lastError": "connection refused" }
    }
  ],
  "count": 2
}
```

### Enhanced /readyz Endpoint

The `readyHandler` now iterates over both `s.plugins` (healthy) and `s.failedPlugins` (failed). If any failed plugin exists, the endpoint returns `503 Service Unavailable` with `"status": "not_ready"` and a per-plugin boolean map.

```go
for _, fp := range s.failedPlugins {
    pluginStatus[fp.plugin.Name()] = false
    allHealthy = false
}
```

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Optional interfaces via type assertion | Backward-compatible -- existing plugins need no changes | Embedding required fields in `CatalogPlugin` (would break all existing plugins) |
| Log-and-continue on Init failure | A single broken plugin should not take down the entire server | Return first error (original behavior); collect errors and return multi-error |
| Store failed plugins in a separate slice | Clean separation between serving and broken plugins; avoids nil-checking throughout | Single slice with a status flag per entry |
| Synthetic `PluginStatus` for failed plugins | Operators see a consistent status shape for all plugins in `/api/plugins` | Omit failed plugins from the response (hides problems) |

## Testing

Three new test functions were added to `pkg/catalog/plugin/server_test.go`:

- **`TestServerInitPluginFailureIsolation`** -- Registers a `failingPlugin` and a `testPlugin`. Verifies that `Init()` succeeds, only the working plugin is in `Plugins()`, and `/api/plugins` lists both (the failing one with error status).
- **`TestServerPluginsEndpointWithCapabilities`** -- Registers a `capablePlugin` that implements `CapabilitiesProvider`. Verifies `/api/plugins` includes the `capabilities` and `entityKinds` fields.
- **`TestServerReadyEndpointWithFailedPlugin`** -- Registers only a failing plugin. Verifies `/readyz` returns 503 with `"status": "not_ready"` and the plugin marked unhealthy.

Run with:
```bash
go test ./pkg/catalog/plugin/ -v -run "TestServerInit|TestServerPlugins|TestServerReady"
```

## Verification

```bash
# Run the plugin framework tests
go test ./pkg/catalog/plugin/ -v

# Start the catalog server locally (requires DB) and verify endpoints
# 1. Check plugin listing
curl -s http://localhost:8080/api/plugins | jq .

# 2. Check readiness (should include per-plugin status)
curl -s http://localhost:8080/readyz | jq .
```

## Dependencies & Impact

- **Enables**: M2 (MCP Plugin End-to-End) and all future plugins benefit from failure isolation and capability advertisement. UIs and CLIs can discover plugin capabilities at runtime.
- **Depends on**: The existing plugin framework (`CatalogPlugin` interface, `Server`, `Registry`) introduced in prior work.
- **Backward compatibility**: Fully backward-compatible. The two new interfaces are optional; existing plugins that do not implement them continue to work unchanged.

## Open Items

- The `CapabilitiesProvider` fields are static today. A future milestone could make capabilities dynamic (e.g., a plugin that starts without artifact support and gains it after ingestion).
- There is no retry or auto-recovery for failed plugins. A plugin that fails `Init()` stays in the failed list until the server restarts.
