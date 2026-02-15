# CSI Storage Initializer

This document covers the Custom Storage Initializer (CSI) that enables KServe to serve models indexed in the Model Registry.

## Overview

The Model Registry CSI is a KServe-compliant storage initialization driver that bridges KServe inference service deployments with models stored in the Model Registry. It registers the `model-registry://` protocol and resolves Model Registry URIs to actual storage locations.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    CSI Storage Initializer                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              InferenceService Pod                            │ │
│  │                                                               │ │
│  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │             Init Container (CSI Driver)                  │ │ │
│  │  │                                                           │ │ │
│  │  │  Input:  model-registry://model-name/version             │ │ │
│  │  │                      │                                    │ │ │
│  │  │                      ▼                                    │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐   │ │ │
│  │  │  │        Model Registry API Client                   │   │ │ │
│  │  │  │  - Resolve model name and version                  │   │ │ │
│  │  │  │  - Fetch ModelArtifact URI                         │   │ │ │
│  │  │  │  - Get actual storage location                     │   │ │ │
│  │  │  └───────────────────────────────────────────────────┘   │ │ │
│  │  │                      │                                    │ │ │
│  │  │                      ▼                                    │ │ │
│  │  │  Actual URI: s3://bucket/model.onnx                      │ │ │
│  │  │                      │                                    │ │ │
│  │  │                      ▼                                    │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐   │ │ │
│  │  │  │         KServe Provider (S3, GCS, HTTP...)        │   │ │ │
│  │  │  │  - Download from actual storage                    │   │ │ │
│  │  │  │  - Place in /mnt/models                            │   │ │ │
│  │  │  └───────────────────────────────────────────────────┘   │ │ │
│  │  │                                                           │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  │                                                               │ │
│  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │            Main Container (Inference Server)             │ │ │
│  │  │                                                           │ │ │
│  │  │  Model loaded from: /mnt/models                          │ │ │
│  │  │                                                           │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  │                                                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## File Structure

```
cmd/csi/
├── main.go                    # Entry point
├── Dockerfile.csi             # Container build
├── README.md                  # Documentation
├── GET_STARTED.md            # Quick start guide
└── samples/
    └── modelregistry.clusterstoragecontainer.yaml

internal/csi/
├── constants/
│   └── constants.go           # Protocol definition
├── modelregistry/
│   └── api_client.go          # MR API interaction
└── storage/
    └── modelregistry_provider.go  # Provider implementation
```

## URI Formats

The CSI driver supports four URI formats:

| Format | Example |
|--------|---------|
| Model only | `model-registry://iris` |
| Model + version | `model-registry://iris/v1` |
| Custom MR + model | `model-registry://registry.example.com:8080/iris` |
| Custom MR + model + version | `model-registry://registry.example.com:8080/iris/v1` |

### URI Parsing Logic

```
model-registry://[host[:port]/]modelName[/modelVersion]

Examples:
  model-registry://iris
    → modelName=iris, version=latest

  model-registry://iris/v1
    → modelName=iris, version=v1

  model-registry://registry.kubeflow.svc:8080/iris
    → host=registry.kubeflow.svc:8080, modelName=iris, version=latest

  model-registry://registry.kubeflow.svc:8080/iris/v1
    → host=registry.kubeflow.svc:8080, modelName=iris, version=v1
```

## How It Works

### Execution Flow

1. **Input**: KServe invokes CSI with two arguments:
   - Source URI: `model-registry://...`
   - Destination Path: `/mnt/models`

2. **URI Parsing**: Extract model name, version, and optional MR URL

3. **Model Resolution**:
   - Query Model Registry API for RegisteredModel
   - Get ModelVersion (specific or latest)
   - Retrieve ModelArtifact with actual storage URI

4. **Model Download**:
   - Identify storage protocol (S3, GCS, HTTP, etc.)
   - Delegate to appropriate KServe provider
   - Download to destination path

### ModelRegistryProvider

The core provider implements KServe's `Provider` interface:

```go
// storage/modelregistry_provider.go
type ModelRegistryProvider struct {
    Client     *api.Client
    Providers  map[string]storage.Provider
}

func (p *ModelRegistryProvider) DownloadModel(
    modelDir string,
    modelName string,
    storageUri string,
) error {
    // 1. Parse URI
    registeredModelName, version := parseModelVersion(storageUri)

    // 2. Fetch from Model Registry
    modelVersion := fetchModelVersion(registeredModelName, version)

    // 3. Get artifact URI
    artifacts := getModelArtifacts(modelVersion.ID)
    artifactURI := artifacts[0].URI  // Most recent

    // 4. Delegate to storage provider
    protocol := extractProtocol(artifactURI)
    provider := p.Providers[protocol]
    return provider.DownloadModel(modelDir, modelName, artifactURI)
}
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MODEL_REGISTRY_BASE_URL` | Model Registry service URL | `localhost:8080` |
| `MODEL_REGISTRY_SCHEME` | HTTP scheme | `http` |

