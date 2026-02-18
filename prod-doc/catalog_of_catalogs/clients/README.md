# Client Integration

This section documents the three client surfaces that consume the catalog-of-catalogs API: the BFF proxy layer, the generic React UI, and the catalogctl CLI.

## Contents

| Document | Description |
|----------|-------------|
| [BFF Integration](./bff-integration.md) | Backend-for-Frontend proxy handlers |
| [Generic UI](./generic-ui.md) | Capabilities-driven React components |
| [catalogctl and Conformance](./catalogctl-and-conformance.md) | Dynamic CLI and conformance test suite |

## Quick Summary

All three client surfaces are **capabilities-driven** -- they discover plugins and their entities at runtime, with no plugin-specific code:

- **BFF** proxies catalog-server API to the frontend with path translation
- **Generic UI** renders list/detail/action views from V2 capabilities
- **catalogctl** builds dynamic command trees from plugin discovery
- **Conformance harness** (Phase 9) -- importable library at `pkg/catalog/conformance/` with 6 test categories and JSON reports

## Architecture Overview

```
┌───────────────────────────────────────────────────┐
│  React UI (GenericCatalog)                         │
│    CatalogContextProvider                          │
│    GenericListView / GenericDetailView              │
│    GenericActionDialog / GenericFilterBar           │
└──────────────────┬────────────────────────────────┘
                   │ REST
┌──────────────────▼────────────────────────────────┐
│  BFF Layer (:4000)                                 │
│    /api/v1/catalog/plugins                         │
│    /api/v1/catalog/:plugin/:entity                 │
│    /api/v1/catalog/:plugin/management/...          │
└──────────────────┬────────────────────────────────┘
                   │ REST
┌──────────────────▼────────────────────────────────┐
│  Catalog Server (:8080)                            │
└───────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────┐
│  catalogctl CLI                                    │
│    discoverPlugins() → dynamic subcommands         │
└──────────────────┬────────────────────────────────┘
                   │ REST (direct)
┌──────────────────▼────────────────────────────────┐
│  Catalog Server (:8080)                            │
└───────────────────────────────────────────────────┘
```

---

[Back to Catalog of Catalogs](../README.md)
