# Architecture Documentation

This section provides comprehensive documentation of the Kubeflow Model Registry system architecture.

## Contents

| Document | Description |
|----------|-------------|
| [Overview](./overview.md) | High-level system architecture and component relationships |
| [Tech Stack](./tech-stack.md) | Complete technology stack and dependencies |
| [Data Models](./data-models.md) | Entity relationships, schemas, and property system |
| [API Design](./api-design.md) | REST API design patterns and OpenAPI approach |
| [Deployment Modes](./deployment-modes.md) | Standalone, Kubeflow, and Federated deployment modes |

## Quick Summary

The Kubeflow Model Registry is a **microservices-based ML metadata management system** consisting of:

- **Core Model Registry** - REST API server for model metadata CRUD operations
- **Model Catalog Service** - Federated discovery across external catalogs (HuggingFace, YAML)
- **UI Backend for Frontend (BFF)** - Go-based gateway for the React frontend
- **UI Frontend** - React/TypeScript single-page application
- **Kubernetes Controller** - CRD management for Kubernetes-native deployments
- **CSI Driver** - Container Storage Interface for model artifact access

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Clients                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Web UI    │  │Python Client│  │  REST API   │  │  K8s CRDs   │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
└─────────┼────────────────┼────────────────┼────────────────┼────────────────┘
          │                │                │                │
          ▼                │                │                ▼
┌─────────────────┐        │                │     ┌─────────────────┐
│   UI BFF (Go)   │        │                │     │  K8s Controller │
│   Port: 8080    │        │                │     │                 │
└────────┬────────┘        │                │     └────────┬────────┘
         │                 │                │              │
         ▼                 ▼                ▼              │
┌─────────────────────────────────────────────────────────┐│
│              Core Model Registry (Go)                    ││
│                    Port: 8080                            ││
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     ││
│  │ OpenAPI     │  │   Core      │  │ Repository  │     ││
│  │ Server      │──│  Service    │──│   Layer     │     ││
│  └─────────────┘  └─────────────┘  └──────┬──────┘     ││
└───────────────────────────────────────────┼─────────────┘│
                                            │              │
         ┌──────────────────────────────────┼──────────────┘
         │                                  │
         ▼                                  ▼
┌─────────────────┐              ┌─────────────────┐
│  Model Catalog  │              │    Database     │
│    Service      │              │  MySQL/Postgres │
│   Port: 8080    │              │                 │
└────────┬────────┘              └─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│              External Catalogs                   │
│  ┌─────────────┐  ┌─────────────┐              │
│  │ HuggingFace │  │ YAML Files  │              │
│  │     Hub     │  │             │              │
│  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────┘
```

## Key Architectural Principles

1. **Contract-First API Design** - OpenAPI specifications drive all API development
2. **Layered Architecture** - Clear separation between API, service, and data layers
3. **Repository Pattern** - Generic, type-safe data access abstraction
4. **Pluggable Datastores** - Connector pattern for multiple database backends
5. **Code Generation** - Automated generation of API handlers, type converters, and database models

---

[Back to Documentation Root](../README.md)
