# Programming Guidelines

This document describes the coding conventions, architecture patterns, and development practices for the Kubeflow Model Registry project.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Repository Structure](#repository-structure)
- [Go Guidelines](#go-guidelines)
- [Python Guidelines](#python-guidelines)
- [TypeScript/UI Guidelines](#typescriptui-guidelines)
- [API Design](#api-design)
- [Database Patterns](#database-patterns)
- [Code Generation](#code-generation)
- [Testing](#testing)
- [Build and CI/CD](#build-and-cicd)
- [Docker and Deployment](#docker-and-deployment)
- [Contribution Workflow](#contribution-workflow)
- [Adding New Entities to the Model Catalog](#adding-new-entities-to-the-model-catalog)

---

## Architecture Overview

The Model Registry is a multi-component system providing a central repository for ML model metadata. It follows a **contract-first** design where the OpenAPI specification is the source of truth for the REST API.

### Architectural Layers

```
OpenAPI Spec (api/openapi/)
    |
    v
Generated Server Stubs (internal/server/openapi/)
    |
    v
Core Business Logic (internal/core/) -- implements --> ModelRegistryApi (pkg/api/)
    |
    v
Data Mapping (internal/converter/, internal/mapper/)
    |
    v
Database Layer (internal/db/, internal/datastore/) -- via --> GORM (MySQL / PostgreSQL)
```

### Core Domain Entities

- **RegisteredModel** -- a logical ML model (e.g., a Git repo)
- **ModelVersion** -- a specific version of a RegisteredModel
- **ModelArtifact** / **DocArtifact** -- artifact files (ONNX, Pickle, docs, etc.)
- **ServingEnvironment** -- environment for model serving
- **InferenceService** -- service for model inference
- **ServeModel** -- a model being served
- **Experiment** / **ExperimentRun** -- ML experiments and their runs

---

## Repository Structure

```
api/openapi/              # OpenAPI specifications (source of truth)
  src/                    # Source YAML files merged into final spec
bin/                      # Locally installed tool binaries (gitignored)
catalog/                  # Catalog subsystem (separate Go module)
clients/
  python/                 # Python client (Poetry-based)
  ui/
    bff/                  # Go Backend-for-Frontend
    frontend/             # React/Next.js frontend
cmd/
  controller/             # Kubernetes controller entrypoint
  csi/                    # Container Storage Interface driver
  config/                 # Configuration CLI
internal/
  apiutils/               # API utility functions
  controller/             # Kubernetes controller logic
  core/                   # Core business logic (ModelRegistryService)
  converter/              # Type converters (goverter-generated)
  csi/                    # CSI driver logic
  datastore/              # Database connector abstraction
    embedmd/
      mysql/migrations/   # MySQL migration files
      postgres/migrations/# PostgreSQL migration files
  db/
    dbutil/               # Database error handling utilities
    filter/               # Filter query parser (Participle DSL)
    models/               # GORM model wrappers
    schema/               # Generated GORM schema structs
    scopes/               # GORM query scopes (pagination, etc.)
    service/              # Generic repository implementations
  mapper/                 # Data mapping layer
  server/openapi/         # Generated OpenAPI server stubs
jobs/async-upload/        # Background job for async model uploads
gorm-gen/                 # GORM struct generator tool
manifests/kustomize/      # Kubernetes deployment manifests
pkg/
  api/                    # Public API interface (ModelRegistryApi)
  openapi/                # Generated OpenAPI Go client/models
scripts/                  # Build and utility scripts
templates/                # OpenAPI generator templates
patches/                  # Patches for generated code
devenv/                   # Local development environment setup
test/                     # Integration tests
```

---

## Go Guidelines

### Version and Module

- **Go version**: 1.24.6 (CI runs on 1.25.3)
- **Module path**: `github.com/kubeflow/model-registry`
- Uses Go workspace (`go.work`) with local module replacements for `pkg/openapi` and `catalog/pkg/openapi`

### Key Dependencies

| Purpose | Package |
|---------|---------|
| HTTP router | `go-chi/chi/v5` |
| CORS | `go-chi/cors` |
| ORM | `gorm.io/gorm` (MySQL + PostgreSQL drivers) |
| CLI | `spf13/cobra`, `spf13/viper`, `spf13/pflag` |
| Logging | `golang/glog` |
| Testing | `stretchr/testify`, `onsi/ginkgo/v2`, `onsi/gomega` |
| Integration tests | `testcontainers-go` |
| Database migrations | `golang-migrate/migrate/v4` |
| Code generation | goverter, openapi-generator, controller-gen |
| Kubernetes | `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` |
| Filter DSL parsing | `alecthomas/participle/v2` |

### Naming Conventions

- **Exported names**: `PascalCase` (e.g., `ModelRegistryService`, `UpsertRegisteredModel`)
- **Unexported names**: `camelCase` (e.g., `upsertArtifact`, `ensureArtifactName`)
- **Files**: `snake_case.go` (e.g., `artifact.go`, `registered_model.go`)
- **Test files**: `*_test.go` suffix in the same package
- **Interfaces**: Named after the behavior they define (e.g., `ModelRegistryApi`, `Connector`)
- **Error variables**: `Err` prefix (e.g., `ErrBadRequest`, `ErrNotFound`, `ErrConflict`)

### Package Organization

Follow feature-based organization:
- Each entity type (artifact, model version, etc.) gets its own file in each layer
- `internal/core/artifact.go` implements business logic for artifacts
- `internal/db/service/artifact.go` implements the repository for artifacts
- `internal/converter/artifact.go` defines the converter interface for artifacts

### Import Conventions

Imports are grouped in three blocks separated by blank lines, enforced by `goimports`:

```go
import (
    // Standard library
    "errors"
    "fmt"

    // External dependencies
    "github.com/go-chi/chi/v5"
    "gorm.io/gorm"

    // Internal packages
    "github.com/kubeflow/model-registry/internal/apiutils"
    "github.com/kubeflow/model-registry/pkg/api"
)
```

### Error Handling

- Return explicit `error` values; never panic in library code
- Wrap errors with context using `fmt.Errorf("...: %w", err)`
- Use sentinel errors from `pkg/api` (`api.ErrBadRequest`, `api.ErrNotFound`, `api.ErrConflict`)
- Sanitize database errors before returning to users (see `internal/db/dbutil/errors.go`)
- Log internal error details with `glog.Warningf()`, return sanitized messages to callers

```go
// Good
return nil, fmt.Errorf("invalid artifact pointer, cannot be nil: %w", api.ErrBadRequest)

// Good -- sanitize DB errors
if isDatabaseTypeConversionError(err) {
    glog.Warningf("Database type conversion error: %v", err)
    return fmt.Errorf("invalid filter query, type mismatch: %w", api.ErrBadRequest)
}
```

### Interface Pattern

The core API is defined as an interface in `pkg/api/api.go`:

```go
type ModelRegistryApi interface {
    UpsertRegisteredModel(registeredModel *openapi.RegisteredModel) (*openapi.RegisteredModel, error)
    GetRegisteredModelById(id string) (*openapi.RegisteredModel, error)
    // ...
}
```

Implementations use compile-time interface assertion:

```go
var _ api.ModelRegistryApi = (*ModelRegistryService)(nil)
```

### Repository Pattern (Generics)

The data layer uses Go generics for type-safe repositories:

```go
type GenericRepository[TEntity, TSchema, TProp, TListOpts any] struct {
    // ...
}
```

Each entity type gets a concrete repository with configured mappers and filters.

### Logging

- Use `glog` for structured logging:
  - `glog.Infof()` -- informational messages
  - `glog.Warningf()` -- non-critical issues
  - `glog.Exitf()` -- fatal errors with exit
- Default flags: `--logtostderr=true`
- Mark critical alerts with `{{ALERT}}` in log messages for monitoring

### Linting

- **Tool**: golangci-lint v2.6.2
- **Config**: `.golangci.yaml` (in `clients/ui/bff/` for the BFF component)
- **Run**: `make lint`
- Go vet is run separately; the filter package is excluded due to Participle struct tags
- Generated code uses `lax` exclusion mode

### Comments and Documentation

- Package-level comments before the `package` declaration
- Exported function comments follow the `// FunctionName does X.` pattern
- Comment the "why", not the "what"
- Inline comments for non-obvious logic

```go
// ensureArtifactName ensures that an artifact has a name during creation.
// If the artifact has no ID (creation) and no name, it generates a UUID.
func ensureArtifactName(artifact *openapi.Artifact) {
```

---

## Python Guidelines

### Version and Build System

- **Python**: >= 3.10, < 4.0
- **Package manager**: Poetry
- **Test automation**: Nox
- **Source layout**: `src/model_registry/` and `src/mr_openapi/` (generated)

### Linting and Formatting (Ruff)

Configured in `pyproject.toml`:

- **Target**: Python 3.10
- **Line length**: 119
- **Excluded**: `src/mr_openapi/` (auto-generated code)

Enabled rule sets:

| Rule | Description |
|------|-------------|
| F | Pyflakes |
| W, E | Pycodestyle warnings and errors |
| C90 | McCabe complexity (max 8) |
| B | Flake8-bugbear |
| S | Flake8-bandit (security) |
| C4 | Flake8-comprehensions |
| D | Pydocstyle (Google convention) |
| EM | Flake8-errmsg |
| I | isort |
| PT | Flake8-pytest-style |
| Q | Flake8-quotes |
| RET | Flake8-return |
| SIM | Flake8-simplify |
| UP | Pyupgrade |

Ignored rules: `D105` (magic method docstrings), `E501` (line length), `S101` (assert in tests).

Tests are exempt from docstring requirements.

### Type Checking (mypy)

- Target: Python 3.10
- Not strict mode, but several warnings enabled
- `check_untyped_defs = true`
- `mr_openapi.*` module errors ignored (generated code)

### Docstring Convention

Use **Google-style** docstrings:

```python
"""Summary line.

Longer description with more details.

Args:
    param1: Description of the parameter.

Returns:
    Description of return value.

Raises:
    ValueError: When something is invalid.
"""
```

### Naming Conventions

- **Functions/variables**: `snake_case`
- **Classes**: `PascalCase`
- **Constants**: `UPPER_SNAKE_CASE`
- **Modules/packages**: `snake_case`

### Import Conventions

- Use `from __future__ import annotations` for forward compatibility
- Group imports: standard library, external packages, internal modules
- Prefer absolute imports over relative

### Async Patterns

- The client uses `asyncio` with `nest-asyncio` for sync/async interop
- Test suite uses `pytest-asyncio` with `asyncio_mode = "auto"`

### Logging

```python
logging.basicConfig(
    format="%(asctime)s.%(msecs)03d - %(name)s:%(levelname)s: %(message)s",
    level=logging.WARNING
)
logger = logging.getLogger("model-registry")
```

---

## TypeScript/UI Guidelines

- **Frontend**: React with Next.js
- **Backend-for-Frontend (BFF)**: Go with chi router
- **BFF linting**: golangci-lint v2 with its own `.golangci.yaml`
- Generated code uses `lax` exclusion mode for linting

---

## API Design

### Contract-First OpenAPI

- **Spec version**: OpenAPI 3.0.3
- **API version**: `v1alpha3`
- **Base path**: `/api/model_registry/v1alpha3/`
- Source specs live in `api/openapi/src/` and are merged via `scripts/merge_openapi.sh`
- Final spec: `api/openapi/model-registry.yaml`

### REST Patterns

- **Upsert semantics**: Create if no ID provided, update if ID provided
- **Pagination**: `pageSize`, `nextPageToken`, `orderBy`, `sortOrder` (ASC/DESC)
- **Filtering**: `filterQuery` parameter with custom DSL (parsed by Participle)
- **Soft delete**: Entities are archived (status set to `ARCHIVED`), not hard-deleted
- **Error responses**: Standard HTTP status codes (400, 401, 404, 500, 503) with JSON error bodies

### Sentinel Errors

Defined in `pkg/api/`:
- `ErrBadRequest` -- 400
- `ErrNotFound` -- 404
- `ErrConflict` -- 409

---

## Database Patterns

### ORM and Drivers

- **ORM**: GORM
- **Supported databases**: MySQL 8.3, PostgreSQL
- **Migrations**: golang-migrate with versioned SQL files
  - MySQL: `internal/datastore/embedmd/mysql/migrations/`
  - PostgreSQL: `internal/datastore/embedmd/postgres/migrations/`

### Schema Generation

GORM structs are auto-generated from the live database schema:

```bash
make gen/gorm/mysql      # Generate from MySQL
make gen/gorm/postgres   # Generate from PostgreSQL
```

Generated files live in `internal/db/schema/`. CI validates that generated structs match committed code.

### Datastore Abstraction

- `Connector` interface abstracts the database implementation
- `RepoSet` provides a registry of repositories
- Repository implementations live in `internal/db/service/`

### Pagination

- Cursor-based pagination using `nextPageToken`
- GORM scopes in `internal/db/scopes/paginate.go`
- Configurable `pageSize` with defaults

### Filter Queries

- Custom DSL parsed by Participle v2 in `internal/db/filter/parser.go`
- Property mapping translates REST field names to database columns
- Converted to GORM `Where` clauses

---

## Code Generation

Multiple generators are used -- never edit generated files directly:

| Generator | Purpose | Trigger |
|-----------|---------|---------|
| openapi-generator | Server stubs and Go/Python client models | `make gen/openapi`, `make gen/openapi-server` |
| goverter | Type converter functions | `make gen/converter` |
| controller-gen | Kubernetes CRD manifests and DeepCopy methods | `make controller/manifests`, `make controller/generate` |
| gorm-gen | GORM structs from database schema | `make gen/gorm` |

Run all generators: `make gen`

CI enforces that generated code is up-to-date:
- Weekly scheduled `go generate ./...` with auto-PR
- DB schema struct checks on every PR
- OpenAPI spec validation on API changes

---

## Testing

### Go Testing

- **Framework**: Standard `testing` package + `testify` (assert, require)
- **Controller tests**: Ginkgo/Gomega with envtest
- **Integration tests**: TestContainers for Go (PostgreSQL, MySQL)
- **Database mocking**: `go-sqlmock`
- **Test utilities**: `internal/testutils/` (shared container setup)

Run tests:

```bash
make test          # Unit tests (excludes controller)
make test-cover    # With coverage
make controller/test  # Controller tests with envtest
```

Patterns:
- Table-driven tests with `t.Run()` subtests
- Helper functions for database setup (`SetupSharedPostgres()`, `SetupPostgresWithMigrations()`)
- Integration tests require Docker/Podman for TestContainers

### Python Testing

- **Framework**: pytest with plugins (pytest-cov, pytest-mock, pytest-xdist, pytest-asyncio)
- **Property-based testing**: Schemathesis (OpenAPI fuzz testing)
- **Async tests**: `asyncio_mode = "auto"`
- **Test automation**: Nox sessions

Custom markers:
- `@pytest.mark.e2e` -- end-to-end tests
- `@pytest.mark.fuzz` -- fuzzing/property-based tests

Nox sessions: `tests`, `e2e`, `fuzz`, `lint`, `mypy`, `docs-build`, `coverage`

---

## Build and CI/CD

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Full build: generate + vet + lint + compile |
| `make gen` | Run all code generation |
| `make lint` | Run golangci-lint |
| `make vet` | Run go vet |
| `make test` | Run unit tests |
| `make test-cover` | Tests with coverage report |
| `make run/proxy` | Run proxy server from source |
| `make image/build` | Build Docker image |
| `make compose/up` | Start services with Docker Compose (MySQL) |
| `make compose/up/postgres` | Start services with Docker Compose (PostgreSQL) |

### CI Workflows (GitHub Actions)

**Go pipelines:**
- Build + unit tests + coverage
- `go mod tidy` diff check
- OpenAPI spec validation (on API changes)
- DB schema struct validation (MySQL + PostgreSQL)
- Weekly `go generate` sync with auto-PR

**Python pipelines:**
- Ruff linting + mypy type checking
- Auto-generated client sync check
- E2E tests (matrix: Python 3.10-3.12 x K8s versions x DB types)
- Schemathesis fuzz tests (on main merges or API changes)
- Sphinx docs build

**Other:**
- Controller and CSI tests
- Docker image builds (PR and release)
- UI frontend and BFF builds
- Trivy image scanning
- FOSSA license scanning
- OpenSSF Scorecard
- Dependabot (gomod, pip, docker, github-actions, npm -- all weekly)

### Pre-commit Hooks

Configured in `.pre-commit-config.yaml`:
- Standard checks: trailing whitespace, end-of-file-fixer, merge-conflict detection
- `check-added-large-files`, `check-ast`, `check-case-conflict`, `check-json`
- `detect-private-key` (excludes test files)
- Ruff linting with `--fix` for `clients/python/`
- Ruff formatting for generated Python code (`src/mr_openapi/`)

---

## Docker and Deployment

### Container Images

- **Build image**: `registry.access.redhat.com/ubi9/go-toolset:1.25`
- **Runtime image**: `registry.access.redhat.com/ubi9/ubi-minimal:latest`
- **Non-root user**: `65532:65532`
- **Multi-arch**: `linux/arm64`, `linux/amd64`, `linux/s390x`, `linux/ppc64le`
- **Registry**: `ghcr.io/kubeflow/model-registry/server`

### Kubernetes

- Kustomize-based deployment manifests in `manifests/kustomize/`
- Kubernetes controller managed via controller-runtime
- CRD definitions generated by controller-gen
- KServe integration for inference services

### Local Development

- Docker Compose profiles for MySQL and PostgreSQL
- DevContainer support for Apple Silicon/ARM (x86 emulation)
- Kind cluster deployment for local K8s testing

---

## Contribution Workflow

### DCO Requirement

All commits **must** be signed off using the Developer Certificate of Origin:

```bash
git commit -s -m "your commit message"
```

### PR Checklist

- Meaningful commit messages
- Automated tests for major new functionality
- Manual testing verification
- Code follows Kubeflow contribution guidelines
- First-time contributors need `ok-to-test` label from reviewers

### Key Practices

1. **Never edit generated files** -- modify the source (OpenAPI spec, converter interfaces, migrations) and regenerate
2. **Run `make gen`** after modifying OpenAPI specs or converter interfaces
3. **Run `make gen/gorm`** after modifying database migrations
4. **Run `make build`** before submitting -- it runs generation, vet, lint, and compile
5. **Keep generated code committed** -- CI validates that generated code matches
6. **Environment variables** use the `MR_` prefix for configuration
7. **Deletion** is soft delete via `ARCHIVED` status, not hard delete

### OWNERS

The project uses Kubeflow's OWNERS file-based PR workflow. Any repository approver can merge when CI passes and review criteria are met.

---

## Adding New Entities to the Model Catalog

The Model Catalog subsystem (`catalog/`) supports multiple entity types beyond ML models. This section describes the step-by-step pattern for adding a new entity type, based on how MCP (Model Context Protocol) servers were added in PR #2029.

### Overview

Adding a new entity type touches **9 layers** across the codebase. The pattern follows a bottom-up approach: define the API contract, generate code, implement the data layer, then wire everything together.

For this guide, replace `MyEntity` with your actual entity name (e.g., `McpServer`, `Dataset`, `Pipeline`).

### Step 1: Define the OpenAPI Specification

**File**: `api/openapi/src/catalog.yaml`

Define your entity's REST API surface:

1. **Add schemas** for the entity, its list wrapper, and any related types:

```yaml
# Main entity schema
MyEntity:
  description: Description of the entity.
  type: object
  required:
    - id
    - name
  properties:
    id:
      type: string
    name:
      type: string
    source_id:
      type: string
    # ... entity-specific properties
    customProperties:
      type: object
      additionalProperties:
        $ref: "#/components/schemas/MetadataValue"

# List wrapper -- always follows this pattern
MyEntityList:
  description: List of MyEntity entities.
  allOf:
    - type: object
      properties:
        items:
          type: array
          items:
            $ref: "#/components/schemas/MyEntity"
      required:
        - items
    - $ref: "#/components/schemas/BaseResourceList"
```

2. **Add REST paths** under a new service tag:

```yaml
paths:
  /api/model_catalog/v1alpha1/my_entities:
    get:
      tags:
        - MyEntityCatalogService
      parameters:
        - $ref: "#/components/parameters/name"
        - $ref: "#/components/parameters/filterQuery"
        - $ref: "#/components/parameters/pageSize"
        - $ref: "#/components/parameters/orderBy"
        - $ref: "#/components/parameters/sortOrder"
        - $ref: "#/components/parameters/nextPageToken"
      responses:
        "200":
          $ref: "#/components/responses/MyEntityListResponse"
      operationId: findMyEntities

  /api/model_catalog/v1alpha1/my_entities/{entity_id}:
    get:
      tags:
        - MyEntityCatalogService
      operationId: getMyEntity
```

3. **Add responses**:

```yaml
responses:
  MyEntityListResponse:
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/MyEntityList"
  MyEntityResponse:
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/MyEntity"
```

4. **Register the asset type** in the `CatalogAssetType` enum:

```yaml
CatalogAssetType:
  enum:
    - models
    - mcp_servers
    - my_entities    # <-- add here
```

5. **Add `assetType` field** to `CatalogSource` if not already present (it was added in PR #2029).

6. **Run code generation**:

```bash
make gen/openapi          # Regenerate catalog/pkg/openapi/ models
./catalog/scripts/gen_openapi_server.sh   # Regenerate server stubs
./catalog/scripts/gen_type_asserts.sh     # Regenerate type assertions
```

### Step 2: Register the Asset Type

**File**: `catalog/internal/common/asset_types.go`

Add a constant for your new asset type:

```go
const (
    AssetTypeModels     AssetType = "models"
    AssetTypeMcpServers AssetType = "mcp_servers"
    AssetTypeMyEntities AssetType = "my_entities"  // <-- add here
)
```

**File**: `catalog/internal/catalog/asset_types.go`

Re-export the constant:

```go
const (
    AssetTypeModels      = common.AssetTypeModels
    AssetTypeMcpServers  = common.AssetTypeMcpServers
    AssetTypeMyEntities  = common.AssetTypeMyEntities  // <-- add here
)
```

### Step 3: Create Database Models

**File**: `catalog/internal/db/models/my_entity.go`

Follow the standard pattern:

```go
package models

import "github.com/kubeflow/model-registry/internal/db/models"

// MyEntityListOptions defines filtering and pagination for listing.
type MyEntityListOptions struct {
    models.Pagination
    Name      *string
    SourceIDs *[]string
    Query     *string
}

// MyEntityAttributes holds the entity's typed properties.
type MyEntityAttributes struct {
    Name                     *string
    ExternalID               *string
    SourceID                 *string
    Description              *string
    // ... entity-specific fields
    CreateTimeSinceEpoch     *int64
    LastUpdateTimeSinceEpoch *int64
}

// MyEntity is the domain interface.
type MyEntity interface {
    models.Entity[MyEntityAttributes]
}

// MyEntityImpl is the concrete type.
type MyEntityImpl = models.BaseEntity[MyEntityAttributes]

// MyEntityRepository defines persistence operations.
type MyEntityRepository interface {
    GetByID(id int32) (MyEntity, error)
    GetByName(name string) (MyEntity, error)
    List(listOptions MyEntityListOptions) (*models.ListWrapper[MyEntity], error)
    Save(model MyEntity) (MyEntity, error)
    DeleteBySource(sourceID string) error
    DeleteByID(id int32) error
    GetDistinctSourceIDs() ([]string, error)
}
```

Key conventions:
- `Attributes` struct holds typed property values
- `Entity` interface extends `models.Entity[Attributes]`
- Implementation is an alias: `type MyEntityImpl = models.BaseEntity[Attributes]`
- `Repository` interface defines CRUD + list operations

### Step 4: Add Filter Entity Mappings

**File**: `catalog/internal/db/filter/entity_mappings.go`

1. Add a `RestEntityType` constant:

```go
const (
    CatalogModelRestEntityType    RestEntityType = "CatalogModel"
    McpServerRestEntityType       RestEntityType = "McpServer"
    MyEntityRestEntityType        RestEntityType = "MyEntity"  // <-- add here
)
```

2. Map to MLMD entity type in `GetMLMDEntityType()`:

```go
case MyEntityRestEntityType:
    return filter.ContextEntity
```

3. Define filterable properties:

```go
var myEntityProperties = map[string]PropertyDefinition{
    "id":         {Location: EntityTable, ValueType: "int_value", ColumnName: "id"},
    "name":       {Location: EntityTable, ValueType: "string_value", ColumnName: "name"},
    "externalId": {Location: EntityTable, ValueType: "string_value", ColumnName: "external_id"},
    "source_id":  {Location: PropertyTable, ValueType: "string_value"},
    // ... add entity-specific properties
}
```

4. Register in `GetPropertyDefinitionForRestEntity()`:

```go
case MyEntityRestEntityType:
    return getDefinition(myEntityProperties, propertyName)
```

### Step 5: Implement the Repository

**File**: `catalog/internal/db/service/my_entity.go`

Implement the repository using `GenericRepository`:

```go
type MyEntityRepositoryImpl struct {
    *service.GenericRepository[
        models.MyEntity,
        schema.Context,
        schema.ContextProperty,
        *models.MyEntityListOptions,
    ]
}

func NewMyEntityRepository(params datastore.NewRepositoryParams) (models.MyEntityRepository, error) {
    repo, err := service.NewGenericRepository[...](params, service.GenericRepositoryConfig{
        EntityToSchema:       mapMyEntityToContext,
        SchemaToEntity:       mapContextToMyEntity,
        EntityToProperties:   mapMyEntityToProperties,
        ApplyListFilters:     applyMyEntityListFilters,
        PreserveHistoricalTimes: true,  // important for YAML-sourced data
        EntityMappingFuncs:   catalogEntityMappings{},
    })
    return &MyEntityRepositoryImpl{repo}, err
}
```

Write mapping functions between domain entities and database schema (`Context`/`ContextProperty` tables).

### Step 6: Register in the Datastore Spec

**File**: `catalog/internal/db/service/spec.go`

1. Add a type name constant:

```go
const (
    CatalogModelTypeName  = "kf.CatalogModel"
    MyEntityTypeName      = "kf.MyEntity"  // <-- add here
)
```

2. Register in `DatastoreSpec()`:

```go
AddContext(MyEntityTypeName, datastore.NewSpecType(NewMyEntityRepository).
    AddString("source_id").
    AddString("description").
    // ... list all properties stored in ContextProperty table
),
```

3. Add to `Services` struct and `NewServices()`:

```go
type Services struct {
    // ... existing fields
    MyEntityRepository models.MyEntityRepository
}
```

### Step 7: Implement the Catalog Provider

**File**: `catalog/internal/myentity/db_my_entity_catalog.go`

Create a database-backed catalog provider that translates between REST API parameters and repository calls:

```go
type DbMyEntityCatalogProvider struct {
    repo models.MyEntityRepository
}

func NewDbMyEntityCatalogProvider(repo models.MyEntityRepository) *DbMyEntityCatalogProvider {
    return &DbMyEntityCatalogProvider{repo: repo}
}

func (p *DbMyEntityCatalogProvider) FindMyEntities(ctx context.Context, params FindParams) (*openapi.MyEntityList, error) {
    // Convert REST params to list options
    // Call repository
    // Convert results to OpenAPI types
}

func (p *DbMyEntityCatalogProvider) GetMyEntity(ctx context.Context, id string) (*openapi.MyEntity, error) {
    // ...
}
```

### Step 8: Implement the YAML Loader

**File**: `catalog/internal/catalog/my_entity_loader.go`

Create a loader that reads YAML source files and persists entities to the database:

```go
type MyEntityLoader struct {
    services service.Services
    paths    []string
    sources  *SourceCollection
}

func NewMyEntityLoader(services service.Services, paths []string, sources *SourceCollection) *MyEntityLoader {
    return &MyEntityLoader{services: services, paths: paths, sources: sources}
}

func (l *MyEntityLoader) Start(ctx context.Context) error {
    // Parse YAML source configs
    // For each source:
    //   1. Register source in SourceCollection with AssetType
    //   2. Load entities from YAML data files
    //   3. Apply include/exclude filters
    //   4. Merge entities from multiple sources
    //   5. Persist to database via repository
}
```

### Step 9: Implement Source Filtering (Optional)

**File**: `catalog/internal/catalog/my_entity_filter.go`

Support include/exclude glob patterns for entities:

```go
type MyEntityFilter struct {
    included []string  // glob patterns
    excluded []string  // glob patterns
}

func (f *MyEntityFilter) Filter(entities []MyEntityConfig) []MyEntityConfig {
    // Apply glob matching with case-insensitive comparison
    // Excluded patterns take precedence over included
}
```

### Step 10: Implement Source Merging (Optional)

**File**: `catalog/internal/catalog/my_entity_source_merge.go`

Support field-level merging from multiple YAML sources:

```go
func MergeMyEntities(base, override MyEntityConfig) MyEntityConfig {
    // Override non-zero fields from higher-priority source
    // Preserve unset fields from base
    // Append arrays (tags, tools) or replace based on semantics
}
```

### Step 11: Wire the Generated Server Service

**File**: `catalog/internal/server/openapi/api_my_entity_catalog_service_service.go` (generated)

The OpenAPI generator creates a service stub. Implement the generated interface methods by delegating to your catalog provider.

### Step 12: Register in the Main Command

**File**: `catalog/cmd/catalog.go`

1. Add CLI flags for the new source paths:

```go
fs.StringSliceVar(&catalogCfg.MyEntityCatalogPath, "my-entity-catalogs-path",
    catalogCfg.MyEntityCatalogPath, "Path to my entity catalog source configuration")
```

2. Initialize the loader and service:

```go
// Create loader if paths configured
if len(catalogCfg.MyEntityCatalogPath) > 0 {
    myEntityLoader = catalog.NewMyEntityLoader(services, catalogCfg.MyEntityCatalogPath, loader.Sources)
    err = myEntityLoader.Start(context.Background())
}

// Create provider (always uses database)
myEntityProvider := myentity.NewDbMyEntityCatalogProvider(services.MyEntityRepository)

// Create service and controller
myEntitySvc := openapi.NewMyEntityCatalogServiceAPIService(myEntityProvider)
myEntityCtrl := openapi.NewMyEntityCatalogServiceAPIController(myEntitySvc)

// Register in router
return http.ListenAndServe(addr, openapi.NewRouter(ctrl, mcpCtrl, myEntityCtrl))
```

### Step 13: Add BFF Handlers (for UI)

Create three files in the BFF layer:

1. **Handler** (`clients/ui/bff/internal/api/my_entity_handler.go`):

```go
func (app *App) GetAllMyEntitiesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    client := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
    entities, err := app.repositories.ModelCatalogClient.GetAllMyEntities(client, r.URL.Query())
    // ... write JSON response
}
```

2. **Models** (`clients/ui/bff/internal/models/my_entity.go`): Define response types
3. **Repository** (`clients/ui/bff/internal/repositories/my_entities.go`): API client integration
4. **Register routes** in `clients/ui/bff/internal/api/app.go`

### Step 14: Create YAML Test Data

**File**: `catalog/internal/catalog/testdata/dev-my-entity-catalog-sources.yaml`

```yaml
catalogs:
  - id: my_source
    name: "My Entity Source"
    type: my_entities
    enabled: true
    labels:
      - Production
    properties:
      yamlCatalogPath: my-entities.yaml
    includedEntities:
      - "*"
```

### Step 15: Write Tests

Create test files for each major component:

| File | Tests |
|------|-------|
| `catalog/internal/myentity/db_my_entity_catalog_test.go` | Database provider CRUD and queries |
| `catalog/internal/catalog/my_entity_filter_test.go` | Include/exclude glob pattern matching |
| `catalog/internal/catalog/my_entity_source_merge_test.go` | Field-level merging from multiple sources |

Follow the existing test patterns:
- Table-driven tests with `t.Run()` subtests
- TestContainers for database integration tests
- Meaningful test names describing the scenario

### Step 16: Update Kubernetes Manifests

**File**: `manifests/kustomize/options/catalog/overlays/demo/`

Add kustomization entries for your YAML data files as ConfigMaps, and update the catalog deployment to mount them and pass the `--my-entity-catalogs-path` flag.

### Checklist Summary

When adding a new entity type to the catalog, touch these files:

| # | Layer | Files to Create/Modify |
|---|-------|----------------------|
| 1 | OpenAPI spec | `api/openapi/src/catalog.yaml` |
| 2 | Asset type | `catalog/internal/common/asset_types.go`, `catalog/internal/catalog/asset_types.go` |
| 3 | DB models | `catalog/internal/db/models/<entity>.go` (new) |
| 4 | Filter mappings | `catalog/internal/db/filter/entity_mappings.go` |
| 5 | Repository | `catalog/internal/db/service/<entity>.go` (new) |
| 6 | Datastore spec | `catalog/internal/db/service/spec.go` |
| 7 | Catalog provider | `catalog/internal/<entity>/db_<entity>_catalog.go` (new) |
| 8 | YAML loader | `catalog/internal/catalog/<entity>_loader.go` (new) |
| 9 | Filtering | `catalog/internal/catalog/<entity>_filter.go` (new, optional) |
| 10 | Merging | `catalog/internal/catalog/<entity>_source_merge.go` (new, optional) |
| 11 | Main command | `catalog/cmd/catalog.go` |
| 12 | BFF handler | `clients/ui/bff/internal/api/<entity>_handler.go` (new) |
| 13 | BFF models | `clients/ui/bff/internal/models/<entity>.go` (new) |
| 14 | BFF repository | `clients/ui/bff/internal/repositories/<entity>s.go` (new) |
| 15 | BFF routes | `clients/ui/bff/internal/api/app.go` |
| 16 | Tests | Multiple test files (new) |
| 17 | Test data | `catalog/internal/catalog/testdata/` YAML files (new) |
| 18 | K8s manifests | `manifests/kustomize/options/catalog/` |
| 19 | Generated code | Run `make gen/openapi`, `gen_openapi_server.sh`, `gen_type_asserts.sh` |

### Key Design Patterns to Follow

1. **Properties as rows**: Entity attributes are stored as `ContextProperty` rows (name + typed value), not columns. This allows flexible metadata without migrations.
2. **Preserve timestamps**: Set `PreserveHistoricalTimes: true` for YAML-sourced data so original timestamps are retained.
3. **Asset type discrimination**: Each source has an `assetType` field that determines which loader processes it and which API returns its data.
4. **Shared SourceCollection**: All entity types share a `SourceCollection` so the unified `/sources` API returns all sources regardless of type.
5. **Database-backed provider**: Always use database-backed providers for the REST API. YAML loaders persist data to the database; the API layer reads from the database.
6. **CustomProperties**: Follow the Model Registry convention -- tags are `MetadataStringValue` with empty `string_value`, booleans are `MetadataBoolValue`.
