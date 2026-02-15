# Catalog Service Documentation

This section documents the Model Catalog Service, a federated discovery layer for ML models across multiple external catalogs.

## Contents

| Document | Description |
|----------|-------------|
| [Architecture](./architecture.md) | Catalog service architecture and design |
| [Source Providers](./source-providers.md) | YAML and HuggingFace catalog providers |
| [Filtering System](./filtering-system.md) | Query filtering and named queries |
| [Database Models](./database-models.md) | Catalog-specific data models |
| [Performance Metrics](./performance-metrics.md) | Performance artifact handling |

## Quick Summary

The Model Catalog Service provides **read-only discovery** across multiple catalog sources.

### Key Features

- **Federated Discovery** - Aggregate models from multiple sources
- **Pluggable Providers** - YAML files, HuggingFace Hub, extensible
- **Hot-Reload** - Configuration changes without restart
- **Advanced Filtering** - Query DSL with named queries
- **Performance Metrics** - Model benchmark data integration

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Model Catalog Service                                │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        REST API Layer                                │   │
│  │                /api/model_catalog/v1alpha1                          │   │
│  └─────────────────────────────┬───────────────────────────────────────┘   │
│                                │                                            │
│  ┌─────────────────────────────▼───────────────────────────────────────┐   │
│  │                      APIProvider Interface                           │   │
│  │   GetModel(), ListModels(), GetArtifacts(), GetFilterOptions()     │   │
│  └─────────────────────────────┬───────────────────────────────────────┘   │
│                                │                                            │
│         ┌──────────────────────┼──────────────────────┐                    │
│         ▼                      ▼                      ▼                    │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
│  │  YAML Provider  │  │    HF Provider  │  │ Database Catalog│            │
│  │                 │  │                 │  │                 │            │
│  │ - Static files  │  │ - API client    │  │ - GORM storage  │            │
│  │ - Hot-reload    │  │ - Pattern match │  │ - Caching       │            │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘            │
│           │                    │                    │                      │
│           ▼                    ▼                    ▼                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Source Collection                                 │   │
│  │          (Priority-based merging, label filtering)                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/sources` | List catalog sources |
| `GET` | `/models` | Search models (requires source) |
| `GET` | `/models/filter_options` | Get available filters |
| `GET` | `/sources/{id}/models/{name}` | Get specific model |
| `GET` | `/sources/{id}/models/{name}/artifacts` | Get model artifacts |
| `POST` | `/sources/preview` | Preview source config |
| `GET` | `/labels` | List source labels |

## Key Files

| File | Purpose |
|------|---------|
| `catalog/cmd/catalog.go` | Entry point |
| `catalog/internal/catalog/catalog.go` | APIProvider interface |
| `catalog/internal/catalog/db_catalog.go` | Database implementation |
| `catalog/internal/catalog/yaml_catalog.go` | YAML provider |
| `catalog/internal/catalog/hf_catalog.go` | HuggingFace provider |
| `catalog/internal/catalog/loader.go` | Configuration loader |
| `catalog/internal/catalog/sources.go` | Source collection |

---

[Back to Documentation Root](../README.md)
