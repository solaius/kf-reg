# Data Models

This document describes the entity relationships, schemas, and property system used in the Kubeflow Model Registry.

## Overview

The Model Registry uses an **MLMD-based entity system** (ML Metadata) with three primary entity types that map to database tables:

| Entity Type | MLMD Type | Examples |
|-------------|-----------|----------|
| **Context** | Logical grouping | RegisteredModel, ServingEnvironment, Experiment |
| **Artifact** | Versioned assets | ModelVersion, ModelArtifact, DocArtifact |
| **Execution** | Operations | InferenceService, ServeModel, ExperimentRun |

## Core Entities

### RegisteredModel

The top-level entity representing a model in the registry.

```go
type RegisteredModel struct {
    ID                       string
    Name                     string
    Description              string
    Owner                    string
    State                    ModelState
    CustomProperties         map[string]MetadataValue
    CreateTimeSinceEpoch     int64
    LastUpdateTimeSinceEpoch int64
}
```

**MLMD Mapping:** `Context` type

**Relationships:**
- Has many `ModelVersion` (parent)
- Has many `InferenceService` (via property reference)

### ModelVersion

A specific version of a registered model.

```go
type ModelVersion struct {
    ID                       string
    Name                     string
    Description              string
    Author                   string
    State                    ModelVersionState
    RegisteredModelId        string
    CustomProperties         map[string]MetadataValue
    CreateTimeSinceEpoch     int64
    LastUpdateTimeSinceEpoch int64
}
```

**MLMD Mapping:** `Artifact` type

**Relationships:**
- Belongs to `RegisteredModel` (child)
- Has many `Artifact` (ModelArtifact, DocArtifact)

**Naming Convention:** Stored as `{RegisteredModelId}:{VersionName}` in database

### Artifact Types

#### ModelArtifact

Binary model artifact (weights, model files).

```go
type ModelArtifact struct {
    ID                       string
    Name                     string
    URI                      string
    Description              string
    ModelFormatName          string
    ModelFormatVersion       string
    StorageKey               string
    StoragePath              string
    ServiceAccountName       string
    State                    ArtifactState
    CustomProperties         map[string]MetadataValue
}
```

#### DocArtifact

Documentation artifact associated with a model.

```go
type DocArtifact struct {
    ID                       string
    Name                     string
    URI                      string
    Description              string
    State                    ArtifactState
    CustomProperties         map[string]MetadataValue
}
```

### Serving Entities

#### ServingEnvironment

Environment configuration for model serving.

```go
type ServingEnvironment struct {
    ID                       string
    Name                     string
    Description              string
    CustomProperties         map[string]MetadataValue
}
```

**MLMD Mapping:** `Context` type

#### InferenceService

Deployed inference endpoint.

```go
type InferenceService struct {
    ID                       string
    Name                     string
    Description              string
    ServingEnvironmentId     string
    RegisteredModelId        string
    ModelVersionId           string
    Runtime                  string
    State                    InferenceServiceState
    CustomProperties         map[string]MetadataValue
}
```

**MLMD Mapping:** `Execution` type

#### ServeModel

Action of serving a model version.

```go
type ServeModel struct {
    ID                       string
    Name                     string
    LastKnownState           ExecutionState
    ModelVersionId           string
    CustomProperties         map[string]MetadataValue
}
```

**MLMD Mapping:** `Execution` type

### Experiment Tracking

#### Experiment

Logical grouping of experiment runs.

```go
type Experiment struct {
    ID                       string
    Name                     string
    Description              string
    ExternalId               string
    CustomProperties         map[string]MetadataValue
}
```

**MLMD Mapping:** `Context` type

#### ExperimentRun

Individual experiment execution.

```go
type ExperimentRun struct {
    ID                       string
    Name                     string
    State                    ExperimentRunState
    Description              string
    ExperimentId             string
    CustomProperties         map[string]MetadataValue
}
```

**MLMD Mapping:** `Execution` type

---

## Property System

### Overview

The property system provides typed key-value metadata storage for all entities.

### Property Types

```go
type Properties struct {
    Name             string
    IsCustomProperty bool
    IntValue         *int32
    DoubleValue      *float64
    StringValue      *string
    BoolValue        *bool
    ByteValue        *[]byte
    ProtoValue       *[]byte
}
```

### Metadata Value Types

For API-level representation:

