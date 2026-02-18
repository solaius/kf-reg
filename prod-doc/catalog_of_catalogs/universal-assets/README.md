# Universal Asset Framework

This section documents the capabilities-driven discovery system, universal asset contract, and action framework that enable new plugins to appear in the UI and CLI with zero code changes.

## Contents

| Document | Description |
|----------|-------------|
| [Capabilities Discovery](./capabilities-discovery.md) | V2 capabilities schema and builder |
| [Asset Contract](./asset-contract.md) | AssetResource envelope, AssetMapper, overlay store |
| [Action Framework](./action-framework.md) | ActionProvider, builtin actions, :action endpoints |

## Quick Summary

The universal asset framework is the key innovation that transforms the catalog from a fixed set of entity types into an extensible platform:

- **Capabilities Discovery** - Each plugin declares its entities, fields, endpoints, and actions in a V2 document
- **Universal Asset Contract** - Plugins project native entities into a common `AssetResource` envelope
- **Action Framework** - Standardized action execution with dry-run support and overlay-based persistence
- **Zero-Code-Change Extensibility** - UI and CLI render any plugin purely from its capabilities document

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│                    Plugin (native entities)                │
│                    e.g., McpServer, Agent                 │
└────────────────────────┬─────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────┐
│                    AssetMapper                             │
│              (plugin-specific projection)                  │
└────────────────────────┬─────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────┐
│                    AssetResource                           │
│              (universal envelope)                          │
│    apiVersion, kind, metadata, spec, status               │
└────────────────────────┬─────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────┐
│                    OverlayStore                            │
│         (user modifications: tags, annotations,           │
│          lifecycle, persisted in catalog_overlays)         │
└────────────────────────┬─────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────┐
│                    CapabilitiesV2                          │
│         (discovery document drives rendering)             │
└────────────────┬───────────────────┬─────────────────────┘
                 │                   │
    ┌────────────▼──────┐  ┌────────▼──────────┐
    │    Generic UI      │  │   catalogctl CLI   │
    │  (React/PatternFly)│  │  (Cobra, dynamic)  │
    └───────────────────┘  └───────────────────┘
```

---

[Back to Catalog of Catalogs](../README.md)
