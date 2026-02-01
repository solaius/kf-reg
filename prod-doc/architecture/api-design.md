# API Design

This document describes the REST API design patterns, OpenAPI approach, and conventions used in the Kubeflow Model Registry.

## Contract-First OpenAPI Approach

The Model Registry follows a **contract-first** API design where the OpenAPI specification is the source of truth.

### Workflow

```
1. Define/Update OpenAPI Spec
        │
        ▼
2. Validate Specification
   make openapi/validate
        │
        ▼
3. Generate Server Code
   make gen/openapi-server
        │
        ▼
4. Generate Client Code
   make gen/openapi
        │
        ▼
5. Implement Business Logic
   (Service layer)
        │
        ▼
6. Test & Deploy
```

### OpenAPI Specifications

| Specification | Location | Purpose |
|---------------|----------|---------|
| Model Registry | `api/openapi/model-registry.yaml` | Core API |
| Catalog Service | `api/openapi/catalog.yaml` | Catalog API |
| UI BFF | `clients/ui/api/openapi/mod-arch.yaml` | BFF API |

**OpenAPI Version:** 3.0.3

---

## API Versioning

### URL-Based Versioning

```
/api/model_registry/v1alpha3/...  # Model Registry
/api/model_catalog/v1alpha1/...   # Catalog Service
/api/v1/...                        # UI BFF
```

### Version Lifecycle

| Stage | Stability | Changes Allowed |
|-------|-----------|-----------------|
| `v1alpha1` | Experimental | Breaking changes |
| `v1beta1` | Pre-release | Minor breaking changes |
| `v1` | Stable | Backward compatible only |

---

## REST API Patterns

### Resource Naming

| Pattern | Example |
|---------|---------|
| Collection | `/registered_models` |
| Resource | `/registered_models/{registeredModelId}` |
| Sub-collection | `/registered_models/{id}/versions` |
| Action | `/registered_models/{id}:archive` |

### HTTP Methods

| Method | Purpose | Idempotent | Safe |
|--------|---------|------------|------|
| `GET` | Retrieve resource(s) | Yes | Yes |
| `POST` | Create resource | No | No |
| `PATCH` | Partial update | Yes | No |
| `PUT` | Full replacement | Yes | No |
| `DELETE` | Remove resource | Yes | No |

### Standard CRUD Operations

```yaml
# List resources
GET /registered_models
Response: RegisteredModelList

# Create resource
POST /registered_models
Request: RegisteredModelCreate
Response: RegisteredModel (201)

# Get single resource
GET /registered_models/{registeredModelId}
Response: RegisteredModel

# Update resource
PATCH /registered_models/{registeredModelId}
Request: RegisteredModelUpdate
Response: RegisteredModel

# Delete/Archive resource
PATCH /registered_models/{registeredModelId}
Request: { "state": "ARCHIVED" }
Response: RegisteredModel
```

---

## Pagination

### Cursor-Based Pagination

The API uses cursor-based pagination for efficient traversal of large datasets.

**Request Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `pageSize` | integer | Number of items per page (default: 20, max: 100) |
| `nextPageToken` | string | Cursor for next page |
| `orderBy` | string | Sort order (e.g., `NAME`, `CREATE_TIME`) |
| `sortOrder` | string | `ASC` or `DESC` |

**Response Structure:**

```json
{
  "items": [...],
  "nextPageToken": "eyJpZCI6MTAwLCJ2YWx1ZSI6Im1vZGVsLXgiLCJvcmRlciI6Ik5BTUUifQ==",
  "pageSize": 20,
  "size": 20
}
```

**Token Format:**
- Base64-encoded JSON: `{id}:{value}`
- Maximum size: 1024 bytes
- Contains last item's sort value for efficient cursor queries

### Example

```bash
# First page
GET /api/model_registry/v1alpha3/registered_models?pageSize=10

# Next page
GET /api/model_registry/v1alpha3/registered_models?pageSize=10&nextPageToken=abc123
```

---

## Filtering

### Filter Query Syntax

The API supports advanced filtering via the `filterQuery` parameter.

**Operators:**

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equals | `name="my-model"` |
| `!=` | Not equals | `state!="ARCHIVED"` |
| `IN` | In list | `state IN ("LIVE", "PENDING")` |
| `CONTAINS` | Contains | `tasks CONTAINS "classification"` |

**Logical Operators:**

| Operator | Description |
|----------|-------------|
| `AND` | Both conditions must match |
| `OR` | Either condition must match |
| `()` | Grouping |

**Examples:**

```bash
# Simple filter
GET /models?filterQuery=name="my-model"

# Combined filters
GET /models?filterQuery=state="LIVE" AND owner="alice"

# Custom property filter
GET /models?filterQuery=customProperties.model_type.string_value="generative"
```