| Type | Go Type | JSON Example |
|------|---------|--------------|
| `MetadataStringValue` | `string` | `{"string_value": "text"}` |
| `MetadataIntValue` | `int64` | `{"int_value": 42}` |
| `MetadataDoubleValue` | `float64` | `{"double_value": 3.14}` |
| `MetadataBoolValue` | `bool` | `{"bool_value": true}` |

### Standard Properties

Certain properties have special meaning:

| Property | Entity | Purpose |
|----------|--------|---------|
| `registered_model_id` | InferenceService | Links to RegisteredModel |
| `model_version_id` | InferenceService, ServeModel | Links to ModelVersion |
| `serving_environment_id` | InferenceService | Links to ServingEnvironment |
| `experiment_id` | ExperimentRun | Links to Experiment |

### Custom Properties

User-defined properties for extensible metadata:

```yaml
customProperties:
  model_type:
    metadataType: MetadataStringValue
    string_value: "generative"
  accuracy:
    metadataType: MetadataDoubleValue
    double_value: 0.95
  validated:
    metadataType: MetadataBoolValue
    bool_value: true
```

---

## Database Schema

### Core Tables (MLMD-Based)

```sql
-- Type definitions
CREATE TABLE Type (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL UNIQUE,
    version VARCHAR(255),
    type_kind ENUM('CONTEXT', 'ARTIFACT', 'EXECUTION'),
    description TEXT,
    external_id VARCHAR(255)
);

-- Context entities (RegisteredModel, ServingEnvironment, Experiment)
CREATE TABLE Context (
    id INT PRIMARY KEY AUTO_INCREMENT,
    type_id INT NOT NULL REFERENCES Type(id),
    name VARCHAR(255),
    external_id VARCHAR(255),
    create_time_since_epoch BIGINT,
    last_update_time_since_epoch BIGINT
);

-- Artifact entities (ModelVersion, ModelArtifact, DocArtifact)
CREATE TABLE Artifact (
    id INT PRIMARY KEY AUTO_INCREMENT,
    type_id INT NOT NULL REFERENCES Type(id),
    uri TEXT,
    state INT,
    name VARCHAR(255),
    external_id VARCHAR(255),
    create_time_since_epoch BIGINT,
    last_update_time_since_epoch BIGINT
);

-- Execution entities (InferenceService, ServeModel, ExperimentRun)
CREATE TABLE Execution (
    id INT PRIMARY KEY AUTO_INCREMENT,
    type_id INT NOT NULL REFERENCES Type(id),
    name VARCHAR(255),
    external_id VARCHAR(255),
    last_known_state INT,
    create_time_since_epoch BIGINT,
    last_update_time_since_epoch BIGINT
);
```

### Property Tables

```sql
-- Context properties
CREATE TABLE ContextProperty (
    context_id INT NOT NULL REFERENCES Context(id),
    name VARCHAR(255) NOT NULL,
    is_custom_property BOOLEAN,
    int_value INT,
    double_value DOUBLE,
    string_value TEXT,
    bool_value BOOLEAN,
    byte_value BLOB,
    proto_value BLOB,
    PRIMARY KEY (context_id, name, is_custom_property)
);

-- Artifact properties
CREATE TABLE ArtifactProperty (
    artifact_id INT NOT NULL REFERENCES Artifact(id),
    name VARCHAR(255) NOT NULL,
    is_custom_property BOOLEAN,
    -- ... same value columns
    PRIMARY KEY (artifact_id, name, is_custom_property)
);

-- Execution properties
CREATE TABLE ExecutionProperty (
    execution_id INT NOT NULL REFERENCES Execution(id),
    name VARCHAR(255) NOT NULL,
    is_custom_property BOOLEAN,
    -- ... same value columns
    PRIMARY KEY (execution_id, name, is_custom_property)
);
```

### Relationship Tables

```sql
-- Context-to-Context relationships
CREATE TABLE ParentContext (
    context_id INT NOT NULL REFERENCES Context(id),
    parent_context_id INT NOT NULL REFERENCES Context(id),
    PRIMARY KEY (context_id, parent_context_id)
);

-- Artifact attribution to Context
CREATE TABLE Attribution (
    context_id INT NOT NULL REFERENCES Context(id),
    artifact_id INT NOT NULL REFERENCES Artifact(id),
    PRIMARY KEY (context_id, artifact_id)
);

-- Execution association with Context
CREATE TABLE Association (
    context_id INT NOT NULL REFERENCES Context(id),
    execution_id INT NOT NULL REFERENCES Execution(id),
    PRIMARY KEY (context_id, execution_id)
);

-- Execution-to-Artifact events
CREATE TABLE Event (
    id INT PRIMARY KEY AUTO_INCREMENT,
    artifact_id INT NOT NULL REFERENCES Artifact(id),
    execution_id INT NOT NULL REFERENCES Execution(id),
    type INT,
    milliseconds_since_epoch BIGINT
);
```

