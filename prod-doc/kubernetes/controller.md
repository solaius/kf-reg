# InferenceService Controller

This document covers the Kubernetes controller that synchronizes KServe InferenceServices with the Model Registry.

## Overview

The InferenceService Controller watches KServe InferenceService Custom Resources (CRs) in Kubernetes and automatically registers/updates corresponding InferenceService entities in the Model Registry. This creates bidirectional sync between infrastructure state (Kubernetes) and metadata management (Model Registry).

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  InferenceService Controller                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Controller Manager                        │ │
│  │                                                               │ │
│  │  ┌─────────────────┐    ┌─────────────────────────────────┐ │ │
│  │  │   Reconciler    │    │        Label/Annotation          │ │ │
│  │  │                 │◄───│          Configuration           │ │ │
│  │  └────────┬────────┘    └─────────────────────────────────┘ │ │
│  │           │                                                   │ │
│  │           │ Watches                                           │ │
│  │           ▼                                                   │ │
│  │  ┌─────────────────┐                                         │ │
│  │  │ KServe ISVC CR  │  (serving.kserve.io/v1beta1)            │ │
│  │  └────────┬────────┘                                         │ │
│  │           │                                                   │ │
│  │           │ Syncs                                             │ │
│  │           ▼                                                   │ │
│  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │              Model Registry API                          │ │ │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │ │ │
│  │  │  │   Serving   │  │ Inference   │  │   Registered    │  │ │ │
│  │  │  │ Environment │  │   Service   │  │     Model       │  │ │ │
│  │  │  └─────────────┘  └─────────────┘  └─────────────────┘  │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  │                                                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## File Structure

```
cmd/controller/
├── main.go                    # Entry point, manager setup
├── Dockerfile.controller      # Container build
└── README.md

internal/controller/controllers/
├── inferenceservice_controller.go       # Controller adapter
└── inferenceservice_controller_test.go  # Unit tests

pkg/inferenceservice-controller/
├── controller.go              # Core reconciliation logic
├── controller_test.go         # Integration tests
├── suite_test.go             # Test setup
└── testdata/                 # Test fixtures
    ├── crd/                  # InferenceService CRD
    ├── deploy/               # Model Registry Service
    └── inferenceservices/    # Sample ISVCs
```

## How It Works

### Reconciliation Flow

```
InferenceService created/modified in Kubernetes
    │
    ▼
Controller detects change via watch
    │
    ▼
Extract Model Registry labels from ISVC:
  - inferenceServiceIDLabel → Links to existing MR ISVC
  - registeredModelIDLabel → Model to serve
  - modelVersionIDLabel → Specific version (optional)
  - modelRegistryNameLabel → MR service name
  - modelRegistryURLAnnotation → MR service URL
    │
    ▼
Initialize Model Registry API client
  - Discover MR service via labels or component=model-registry
  - Build HTTP/HTTPS URL from service endpoint
    │
    ▼
Retrieve/Create ServingEnvironment
  - Use K8s namespace as environment name
  - Create if doesn't exist
    │
    ▼
┌─── IF ISVC has ID label ───┐
│                            │
│  Retrieve from MR by ID    │
│  Check URL differences     │
│  Update MR if URL changed  │
│                            │
└────────────────────────────┘
        │
        │ ELSE IF RegisteredModel ID specified
        ▼
┌────────────────────────────┐
│                            │
│  Create new ISVC in MR     │
│  Link to RegisteredModel   │
│  Set DesiredState=DEPLOYED │
│  Store URL in customProps  │
│                            │
└────────────────────────────┘
    │
    ▼
Add Finalizer for cleanup
    │
    ▼
Update ISVC labels with generated ID
    │
    ▼
On Deletion:
  - Set DesiredState=UNDEPLOYED in MR
  - Remove finalizer
  - Allow K8s deletion to proceed
```

### Label Configuration

The controller uses labels and annotations to link K8s resources with Model Registry:

| Label/Annotation | Purpose |
|------------------|---------|
| `modelregistry.kubeflow.org/registered-model-id` | Required: Model to serve |
| `modelregistry.kubeflow.org/model-version-id` | Optional: Specific version |
| `modelregistry.kubeflow.org/inference-service-id` | Auto-generated: MR ISVC ID |
| `modelregistry.kubeflow.org/name` | MR service name |
| `modelregistry.kubeflow.org/namespace` | MR namespace |
| `modelregistry.kubeflow.org/url` (annotation) | MR service external URL |

### Example InferenceService

```yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: iris-model
  namespace: ml-models
  labels:
    modelregistry.kubeflow.org/registered-model-id: "1"
    modelregistry.kubeflow.org/model-version-id: "1"
    modelregistry.kubeflow.org/name: model-registry
    modelregistry.kubeflow.org/namespace: kubeflow
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
      storageUri: "model-registry://iris/v1"
```

