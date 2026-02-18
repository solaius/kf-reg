# Source Management

This section documents the runtime management of catalog data sources, including persistent configuration, multi-layer validation, and refresh operations.

## Contents

| Document | Description |
|----------|-------------|
| [Config Stores](./config-stores.md) | ConfigStore interface, file and Kubernetes backends |
| [Validation Pipeline](./validation-pipeline.md) | Multi-layer validation engine |
| [Refresh and Diagnostics](./refresh-and-diagnostics.md) | On-demand refresh, rate limiting, diagnostics |

## Quick Summary

Source management enables operators to:

- **CRUD Sources** - Add, update, enable, disable, and delete catalog sources at runtime
- **Persist Configuration** - Atomic writes with SHA-256 versioning and revision history
- **Validate Before Apply** - Multi-layer validation catches errors before config is saved
- **Rollback** - Restore configuration to any previous revision
- **Refresh** - Trigger on-demand reload with rate limiting
- **Diagnose** - View per-source health, entity counts, and error state

## Architecture Overview

```
┌────────────────────────────────────────────────────────────┐
│                 Management HTTP Handlers                     │
│  POST /management/sources          (apply)                  │
│  POST /management/validate-source  (validate)               │
│  POST /management/refresh          (refresh all)            │
│  POST /management/rollback         (restore revision)       │
│  GET  /management/diagnostics      (health info)            │
└────────────────┬───────────────────────────────────────────┘
                 │
┌────────────────▼───────────────────────────────────────────┐
│              MultiLayerValidator                             │
│  yaml_parse → strict_fields → semantic → security → provider│
└────────────────┬───────────────────────────────────────────┘
                 │
┌────────────────▼───────────────────────────────────────────┐
│              ConfigStore                                     │
│  ┌──────────────────┐  ┌────────────────────────┐          │
│  │  FileConfigStore  │  │  K8sSourceConfigStore  │          │
│  │  (.history/ dir)  │  │  (ConfigMap + annots)  │          │
│  └──────────────────┘  └────────────────────────┘          │
└────────────────────────────────────────────────────────────┘
```

## Management API Endpoints

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/management/sources` | viewer | List configured sources |
| POST | `/management/sources` | operator | Apply source configuration |
| POST | `/management/validate-source` | viewer | Validate without applying |
| POST | `/management/sources/{id}/enable` | operator | Enable/disable source |
| DELETE | `/management/sources/{id}` | operator | Delete source |
| POST | `/management/refresh` | operator | Refresh all sources |
| POST | `/management/sources/{id}/refresh` | operator | Refresh specific source |
| GET | `/management/revisions` | viewer | List config revisions |
| POST | `/management/rollback` | operator | Restore previous config |
| GET | `/management/diagnostics` | viewer | Plugin diagnostics |

---

[Back to Catalog of Catalogs](../README.md)
