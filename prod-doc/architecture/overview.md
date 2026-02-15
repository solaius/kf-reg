# System Architecture Overview

## Introduction

The Kubeflow Model Registry is an enterprise-grade ML metadata management system designed to provide a central repository for storing and managing models, versions, and artifacts metadata. It operates as a microservices architecture with clear separation of concerns.

## System Components

### 1. Core Model Registry (Proxy Server)

**Location:** `cmd/proxy.go`, `internal/`

The core Model Registry is a Go-based REST API server that implements the OpenAPI specification defined in `api/openapi/model-registry.yaml`.

**Responsibilities:**
- CRUD operations for model metadata (RegisteredModel, ModelVersion, Artifact)
- Inference service and serving environment management
- Experiment and experiment run tracking
- Custom property management for extensible metadata

**Key Features:**
- Dynamic router pattern for graceful initialization
- Health check and readiness probes
- Thread-safe service holder for runtime updates
- Concurrent database connection management

### 2. Model Catalog Service

**Location:** `catalog/`

A federated discovery service that aggregates model metadata from external catalogs.

**Responsibilities:**
- Source provider management (YAML, HuggingFace Hub)
- Model search and discovery across sources
- Performance metrics aggregation
- Hot-reload configuration support

**Supported Sources:**
- **YAML Catalog** - Static file-based model definitions
- **HuggingFace Hub** - Real-time model discovery with pattern matching

### 3. UI Backend for Frontend (BFF)

**Location:** `clients/ui/bff/`

A Go-based intermediary service optimized for the React frontend.

**Responsibilities:**
- API aggregation and transformation
- Kubernetes client integration
- RBAC and authentication handling
- Mock support for development

### 4. UI Frontend

**Location:** `clients/ui/frontend/`

A React/TypeScript single-page application built with PatternFly.

**Responsibilities:**
- Model registry management interface
- Model catalog discovery interface
- Settings and configuration management
- Multi-deployment mode support (Standalone, Kubeflow, Federated)

### 5. Kubernetes Controller

**Location:** `cmd/controller/`

A Kubernetes controller for managing Model Registry Custom Resource Definitions.

**Responsibilities:**
- CRD lifecycle management
- Kubernetes-native model registration
- Integration with Kubernetes RBAC

### 6. CSI Driver

**Location:** `cmd/csi/`

Container Storage Interface driver for model artifact access.

**Responsibilities:**
- Model artifact mounting in Kubernetes pods
- Integration with inference services
- Storage initialization for model deployments

## Request Flow

### Web UI Request Flow

```
User Browser
    │
    ▼
┌───────────────────┐
│   React Frontend  │ (localhost:9000 or deployed)
│   - PatternFly UI │
│   - Context API   │
└─────────┬─────────┘
          │ HTTP/REST
          ▼
┌───────────────────┐
│   UI BFF (Go)     │ (/model-registry/api/v1/)
│   - Handlers      │
│   - Repositories  │
│   - K8s Client    │
└─────────┬─────────┘
          │ HTTP/REST
          ▼
┌───────────────────┐
│  Model Registry   │ (/api/model_registry/v1alpha3/)
│   - OpenAPI       │
│   - Core Service  │
│   - Repositories  │
└─────────┬─────────┘
          │ GORM
          ▼
┌───────────────────┐
│    Database       │
│  MySQL/PostgreSQL │
└───────────────────┘
```

### Python Client Request Flow

```
Python Application
    │
    ▼
┌───────────────────┐
│  Python Client    │ (model-registry SDK)
│   - OpenAPI Gen   │
└─────────┬─────────┘
          │ HTTP/REST
          ▼
┌───────────────────┐
│  Model Registry   │
│   (Direct API)    │
└───────────────────┘
```

### Model Catalog Request Flow

```
User Browser
    │
    ▼
┌───────────────────┐
│   React Frontend  │
│   (Catalog UI)    │
└─────────┬─────────┘
          │ HTTP/REST
          ▼
┌───────────────────┐
│   UI BFF (Go)     │
│   - Catalog Repos │
└─────────┬─────────┘
          │ HTTP/REST
          ▼
┌───────────────────┐
│  Catalog Service  │ (/api/model_catalog/v1alpha1/)
│   - APIProvider   │
│   - Source Mgmt   │
└─────────┬─────────┘
          │
    ┌─────┴─────┐
    ▼           ▼
┌────────┐ ┌────────────┐
│  YAML  │ │ HuggingFace│
│ Files  │ │    Hub     │
└────────┘ └────────────┘
```

## Component Interaction Patterns

### 1. Synchronous REST API

All component-to-component communication uses synchronous HTTP/REST calls:
- Frontend → BFF: REST with JSON payloads
- BFF → Model Registry: REST with OpenAPI models
- BFF → Catalog Service: REST with OpenAPI models

### 2. Database Interaction

- Uses GORM ORM for type-safe database access
- Supports MySQL 8.3+ and PostgreSQL
- Connection pooling managed by GORM driver
- Migration-based schema management

### 3. Kubernetes Integration

- Controller-runtime for CRD management
- Client-go for Kubernetes API access
- RBAC-aware operations
- Service account-based authentication

## Concurrency Model

### Proxy Server Initialization

```go
// Concurrent initialization pattern
go func() {
    // Connect to database in background
    repoSet, err := connector.Connect(spec)
    // Swap router when ready
    router.SetRouter(newRouter)
}()

// Server starts immediately, returns 503 until ready
http.ListenAndServe(address, dynamicRouter)
```

### Thread-Safe Components

- `ModelRegistryServiceHolder` - RWMutex-protected service container
- `dynamicRouter` - RWMutex-protected handler swapping
- `db.Connector` - Singleton with mutex protection
- Database connection pool - GORM-managed concurrency

## Error Handling Strategy

### Domain Errors

```go
var (
    ErrBadRequest = errors.New("bad request")   // 400
    ErrNotFound   = errors.New("not found")     // 404
    ErrConflict   = errors.New("conflict")      // 409
)
```

### Error Propagation

1. Repository layer catches database errors
2. Service layer wraps with domain context
3. OpenAPI layer translates to HTTP status codes
4. BFF layer formats for frontend consumption

## Security Architecture

### Input Validation

- Null byte prevention in middleware
- Parameter validation against OpenAPI schema
- SQL injection prevention via parameterized queries

### Authentication & Authorization

- Kubernetes RBAC integration (in Kubeflow mode)
- Service account-based API access
- TLS/SSL support with configurable certificates

---

[Back to Architecture Index](./README.md) | [Next: Tech Stack](./tech-stack.md)
