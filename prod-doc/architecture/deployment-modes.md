# Deployment Modes

The Kubeflow Model Registry supports multiple deployment modes to accommodate different use cases and environments.

## Overview

| Mode | Use Case | Features |
|------|----------|----------|
| **Standalone** | Independent deployment | Full UI, local auth, all features |
| **Kubeflow** | Kubeflow platform integration | Shared auth, namespace isolation |
| **Federated** | Distributed/multi-cluster | Cross-cluster discovery, central UI |

---

## Standalone Mode

### Description

Standalone mode provides a complete, self-contained Model Registry deployment without external dependencies.

### Characteristics

- **Independent deployment** - No Kubeflow platform required
- **Full-featured UI** - Complete Model Registry and Catalog UI
- **Local authentication** - Optional, configurable auth
- **Single namespace** - Simplified Kubernetes deployment

### Configuration

**Environment Variables:**

```bash
DEPLOYMENT_MODE=standalone
```

**Docker Compose:**

```bash
make compose/up  # MySQL
make compose/up/postgres  # PostgreSQL
```

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Standalone Deployment                 │
│                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Frontend  │  │     BFF     │  │   Backend   │     │
│  │   (React)   │──│    (Go)     │──│    (Go)     │     │
│  └─────────────┘  └─────────────┘  └──────┬──────┘     │
│                                           │             │
│                                    ┌──────┴──────┐     │
│                                    │   Database  │     │
│                                    │  MySQL/PG   │     │
│                                    └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

### UI Features

In standalone mode, the UI includes:
- Model Registry management
- Model Catalog discovery
- Catalog Settings configuration
- Registry Settings (admin)
- Full navigation sidebar

### Kustomize Deployment

```bash
# Deploy standalone
kubectl apply -k manifests/kustomize/overlays/standalone
```

---

## Kubeflow Mode

### Description

Kubeflow mode integrates the Model Registry as a component within the Kubeflow platform.

### Characteristics

- **Shared authentication** - Uses Kubeflow's auth (Dex, OIDC)
- **Namespace isolation** - Multi-tenant with namespace-based access
- **Platform integration** - Shares resources with other Kubeflow components
- **RBAC-enabled** - Kubernetes RBAC for authorization

### Configuration

**Environment Variables:**

```bash
DEPLOYMENT_MODE=kubeflow
```

**BFF Configuration:**

```go
type Config struct {
    DeploymentMode string // "kubeflow"
    AuthMethod     string // "kubernetes"
    // ...
}
```

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Kubeflow Platform                                   │
│                                                                              │
│  ┌────────────────┐                                                         │
│  │ Kubeflow       │  ┌─────────────────────────────────────────────────┐   │
│  │ Dashboard      │  │            Model Registry Component              │   │
│  │                │  │                                                   │   │
│  │  ┌──────────┐  │  │  ┌─────────────┐  ┌─────────┐  ┌─────────┐     │   │
│  │  │ MR iframe│──┼──│──│   Frontend  │──│   BFF   │──│ Backend │     │   │
│  │  └──────────┘  │  │  └─────────────┘  └─────────┘  └────┬────┘     │   │
│  │                │  │                                      │          │   │
│  └────────────────┘  │                               ┌──────┴──────┐   │   │
│                      │                               │   Database  │   │   │
│  ┌────────────────┐  │                               └─────────────┘   │   │
│  │  Dex (Auth)    │  └─────────────────────────────────────────────────┘   │
│  └────────────────┘                                                         │
│                                                                              │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐               │
│  │   Notebooks    │  │   Pipelines    │  │   KServe       │               │
│  └────────────────┘  └────────────────┘  └────────────────┘               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### UI Features

