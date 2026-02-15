# Clients Documentation

This section covers client libraries for interacting with the Kubeflow Model Registry.

## Overview

The Model Registry provides client libraries that abstract the REST API into language-specific interfaces:

- **Python Client**: Full-featured async/sync client with upload support
- **Go OpenAPI Client**: Auto-generated from OpenAPI specification

## Documentation

| Document | Description |
|----------|-------------|
| [Python Client](./python-client.md) | Python client installation, usage, and API reference |

## Client Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Libraries                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────┐    ┌─────────────────────────────┐ │
│  │     Python Client       │    │     Go OpenAPI Client       │ │
│  │  (model_registry pkg)   │    │   (pkg/openapi module)      │ │
│  └───────────┬─────────────┘    └────────────┬────────────────┘ │
│              │                                │                  │
│              │  High-level wrapper            │ Generated        │
│              ▼                                ▼                  │
│  ┌─────────────────────────┐    ┌─────────────────────────────┐ │
│  │   mr_openapi (generated) │    │  OpenAPI Spec v1alpha3      │ │
│  │   async HTTP client      │    │  api/openapi/model-registry │ │
│  └───────────┬─────────────┘    └────────────┬────────────────┘ │
│              │                                │                  │
│              └────────────┬──────────────────┘                  │
│                           ▼                                      │
│              ┌─────────────────────────┐                        │
│              │    REST API Endpoints   │                        │
│              │   /api/model_registry/  │                        │
│              │        v1alpha3         │                        │
│              └─────────────────────────┘                        │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Python

```bash
pip install model-registry

# With Hugging Face support
pip install "model-registry[hf]"
```

```python
from model_registry import ModelRegistry

registry = ModelRegistry(
    "https://model-registry.example.com",
    author="Data Scientist"
)

model = registry.register_model(
    name="my-model",
    uri="s3://bucket/model.onnx",
    version="1.0.0",
    model_format_name="onnx",
    model_format_version="1"
)
```

## API Versioning

| API Version | Status | Client Support |
|-------------|--------|----------------|
| v1alpha3 | Current | Python, Go |
| v1alpha2 | Deprecated | Legacy |
| v1alpha1 | Removed | None |

## Package Distribution

| Package | Distribution | Version |
|---------|--------------|---------|
| Python | PyPI (`model-registry`) | 0.3.5 |
| Go | Go Module (`github.com/kubeflow/model-registry`) | HEAD |

---

[Back to Main Index](../README.md)
