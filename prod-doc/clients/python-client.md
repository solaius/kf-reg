# Python Client Documentation

This document covers the Python client for the Kubeflow Model Registry.

## Overview

The Python client provides a high-level, Pythonic interface for interacting with the Model Registry API. It wraps an auto-generated OpenAPI client with convenient methods for model registration, versioning, and experiment tracking.

## Installation

```bash
# Basic installation
pip install model-registry

# With Hugging Face Hub support
pip install "model-registry[hf]"

# With S3/boto3 support
pip install "model-registry[boto3]"

# With OCI registry support
pip install "model-registry[olot]"

# All extras
pip install "model-registry[hf,boto3,olot]"
```

## Package Structure

```
clients/python/
├── src/
│   ├── model_registry/              # High-level Pythonic wrapper
│   │   ├── __init__.py              # Package entry (exports ModelRegistry)
│   │   ├── _client.py               # Main ModelRegistry client class
│   │   ├── _experiments.py          # Experiment run tracking
│   │   ├── core.py                  # Low-level API client wrapper
│   │   ├── exceptions.py            # Custom exception classes
│   │   ├── utils.py                 # Utility functions (S3, OCI, URI)
│   │   └── types/                   # Type definitions
│   │       ├── artifacts.py         # Artifact, ModelArtifact, Metric
│   │       ├── contexts.py          # RegisteredModel, ModelVersion
│   │       ├── experiments.py       # Experiment, ExperimentRun
│   │       ├── base.py              # BaseResourceModel
│   │       ├── options.py           # ListOptions, ArtifactTypeQueryParam
│   │       └── pager.py             # Pager for pagination
│   └── mr_openapi/                  # Auto-generated OpenAPI client
│       ├── api/                     # Generated API service
│       ├── models/                  # ~70+ model classes
│       ├── api_client.py            # Generic client
│       └── configuration.py         # Connection config
├── pyproject.toml                   # Poetry configuration
├── Makefile                         # Build targets
└── templates/                       # OpenAPI generator templates
```

## Quick Start

### Basic Connection

```python
from model_registry import ModelRegistry

# Secure connection (TLS)
registry = ModelRegistry(
    "https://model-registry.example.com",
    author="Data Scientist",
    user_token="<token>"
)

# Insecure connection (HTTP)
registry = ModelRegistry(
    "http://localhost",
    port=8080,
    author="Developer",
    is_secure=False
)
```

### Register a Model

```python
model = registry.register_model(
    name="fraud-detector",
    uri="s3://ml-models/fraud-detection/model.onnx",
    version="1.0.0",
    model_format_name="onnx",
    model_format_version="1",
    author="ML Team",
    metadata={
        "accuracy": 0.95,
        "dataset": "transactions-v2",
        "framework": "scikit-learn"
    }
)

print(f"Registered: {model.name} (ID: {model.id})")
```

### Retrieve Models

```python
# Get specific model
model = registry.get_registered_model("fraud-detector")

# Get specific version
version = registry.get_model_version("fraud-detector", "1.0.0")

# Get artifact
artifact = registry.get_model_artifact("fraud-detector", "1.0.0")

# List all models with pagination
for model in registry.get_registered_models().page_size(10):
    print(f"{model.name}: {model.state}")

# List versions for a model
for version in registry.get_model_versions("fraud-detector"):
    print(f"Version: {version.name}")
```

## Client Class

### Constructor Parameters

```python
ModelRegistry(
    server_address: str,           # Server URL
    port: int = 443,               # Server port
    author: str,                   # Name of author (required)
    is_secure: bool = True,        # Use TLS
    user_token: str = None,        # PEM-encoded token
    user_token_envvar: str = None, # Env var for token
    custom_ca: str = None,         # Path to CA certificate
    custom_ca_envvar: str = None,  # Env var for CA path
    log_level: int = logging.WARNING,
    async_runner: Callable = None  # Custom async executor
)
```

### Authentication

**Token Sources (Priority Order):**

1. `user_token` parameter (explicit)
2. `user_token_envvar` parameter (custom env var)
3. `KF_PIPELINES_SA_TOKEN_PATH` environment variable
4. `/var/run/secrets/kubernetes.io/serviceaccount/token` (K8s SA)

```python
# Explicit token
registry = ModelRegistry(
    "https://server",
    author="user",
    user_token="<pem-encoded-token>"
)

# Environment variable
import os
os.environ["MY_TOKEN_PATH"] = "/path/to/token"

registry = ModelRegistry(
    "https://server",
    author="user",
    user_token_envvar="MY_TOKEN_PATH"
)
```

### Custom CA Certificates

```python
registry = ModelRegistry(
    "https://internal-registry.corp.com",
    author="user",
    custom_ca="/etc/ssl/certs/internal-ca.pem"
)

# Or via environment variable
registry = ModelRegistry(
    "https://internal-registry.corp.com",
    author="user",
    custom_ca_envvar="CUSTOM_CA_PATH"
)
```

## API Methods

### Model Registration

```python
# Basic registration
register_model(
    name: str,
    uri: str,
    version: str,
    model_format_name: str,
    model_format_version: str,
    storage_key: str = None,
    storage_path: str = None,
    author: str = None,         # Defaults to client author
    owner: str = None,          # Defaults to client author
    version_description: str = None,
    metadata: Mapping[str, SupportedTypes] = None
) -> RegisteredModel

# Upload and register
upload_artifact_and_register_model(
    name: str,
    model_files_path: str,
    upload_params: S3Params | OCIParams,
    version: str,
    model_format_name: str,
    model_format_version: str,
    ...
) -> RegisteredModel

# Register from Hugging Face
register_hf_model(
    repo: str,
    path: str,
    version: str,
    model_format_name: str,
    model_format_version: str,
    model_name: str = None,
    author: str = None,
    ...
) -> RegisteredModel
```