### Free-Form Search

The `q` parameter enables keyword search across multiple fields:

```bash
GET /models?q=llama
```

Searches: `name`, `description`, `provider`

---

## Error Handling

### Error Response Format

```json
{
  "code": "NOT_FOUND",
  "message": "RegisteredModel with id 'abc123' not found",
  "details": {
    "resourceType": "RegisteredModel",
    "resourceId": "abc123"
  }
}
```

### HTTP Status Codes

| Code | Meaning | When Used |
|------|---------|-----------|
| 200 | OK | Successful GET, PATCH, PUT |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Invalid input, validation error |
| 401 | Unauthorized | Missing authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Duplicate resource, version conflict |
| 500 | Internal Error | Server-side error |

### Domain Error Mapping

```go
var (
    ErrBadRequest = errors.New("bad request")   // → 400
    ErrNotFound   = errors.New("not found")     // → 404
    ErrConflict   = errors.New("conflict")      // → 409
)
```

---

## Request/Response Patterns

### Create Pattern

**Request:** `{Entity}Create` (excludes auto-generated fields)

```yaml
RegisteredModelCreate:
  type: object
  properties:
    name:
      type: string
    description:
      type: string
    customProperties:
      type: object
  required:
    - name
```

**Response:** `{Entity}` (includes all fields)

```yaml
RegisteredModel:
  type: object
  properties:
    id:
      type: string
    name:
      type: string
    description:
      type: string
    customProperties:
      type: object
    createTimeSinceEpoch:
      type: integer
    lastUpdateTimeSinceEpoch:
      type: integer
```

### Update Pattern

**Request:** `{Entity}Update` (partial update, all fields optional)

```yaml
RegisteredModelUpdate:
  type: object
  properties:
    description:
      type: string
    state:
      $ref: '#/components/schemas/ModelState'
    customProperties:
      type: object
```

### List Pattern

**Response:** `{Entity}List`

```yaml
RegisteredModelList:
  type: object
  properties:
    items:
      type: array
      items:
        $ref: '#/components/schemas/RegisteredModel'
    nextPageToken:
      type: string
    pageSize:
      type: integer
    size:
      type: integer
```

---

## Authentication & Authorization

### Kubernetes Mode

In Kubeflow deployments, authentication uses Kubernetes service accounts:

```yaml
Authorization: Bearer <service-account-token>
```

### RBAC Integration

The BFF layer integrates with Kubernetes RBAC:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: model-registry-viewer
rules:
- apiGroups: [""]
  resources: ["modelregistries"]
  verbs: ["get", "list", "watch"]
```

### Standalone Mode

In standalone mode, authentication can be disabled or use custom middleware.

---

## API Endpoints Reference

### Model Registry API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/registered_models` | List registered models |
| `POST` | `/registered_models` | Create registered model |
| `GET` | `/registered_models/{id}` | Get registered model |
| `PATCH` | `/registered_models/{id}` | Update registered model |
| `GET` | `/registered_models/{id}/versions` | List model versions |
| `POST` | `/registered_models/{id}/versions` | Create model version |
| `GET` | `/model_versions/{id}` | Get model version |
| `PATCH` | `/model_versions/{id}` | Update model version |
| `GET` | `/model_versions/{id}/artifacts` | List artifacts |
| `POST` | `/model_artifacts` | Create model artifact |
| `GET` | `/inference_services` | List inference services |
| `POST` | `/inference_services` | Create inference service |
| `GET` | `/serving_environments` | List serving environments |
| `POST` | `/serving_environments` | Create serving environment |

### Catalog API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/sources` | List catalog sources |
| `GET` | `/models` | Search models |
| `GET` | `/models/filter_options` | Get filter options |
| `GET` | `/sources/{id}/models/{name}` | Get specific model |
| `GET` | `/sources/{id}/models/{name}/artifacts` | Get model artifacts |
| `POST` | `/sources/preview` | Preview source config |
| `GET` | `/labels` | List source labels |

---

## OpenAPI Code Generation

### Server Generation

```bash
make gen/openapi-server
```

Generates:
- `internal/server/openapi/api_*.go` - Route handlers
- `internal/server/openapi/model_*.go` - Request/Response types
- `internal/server/openapi/routers.go` - Router configuration

### Client Generation

```bash
make gen/openapi
```

Generates:
- `pkg/openapi/api_*.go` - Client methods
- `pkg/openapi/model_*.go` - Model types
- `pkg/openapi/client.go` - Client configuration

---

[Back to Architecture Index](./README.md) | [Previous: Data Models](./data-models.md) | [Next: Deployment Modes](./deployment-modes.md)
