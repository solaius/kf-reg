# Plugin Framework

This section documents the self-registering plugin architecture that enables the catalog-of-catalogs system to host multiple AI asset catalogs in a single server process.

## Contents

| Document | Description |
|----------|-------------|
| [Architecture](./architecture.md) | Core interfaces, plugin registry, server lifecycle, failure isolation |
| [Creating Plugins](./creating-plugins.md) | Step-by-step guide to building a new plugin |
| [Configuration](./configuration.md) | sources.yaml format, config loading, environment variables |

## Quick Summary

The plugin framework provides:

- **Self-Registration** - Plugins register via Go `init()` functions, discovered automatically at startup
- **Failure Isolation** - One broken plugin does not crash the server
- **Unified HTTP Server** - All plugins share a single chi router with scoped sub-routers
- **Optional Interfaces** - Plugins opt into features (management, actions, capabilities) by implementing interfaces
- **Config Reconciliation** - External config changes detected and applied every 30 seconds

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                      Plugin Server                            │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                   chi.Router                            │  │
│  │  /api/plugins              Plugin info + V2 caps        │  │
│  │  /healthz, /livez, /readyz Health endpoints             │  │
│  └────────┬───────────────────────────────────────────────┘  │
│           │                                                   │
│  ┌────────▼───────────────────────────────────────────────┐  │
│  │              Per-Plugin Sub-Routers                      │  │
│  │                                                          │  │
│  │  /api/model_catalog/v1alpha1/       Model routes         │  │
│  │  /api/mcp_catalog/v1alpha1/         MCP routes           │  │
│  │  /api/agents_catalog/v1alpha1/      Agents routes        │  │
│  │  .../{basePath}/management/         Management routes    │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              Plugin Registry (global)                    │  │
│  │  Register() / All() / Get()                             │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Key Interfaces

| Interface | Required | Purpose |
|-----------|----------|---------|
| `CatalogPlugin` | Yes | Core lifecycle (init, start, stop, routes, health) |
| `CapabilitiesV2Provider` | Recommended | V2 discovery document for generic UI/CLI |
| `AssetMapperProvider` | Recommended | Universal entity projection |
| `ActionProvider` | Optional | Entity and source action handling |
| `SourceManager` | Optional | Runtime source CRUD |
| `RefreshProvider` | Optional | On-demand data reload |

---

[Back to Catalog of Catalogs](../README.md)