In Kubeflow mode, the UI:
- Embeds within Kubeflow Dashboard
- Uses shared namespace selector
- Omits redundant navigation (uses Kubeflow's)
- Integrates with Kubeflow authentication

### Kustomize Deployment

```bash
# Deploy with Kubeflow
kubectl apply -k github.com/kubeflow/manifests/apps/model-registry/upstream
```

### Namespace Isolation

```yaml
# Each namespace gets its own Model Registry instance
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-registry
  namespace: user-namespace
```

---

## Federated Mode

### Description

Federated mode enables distributed Model Registry deployments with centralized discovery.

### Characteristics

- **Multi-cluster support** - Registries across multiple clusters
- **Central catalog** - Unified view of all models
- **Cross-cluster discovery** - Search across registries
- **Independent registries** - Each cluster manages its own data

### Configuration

**Environment Variables:**

```bash
DEPLOYMENT_MODE=federated
```

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Federated Deployment                               │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Central Catalog UI                                │   │
│  │                                                                       │   │
│  │  ┌─────────────┐  ┌─────────────┐                                   │   │
│  │  │   Frontend  │──│   Catalog   │                                   │   │
│  │  │   (React)   │  │   Service   │                                   │   │
│  │  └─────────────┘  └──────┬──────┘                                   │   │
│  └───────────────────────────┼─────────────────────────────────────────┘   │
│                              │                                              │
│         ┌────────────────────┼────────────────────┐                        │
│         ▼                    ▼                    ▼                        │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                  │
│  │  Cluster A  │     │  Cluster B  │     │  Cluster C  │                  │
│  │             │     │             │     │             │                  │
│  │ ┌─────────┐ │     │ ┌─────────┐ │     │ ┌─────────┐ │                  │
│  │ │Registry │ │     │ │Registry │ │     │ │Registry │ │                  │
│  │ └─────────┘ │     │ └─────────┘ │     │ └─────────┘ │                  │
│  │             │     │             │     │             │                  │
│  │ ┌─────────┐ │     │ ┌─────────┐ │     │ ┌─────────┐ │                  │
│  │ │   DB    │ │     │ │   DB    │ │     │ │   DB    │ │                  │
│  │ └─────────┘ │     │ └─────────┘ │     │ └─────────┘ │                  │
│  └─────────────┘     └─────────────┘     └─────────────┘                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### UI Features

In federated mode, the UI:
- Shows Model Catalog for cross-cluster discovery
- Provides links to individual cluster registries
- Supports source-based filtering
- Aggregates models from all sources

### Source Configuration

```yaml
# catalog-sources.yaml
catalogs:
  - id: "cluster-a"
    name: "Cluster A Registry"
    type: "model-registry"
    enabled: true
    properties:
      endpoint: "https://cluster-a.example.com/api"

  - id: "cluster-b"
    name: "Cluster B Registry"
    type: "model-registry"
    enabled: true
    properties:
      endpoint: "https://cluster-b.example.com/api"
```

---

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DEPLOYMENT_MODE` | `standalone` | Deployment mode |
| `AUTH_METHOD` | `none` | Authentication method |
| `KUBEFLOW_USERID_HEADER` | `kubeflow-userid` | User ID header |
| `KUBEFLOW_GROUPS_HEADER` | `kubeflow-groups` | Groups header |

### BFF Configuration

```go
type Config struct {
    DeploymentMode      string // standalone, kubeflow, federated
    AuthMethod          string // none, kubernetes
    MockK8sClient       bool   // Use mock K8s client
    MockMRClient        bool   // Use mock MR client
    ModelRegistryURL    string // Backend URL
    CatalogURL          string // Catalog service URL
}
```

### Frontend Configuration

```typescript
// utilities/const.ts
export const DEPLOYMENT_MODE = process.env.DEPLOYMENT_MODE || 'standalone';
export const isStandalone = DEPLOYMENT_MODE === 'standalone';
export const isKubeflow = DEPLOYMENT_MODE === 'kubeflow';
export const isFederated = DEPLOYMENT_MODE === 'federated';
```

---

## Docker Compose Examples

### Standalone with MySQL

```bash
make compose/up
```

### Standalone with PostgreSQL

```bash
make compose/up/postgres
```

### Local Development

```bash
# Start BFF and frontend
make dev-start

# Start in Kubeflow mode
make dev-start-kubeflow

# Start in federated mode
make dev-start-federated
```

---

## Kubernetes Deployment

### Standalone

```bash
kubectl apply -k manifests/kustomize/overlays/standalone
```

### With Kubeflow Manifests

```bash
kubectl apply -k github.com/kubeflow/manifests/apps/model-registry/upstream
```

### Development (Kind)

```bash
# Create Kind cluster
kind create cluster

# Deploy Model Registry
./scripts/deploy_on_kind.sh
```

---

## Mode Selection Guide

| Requirement | Recommended Mode |
|-------------|------------------|
| Quick evaluation | Standalone |
| Single-team use | Standalone |
| Kubeflow platform | Kubeflow |
| Multi-tenant | Kubeflow |
| Multi-cluster | Federated |
| Central catalog | Federated |
| External catalogs (HF) | Any mode + Catalog |

---

[Back to Architecture Index](./README.md) | [Previous: API Design](./api-design.md)
