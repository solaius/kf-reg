# BFF (Backend for Frontend) Documentation

This section covers the BFF layer that acts as a gateway between the React frontend and backend services.

## Overview

The BFF is a Go-based HTTP server using the `julienschmidt/httprouter` package. It provides:

- **API Gateway**: Proxies requests to Model Registry and Catalog services
- **Authentication**: Integrates with Kubernetes RBAC
- **Kubernetes Integration**: Manages model registries, namespaces, and settings
- **Static File Serving**: Serves the React frontend

## Technology Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.24+ | Server runtime |
| httprouter | - | HTTP routing |
| client-go | - | Kubernetes client |
| slog | - | Structured logging |

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](./architecture.md) | BFF layer design and structure |
| [Handlers](./handlers.md) | API handler patterns |
| [Repositories](./repositories.md) | Data access layer |
| [Kubernetes Integration](./kubernetes-integration.md) | K8s client integration |

## Directory Structure

```
clients/ui/bff/
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   ├── api/                       # HTTP handlers
│   │   ├── app.go                 # App setup and routes
│   │   ├── middleware.go          # Middleware functions
│   │   ├── *_handler.go           # Request handlers
│   │   └── errors.go              # Error responses
│   ├── config/                    # Configuration
│   │   └── environment.go         # Environment config
│   ├── constants/                 # Constants
│   ├── helpers/                   # Helper utilities
│   ├── integrations/
│   │   ├── httpclient/            # HTTP client
│   │   └── kubernetes/            # K8s integration
│   ├── mocks/                     # Mock implementations
│   ├── models/                    # Data models
│   ├── repositories/              # Data access
│   └── validation/                # Request validation
└── go.mod
```

## API Routes

### Model Registry Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/model_registry/:id/registered_models` | List models |
| POST | `/api/v1/model_registry/:id/registered_models` | Create model |
| GET | `/api/v1/model_registry/:id/registered_models/:modelId` | Get model |
| PATCH | `/api/v1/model_registry/:id/registered_models/:modelId` | Update model |

### Model Catalog Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/model_catalog/models` | List catalog models |
| GET | `/api/v1/model_catalog/sources` | List sources |
| GET | `/api/v1/model_catalog/models/filter_options` | Get filter options |

### MCP Catalog Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/mcp_catalog/mcp_servers` | List MCP servers |
| GET | `/api/v1/mcp_catalog/mcp_servers/:serverId` | Get MCP server |
| GET | `/api/v1/mcp_catalog/mcp_servers/filter_options` | Get filter options |
| GET | `/api/v1/mcp_catalog/sources` | List MCP sources |

### Kubernetes Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/user` | Get current user |
| GET | `/api/v1/model_registry` | List registries |
| GET | `/api/v1/namespaces` | List namespaces |

## Quick Start

### Local Development

```bash
cd clients/ui/bff

# Set environment variables
export PORT=4000
export DEV_MODE=true
export MOCK_K8S_CLIENT=true
export MOCK_MR_CLIENT=true

# Run the server
go run cmd/main.go
```

### Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `4000` |
| `DEV_MODE` | Enable development mode | `false` |
| `MOCK_K8S_CLIENT` | Use mock Kubernetes client | `false` |
| `MOCK_MR_CLIENT` | Use mock Model Registry client | `false` |
| `STATIC_ASSETS_DIR` | Frontend static files path | `../frontend/dist` |
| `MODEL_REGISTRY_BASE_URL` | Model Registry API URL | - |
| `MODEL_CATALOG_BASE_URL` | Model Catalog API URL | - |

---

[Back to Main Index](../README.md)
