# Backend Documentation

This section provides detailed documentation of the Go backend implementation for the Kubeflow Model Registry.

## Contents

| Document | Description |
|----------|-------------|
| [Core Service](./core-service.md) | ModelRegistryService implementation and business logic |
| [Repository Pattern](./repository-pattern.md) | Generic repository with Go generics |
| [Datastore Abstraction](./datastore-abstraction.md) | Pluggable datastore connectors |
| [Database Layer](./database-layer.md) | GORM, migrations, and schema |
| [Converter/Mapper](./converter-mapper.md) | Type conversion patterns |
| [Middleware](./middleware.md) | Validation, routing, and health checks |
| [Configuration](./configuration.md) | Cobra/Viper configuration management |

## Quick Summary

The backend is a **layered Go application** with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────┐
│                    Entry Points (cmd/)                   │
│           proxy.go, root.go, config.go                  │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                OpenAPI Server Layer                      │
│         internal/server/openapi/ (generated)            │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                  Core Service Layer                      │
│           internal/core/modelregistry_service.go        │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                Converter/Mapper Layer                    │
│       internal/converter/, internal/mapper/             │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                  Repository Layer                        │
│              internal/db/service/                       │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                Datastore Abstraction                     │
│              internal/datastore/                        │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                   Database Layer                         │
│        internal/db/schema/, GORM, Migrations            │
└─────────────────────────────────────────────────────────┘
```

## Key Design Patterns

1. **Dependency Injection** - Constructor-based injection throughout
2. **Repository Pattern** - Generic, type-safe data access
3. **Strategy Pattern** - Pluggable datastore connectors
4. **Adapter Pattern** - Converters between API and domain models
5. **Factory Pattern** - Repository creation via reflection

## Key Files

| File | Purpose |
|------|---------|
| `main.go` | Application entry point |
| `cmd/proxy.go` | Proxy server command |
| `cmd/config.go` | Configuration struct |
| `pkg/api/api.go` | Core API interface |
| `internal/core/modelregistry_service.go` | Service implementation |
| `internal/db/service/generic_repository.go` | Repository pattern |
| `internal/datastore/connector.go` | Datastore abstraction |

---

[Back to Documentation Root](../README.md)
