# Kubernetes Components Documentation

This section covers Kubernetes-specific components of the Model Registry.

## Overview

The Model Registry integrates with Kubernetes through several components:

- **Controller**: Syncs KServe InferenceServices with Model Registry
- **CSI Driver**: Enables KServe to serve models from Model Registry
- **Deployment Manifests**: Kustomize-based deployment configurations

## Documentation

| Document | Description |
|----------|-------------|
| [Controller](./controller.md) | InferenceService controller for K8s-MR sync |
| [CSI Driver](./csi-driver.md) | Custom Storage Initializer for KServe |
| [Deployment Manifests](./deployment-manifests.md) | Kustomize deployment patterns |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Integration                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Kubernetes Cluster                        │ │
│  │                                                               │ │
│  │  ┌─────────────────┐    ┌─────────────────┐                 │ │
│  │  │  InferenceService│    │  CSI Storage    │                 │ │
│  │  │    Controller    │    │   Initializer   │                 │ │
│  │  └────────┬────────┘    └────────┬────────┘                 │ │
│  │           │                      │                           │ │
│  │           │ Watches/Syncs        │ Downloads                 │ │
│  │           ▼                      ▼                           │ │
│  │  ┌─────────────────┐    ┌─────────────────┐                 │ │
│  │  │ KServe ISVC     │    │  Model Files    │                 │ │
│  │  │ (CRD)           │    │ (Init Container)│                 │ │
│  │  └─────────────────┘    └─────────────────┘                 │ │
│  │                                                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│               ┌─────────────────────────────┐                    │
│               │      Model Registry API      │                    │
│               │   /api/model_registry/v1alpha3│                   │
│               └─────────────────────────────┘                    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Component Summary

### InferenceService Controller

Watches KServe InferenceService CRs and automatically:
- Creates/updates InferenceService entities in Model Registry
- Syncs deployment URLs between K8s and Model Registry
- Manages ServingEnvironment (namespace mapping)
- Handles graceful deletion with finalizers

### CSI Storage Initializer

Enables KServe to serve models indexed in Model Registry:
- Registers `model-registry://` protocol with KServe
- Resolves Model Registry URIs to actual storage locations
- Downloads models to pod init containers
- Supports S3, GCS, HTTP, and other storage backends

### Kustomize Manifests

Provides flexible deployment configurations:
- **Base**: Core Model Registry server deployment
- **Overlays**: Database configurations (MySQL, PostgreSQL)
- **Options**: Optional components (CSI, Controller, UI, Istio)

## Quick Start

### Deploy Model Registry

```bash
# Basic deployment with MySQL
kubectl apply -k manifests/kustomize/overlays/db -n kubeflow

# With Istio integration
kubectl apply -k manifests/kustomize/options/istio -n kubeflow
```

### Enable CSI Driver

```bash
# Cluster-scoped (once per cluster)
kubectl apply -k manifests/kustomize/options/csi
```

### Deploy Controller

```bash
kubectl apply -k manifests/kustomize/options/controller/overlays/base -n kubeflow
```

## CRD Overview

| CRD | API Group | Description |
|-----|-----------|-------------|
| InferenceService | serving.kserve.io/v1beta1 | KServe model deployment |
| ClusterStorageContainer | serving.kserve.io/v1alpha1 | CSI driver registration |
| ModelRegistry | modelregistry.kubeflow.org/v1alpha1 | Registry instance (optional) |

## Environment Variables

### Controller

| Variable | Description |
|----------|-------------|
| `INFERENCE_SERVICE_CONTROLLER` | Enable controller (true/false) |
| `MODEL_REGISTRY_URL_ANNOTATION` | MR service URL annotation |
| `MODEL_REGISTRY_NAMESPACE_LABEL` | MR namespace label |
| `REGISTRIES_NAMESPACE` | Default MR namespace |

### CSI Driver

| Variable | Description |
|----------|-------------|
| `MODEL_REGISTRY_BASE_URL` | Model Registry service URL |
| `MODEL_REGISTRY_SCHEME` | HTTP scheme (http/https) |

---

[Back to Main Index](../README.md)