### Model Retrieval

```python
get_registered_model(name: str) -> RegisteredModel | None
get_model_version(name: str, version: str) -> ModelVersion | None
get_model_artifact(name: str, version: str) -> ModelArtifact | None
get_registered_models() -> Pager[RegisteredModel]
get_model_versions(name: str) -> Pager[ModelVersion]
```

### Model Updates

```python
# Generic update for any model type
update(model: TModel) -> TModel

# Example
model = registry.get_registered_model("my-model")
model.description = "Updated description"
registry.update(model)
```

## Pagination

The `Pager` class provides fluent pagination:

```python
# Configure pagination
models = registry.get_registered_models()\
    .page_size(10)\
    .order_by_creation_time()\
    .ascending()

# Synchronous iteration
for model in models:
    print(model.name)

# Async iteration
async for model in models:
    print(model.name)
```

## Experiment Tracking

```python
import json

# Start experiment run (context manager)
with registry.start_experiment_run(
    experiment_name="Model Training V2",
    run_name="run-001"
) as run:
    # Log parameters
    run.log_param("learning_rate", 0.001)
    run.log_param("epochs", 100)

    # Log metrics
    run.log_metric("accuracy", 0.95, step=100)
    run.log_metric("loss", 0.05, step=100)

    # Log dataset
    run.log_dataset(
        name="training_data",
        source_type="local",
        uri="s3://datasets/train.csv",
        schema=json.dumps({"features": [...]}),
        profile="v1"
    )

# Access run info after context exit
print(f"Experiment: {run.info.experiment_id}")
print(f"Run: {run.info.id}")
```

### Nested Runs

```python
with registry.start_experiment_run(
    experiment_name="Hyperparameter Tuning",
    run_name="parent-run"
) as parent:
    parent.log_param("search_space", "grid")

    # Nested child runs
    for lr in [0.001, 0.01, 0.1]:
        with registry.start_experiment_run(
            run_name=f"child-lr-{lr}",
            nested=True
        ) as child:
            child.log_param("learning_rate", lr)
            child.log_metric("accuracy", train_model(lr))
```

## S3 Integration

```python
from model_registry.utils import S3Params

# Upload to S3 and register
model = registry.upload_artifact_and_register_model(
    name="fraud-detector",
    model_files_path="/path/to/model/",
    author="ML Team",
    version="0.1.0",
    upload_params=S3Params(
        bucket_name="ml-models",
        s3_prefix="fraud-detection/v0.1",
        endpoint_url="https://s3.amazonaws.com",  # Optional
        access_key_id="...",                       # Optional
        secret_access_key="..."                    # Optional
    ),
    model_format_name="sklearn",
    model_format_version="1.0"
)

# Build S3 URI utility
from model_registry.utils import s3_uri_from

uri = s3_uri_from(
    path="models/my-model.pkl",
    bucket="ml-bucket",
    region="us-west-2"
)
```

## Type System

### Supported Metadata Types

```python
from typing import Union

SupportedTypes = Union[bool, int, float, str]

# Automatic type mapping
metadata = {
    "bool_key": True,          # MetadataBoolValue
    "int_key": 42,             # MetadataIntValue
    "float_key": 3.14,         # MetadataDoubleValue
    "str_key": "value",        # MetadataStringValue
}
```

### Type Hierarchy

```
BaseResourceModel (ABC)
├── Artifact (ABC)
│   ├── ModelArtifact
│   ├── DocArtifact
│   ├── DataSet
│   ├── Metric
│   └── Parameter
├── RegisteredModel
└── ModelVersion

Experiment (BaseResourceModel)
ExperimentRun (BaseResourceModel)
```

## Exceptions

```python
from model_registry.exceptions import (
    StoreError,
    UnsupportedType,
    TypeNotFound,
    ServerError,
    DuplicateError,
    MissingMetadata,
    ExperimentRunError
)

try:
    registry.register_model(...)
except DuplicateError:
    print("Model already exists")
except ServerError as e:
    print(f"Server error: {e}")
except StoreError as e:
    print(f"Store error: {e}")
```

## Async Operations

The client uses async internally but provides sync wrappers:

```python
# Default: Sync wrapper with nest_asyncio
registry = ModelRegistry("https://server", author="user")

# Custom async runner (for Ray, Uvloop, etc.)
class CustomRunner:
    def run(self, coro):
        # Custom async execution
        return custom_loop.run_until_complete(coro)

registry = ModelRegistry(
    "https://server",
    author="user",
    async_runner=CustomRunner().run
)
```

## Code Generation

The `mr_openapi` package is auto-generated from the OpenAPI spec:

```bash
# From clients/python/
make generate  # Regenerates from OpenAPI spec

# Generation process
# 1. openapi-generator-cli v7.17.0
# 2. Python generator with asyncio library
# 3. Apply patches (asyncio-only, fix-validators)
# 4. Format with ruff
```

## Dependencies

**Core:**
- `python >= 3.9, < 4.0`
- `pydantic ^2.7.4`
- `aiohttp ^3.9.5`
- `aiohttp-retry ^2.8.3`
- `nest-asyncio ^1.6.0`

**Optional:**
- `huggingface-hub` (hf extra)
- `boto3` (boto3 extra)
- `olot` (olot extra)

## Development

```bash
cd clients/python

# Install dependencies
make install

# Run tests
make test

# Run E2E tests (requires cluster)
make test-e2e

# Lint
make lint

# Format
make tidy

# Build
make build
```

---

[Back to Clients Index](./README.md) | [Back to Main Index](../README.md)