---

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            CONTEXT TYPES                                     │
│                                                                              │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐      │
│  │  RegisteredModel │    │ServingEnvironment│    │   Experiment     │      │
│  │                  │    │                  │    │                  │      │
│  │  - name          │    │  - name          │    │  - name          │      │
│  │  - description   │    │  - description   │    │  - description   │      │
│  │  - owner         │    │  - customProps   │    │  - externalId    │      │
│  │  - state         │    │                  │    │  - customProps   │      │
│  └────────┬─────────┘    └────────┬─────────┘    └────────┬─────────┘      │
│           │                       │                       │                 │
└───────────┼───────────────────────┼───────────────────────┼─────────────────┘
            │ 1:N                   │ 1:N                   │ 1:N
            ▼                       ▼                       ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│                            ARTIFACT TYPES                                      │
│                                                                                │
│  ┌──────────────────┐                                                         │
│  │   ModelVersion   │────────────────────────────┐                            │
│  │                  │                            │ 1:N                        │
│  │  - name          │                            ▼                            │
│  │  - description   │              ┌──────────────────┐  ┌──────────────────┐ │
│  │  - author        │              │  ModelArtifact   │  │   DocArtifact    │ │
│  │  - state         │              │                  │  │                  │ │
│  │  - regModelId    │              │  - uri           │  │  - uri           │ │
│  └──────────────────┘              │  - modelFormat   │  │  - description   │ │
│                                    │  - storageKey    │  └──────────────────┘ │
│                                    └──────────────────┘                       │
└───────────────────────────────────────────────────────────────────────────────┘
            │
            │ Referenced by
            ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│                           EXECUTION TYPES                                      │
│                                                                                │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐        │
│  │ InferenceService │    │    ServeModel    │    │  ExperimentRun   │        │
│  │                  │    │                  │    │                  │        │
│  │  - name          │    │  - name          │    │  - name          │        │
│  │  - runtime       │    │  - lastState     │    │  - state         │        │
│  │  - state         │    │  - modelVersionId│    │  - experimentId  │        │
│  │  - regModelId    │    │                  │    │                  │        │
│  │  - modelVersionId│    │                  │    │                  │        │
│  │  - servEnvId     │    │                  │    │                  │        │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘        │
│                                                                                │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## Catalog Service Data Models

The Catalog Service uses additional models for federated discovery:

### CatalogModel

```go
type CatalogModel struct {
    Name                     string
    Description              string
    Readme                   string
    Maturity                 string
    Language                 []string
    Tasks                    []string
    LibraryName              string
    License                  string
    LicenseLink              string
    Provider                 string
    CustomProperties         map[string]MetadataValue
    CreateTimeSinceEpoch     int64
    LastUpdateTimeSinceEpoch int64
}
```

### CatalogArtifact

```go
type CatalogArtifact struct {
    URI              string
    CustomProperties map[string]MetadataValue
}
```

### CatalogSource

```go
type CatalogSource struct {
    ID        string
    Name      string
    Enabled   bool
    Labels    []string
    AssetType CatalogAssetType
}
```

---

## Type Registration

Types are registered at application startup:

```go
var RegisteredModelTypeName = "odh.RegisteredModel"
var ModelVersionTypeName = "odh.ModelVersion"
var ModelArtifactTypeName = "odh.ModelArtifact"
var DocArtifactTypeName = "odh.DocArtifact"
var ServingEnvironmentTypeName = "odh.ServingEnvironment"
var InferenceServiceTypeName = "odh.InferenceService"
var ServeModelTypeName = "odh.ServeModel"
var ExperimentTypeName = "odh.Experiment"
var ExperimentRunTypeName = "odh.ExperimentRun"
```

---

[Back to Architecture Index](./README.md) | [Previous: Tech Stack](./tech-stack.md) | [Next: API Design](./api-design.md)