### ClusterStorageContainer

Register the CSI driver with KServe:

```yaml
apiVersion: serving.kserve.io/v1alpha1
kind: ClusterStorageContainer
metadata:
  name: model-registry-storage-initializer
spec:
  container:
    name: storage-initializer
    image: ghcr.io/kubeflow/model-registry/storage-initializer:latest
    env:
    - name: MODEL_REGISTRY_BASE_URL
      value: "model-registry-service.kubeflow.svc.cluster.local:8080"
    - name: MODEL_REGISTRY_SCHEME
      value: "http"
    resources:
      requests:
        memory: 100Mi
        cpu: 100m
      limits:
        memory: 1Gi
  supportedUriFormats:
    - prefix: model-registry://
```

## Usage

### Example InferenceService

```yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: iris-model
  namespace: ml-models
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
      storageUri: "model-registry://iris/v1"
```

### With Custom Registry

```yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: iris-model
  namespace: ml-models
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
      storageUri: "model-registry://registry.custom.svc:8080/iris/v1"
```

## Supported Storage Backends

The CSI driver delegates to KServe's provider system:

| Protocol | Description |
|----------|-------------|
| `s3://` | Amazon S3 / S3-compatible |
| `gs://` | Google Cloud Storage |
| `https://` | HTTP/HTTPS endpoints |
| `hdfs://` | Hadoop Distributed File System |
| `wasbs://` | Azure Blob Storage |

## Model Registry Integration

### Data Flow

```
RegisteredModel (iris)
    │
    └─► ModelVersion (v1)
            │
            └─► ModelArtifact
                    │
                    └─► URI: s3://ml-models/iris/v1/model.pkl
```

### API Calls

1. `GET /api/model_registry/v1alpha3/registered_models?name=iris`
2. `GET /api/model_registry/v1alpha3/model_versions?registeredModelId=1&name=v1`
   or `GET /api/model_registry/v1alpha3/registered_models/1/versions` (for latest)
3. `GET /api/model_registry/v1alpha3/model_versions/1/artifacts`

## Deployment

### Kustomize Deployment

```bash
# Cluster-scoped (once per cluster)
kubectl apply -k manifests/kustomize/options/csi
```

### Manifest Structure

```
manifests/kustomize/options/csi/
├── clusterstoragecontainer.yaml
└── kustomization.yaml
```

## Building

### Docker Build

```bash
# From repository root
make docker-build-csi

# With custom image
make IMG=myregistry/storage-initializer:v1 docker-build-csi
make docker-push-csi
```

### Dockerfile

The CSI driver uses a multi-stage build:

```dockerfile
# Builder stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.22
WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o mr-storage-initializer ./cmd/csi

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5
COPY --from=builder /workspace/mr-storage-initializer /mr-storage-initializer
USER 65532:65532
ENTRYPOINT ["/mr-storage-initializer"]
```

## Error Handling

### Error Types

| Error | Description |
|-------|-------------|
| `ErrInvalidMRURI` | Invalid URI format |
| `ErrNoVersionAssociated` | No model versions found |
| `ErrNoArtifactAssociated` | No artifacts for version |
| `ErrProtocolNotSupported` | Unsupported storage protocol |
| `ErrModelArtifactEmptyURI` | Artifact has empty URI |

### Fallback Behavior

- If custom URL parsing fails, falls back to base URL
- If version not specified, retrieves latest version
- Artifacts sorted by creation time (most recent first)

## Testing

### E2E Test Structure

```bash
test/csi/
├── e2e_test.sh           # Test scenarios
├── setup_test_env.sh     # Environment setup
└── test_utils.sh         # Utilities
```

### Test Scenarios

1. **Default MR with explicit version**
   - `model-registry://iris/v1`

2. **Default MR without version (latest)**
   - `model-registry://iris`

3. **Custom MR namespace with version**
   - `model-registry://registry.custom:8080/iris/v1`

4. **Custom MR namespace without version**
   - `model-registry://registry.custom:8080/iris`

### Running Tests

```bash
# Setup test environment
./test/csi/setup_test_env.sh

# Run E2E tests
./test/csi/e2e_test.sh
```

## Security

### Pod Security

- Non-root user (65532)
- Minimal base image (UBI 9 minimal)
- Read-only filesystem where possible

### Network Access

- Requires access to Model Registry API
- Requires access to storage backends (S3, GCS, etc.)

---

[Back to Kubernetes Index](./README.md) | [Previous: Controller](./controller.md) | [Next: Deployment Manifests](./deployment-manifests.md)