## Model Registry Entities

### ServingEnvironment

Maps Kubernetes namespaces to serving environments:

```json
{
  "id": "1",
  "name": "ml-models",
  "description": "ML models namespace",
  "customProperties": {}
}
```

### InferenceService (in MR)

Represents a deployed model version:

```json
{
  "id": "1",
  "registeredModelId": "1",
  "modelVersionId": "1",
  "servingEnvironmentId": "1",
  "runtime": "kserve",
  "desiredState": "DEPLOYED",
  "customProperties": {
    "url": {
      "string_value": "https://iris-model.ml-models.example.com"
    }
  }
}
```

## Environment Configuration

### Controller Environment Variables

```bash
# Enable/disable the controller
INFERENCE_SERVICE_CONTROLLER=true

# Label/annotation keys (defaults shown)
NAMESPACE_LABEL=modelregistry.kubeflow.org/namespace
NAME_LABEL=modelregistry.kubeflow.org/name
URL_ANNOTATION=modelregistry.kubeflow.org/url
INFERENCE_SERVICE_ID_LABEL=modelregistry.kubeflow.org/inference-service-id
MODEL_VERSION_ID_LABEL=modelregistry.kubeflow.org/model-version-id
REGISTERED_MODEL_ID_LABEL=modelregistry.kubeflow.org/registered-model-id
FINALIZER=modelregistry.kubeflow.org/finalizer
SERVICE_ANNOTATION=modelregistry.kubeflow.org/service-url

# Default MR namespace
REGISTRIES_NAMESPACE=kubeflow

# TLS configuration
SKIP_TLS_VERIFY=false
```

## Deployment

### Kustomize Deployment

```bash
# Deploy controller
kubectl apply -k manifests/kustomize/options/controller/overlays/base -n kubeflow
```

### Manifest Structure

```
manifests/kustomize/options/controller/
├── default/
│   └── kustomization.yaml
├── manager/
│   ├── manager.yaml              # Deployment spec
│   └── kustomization.yaml
├── rbac/
│   ├── role.yaml                 # ClusterRole
│   ├── role_binding.yaml         # ClusterRoleBinding
│   ├── service_account.yaml      # ServiceAccount
│   └── kustomization.yaml
├── metrics/
│   ├── metrics_service.yaml      # Prometheus metrics
│   └── kustomization.yaml
├── network-policy/
│   ├── allow-metrics-traffic.yaml
│   └── kustomization.yaml
└── overlays/
    └── base/
        └── kustomization.yaml
```

### RBAC Permissions

The controller requires the following permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: inferenceservice-controller
rules:
# InferenceService access
- apiGroups: ["serving.kserve.io"]
  resources: ["inferenceservices"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: ["serving.kserve.io"]
  resources: ["inferenceservices/finalizers"]
  verbs: ["update"]
# Service discovery
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch"]
```

## Status Management

### InferenceService States

| State | Description |
|-------|-------------|
| DEPLOYED | Service should be active and serving |
| UNDEPLOYED | Service should be inactive/removed |

### URL Synchronization

The controller tracks deployment URLs:

1. Extracts URL from KServe ISVC status
2. Stores in Model Registry as `customProperties["url"]`
3. Updates when URL changes
4. Supports external URL annotation override

## Error Handling

### Reconciliation Patterns

- **Missing labels**: Skip reconciliation silently
- **MR service not found**: Auto-detect if only one MR in namespace
- **Transient errors**: Return for immediate requeue
- **Deletion**: Use finalizers for graceful cleanup

### Common Issues

**MR service not specified:**
```
Specify the namespace and the name of the Model Registry service to use
by adding labels to the InferenceService.
```

**Multiple MR services:**
```
More than one Model Registry service found in namespace.
Specify which one to use via labels.
```

## Testing

### Unit Tests

```bash
cd internal/controller/controllers
go test -v ./...
```

### Integration Tests

```bash
cd pkg/inferenceservice-controller
go test -v ./...
```

### Test Scenarios

1. InferenceService creation and ID assignment
2. URL reconciliation and updates
3. Deletion with desired state transition
4. Multiple Model Registry handling
5. Missing label scenarios

## Security

### Pod Security

- Non-root execution (user 65532)
- Dropped capabilities
- SeccompProfile: RuntimeDefault

### Network Security

- Metrics endpoint: 8443 (HTTPS) or 8080 (HTTP)
- Health checks: 8081
- NetworkPolicies for metrics access

---

[Back to Kubernetes Index](./README.md) | [Next: CSI Driver](./csi-driver.md)
