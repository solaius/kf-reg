# Security

## Overview

The catalog-server implements a layered security model covering authentication, authorization, credential management, and data protection. The layers work together to enforce least-privilege access: unauthenticated or low-privilege requests can browse catalog entities but cannot mutate source configuration or trigger operational actions.

**Location:** `pkg/catalog/plugin/`

```
                         Incoming HTTP Request
                                 |
                                 v
                   +----------------------------+
                   |  Token / Header Extraction |
                   |  (RoleExtractor function)  |
                   +----------------------------+
                                 |
                         role = viewer | operator
                                 |
                                 v
                   +----------------------------+
                   |    RequireRole Middleware   |
                   |  (per-route enforcement)   |
                   +----------------------------+
                          |              |
                     pass (200)     reject (403)
                          |
                          v
                   +----------------------------+
                   |   Handler Execution        |
                   |                            |
                   |  SecretRef Resolution      |
                   |  (on mutations only)       |
                   |                            |
                   |  Sensitive Value Redaction  |
                   |  (on reads)                |
                   +----------------------------+
                                 |
                                 v
                          JSON Response
```

## RBAC Middleware

### Role Types

Two roles form a simple hierarchy where operator includes all viewer permissions:

```go
// pkg/catalog/plugin/rbac.go
type Role string

const (
    RoleViewer   Role = "viewer"   // Read-only access
    RoleOperator Role = "operator" // Read + management mutations
)
```

| Role | Browse Entities | View Sources | View Diagnostics | Manage Sources | Trigger Refresh | Execute Actions |
|------|:-:|:-:|:-:|:-:|:-:|:-:|
| `viewer` | Yes | Yes | Yes | No | No | No |
| `operator` | Yes | Yes | Yes | Yes | Yes | Yes |

### RoleExtractor Function Type

Role determination is abstracted behind a function type that can be swapped at server startup:

```go
// pkg/catalog/plugin/rbac.go
type RoleExtractor func(r *http.Request) Role
```

The server holds a single `RoleExtractor` instance (set via `WithRoleExtractor` server option) and passes it to every `RequireRole` middleware wrapper when mounting management routes.

### RequireRole Middleware

`RequireRole` is a standard `func(http.Handler) http.Handler` middleware factory. It wraps a handler and checks the caller's role before allowing execution:

```go
// pkg/catalog/plugin/rbac.go
func RequireRole(role Role, extractor RoleExtractor) func(http.Handler) http.Handler
```

- Calls the `RoleExtractor` to determine the caller's role
- Compares using a `hasRole` function that respects the hierarchy (operator satisfies viewer)
- Returns HTTP 403 with `{"error":"forbidden","message":"insufficient permissions"}` on failure
- Falls back to `DefaultRoleExtractor` if the extractor is nil

### Management Route Protection

The management router applies `RequireRole` per-endpoint. Read-only endpoints are open to all roles; mutation endpoints require operator:

| Endpoint | Method | Required Role |
|----------|--------|:---:|
| `.../management/sources` | GET | viewer |
| `.../management/diagnostics` | GET | viewer |
| `.../management/sources/{id}/revisions` | GET | viewer |
| `.../management/actions/source` | GET | viewer |
| `.../management/actions/asset` | GET | viewer |
| `.../management/validate-source` | POST | operator |
| `.../management/apply-source` | POST | operator |
| `.../management/sources/{id}/enable` | POST | operator |
| `.../management/sources/{id}` | DELETE | operator |
| `.../management/sources/{id}:validate` | POST | operator |
| `.../management/sources/{id}:rollback` | POST | operator |
| `.../management/refresh` | POST | operator |
| `.../management/refresh/{id}` | POST | operator |
| `.../management/sources/{id}:action` | POST | operator |
| `.../management/entities/{name}:action` | POST | operator |

### DefaultRoleExtractor (Development Only)

The built-in default reads the `X-User-Role` header. It is intended for local development and testing only:

```go
// pkg/catalog/plugin/rbac.go
const RoleHeader = "X-User-Role"

func DefaultRoleExtractor(r *http.Request) Role {
    header := strings.TrimSpace(strings.ToLower(r.Header.Get(RoleHeader)))
    switch header {
    case string(RoleOperator):
        return RoleOperator
    default:
        return RoleViewer
    }
}
```

Missing or unrecognized header values default to `viewer` (deny-by-default for operator access).

## JWT Authentication

### Enabling JWT Auth

Set `CATALOG_AUTH_MODE=jwt` to replace the default header-based extractor with a JWT-based one. When enabled, the server reads roles from cryptographically verified Bearer tokens.

```
cmd/catalog-server/main.go

    CATALOG_AUTH_MODE
         |
    +----+----+----------+
    |         |          |
  "jwt"    "header"    "" (empty)
    |         |          |
    v         +----+-----+
  JWT-based        |
  extractor   DefaultRoleExtractor
    |         (X-User-Role header)
    v
  WithJWTRoleExtractor(cfg)
```

### Configuration

```go
// pkg/catalog/plugin/jwt_role_extractor.go
type JWTRoleExtractorConfig struct {
    RoleClaim         string   // JWT claim containing role (default: "role")
    OperatorRoleValue string   // Claim value mapping to operator (default: "operator")
    PublicKeyPath     string   // Path to RSA public key (PEM)
    Issuer            string   // Expected issuer (iss)
    Audience          string   // Expected audience (aud)
}
```

| Environment Variable | Config Field | Default | Description |
|----------------------|-------------|---------|-------------|
| `CATALOG_JWT_PUBLIC_KEY_PATH` | `PublicKeyPath` | (empty) | Path to PEM-encoded RSA public key for RS256 verification |
| `CATALOG_JWT_ISSUER` | `Issuer` | (empty) | Expected token issuer; skipped if empty |
| `CATALOG_JWT_AUDIENCE` | `Audience` | (empty) | Expected token audience; skipped if empty |
| `CATALOG_JWT_ROLE_CLAIM` | `RoleClaim` | `"role"` | JWT claim path (supports dot-notation, e.g. `realm_access.roles`) |
| `CATALOG_JWT_OPERATOR_VALUE` | `OperatorRoleValue` | `"operator"` | Claim value that maps to operator role |

### Token Verification

The JWT extractor uses the `golang-jwt/jwt/v5` library with the following behavior:

1. **Token extraction** -- reads `Authorization: Bearer <token>` header
2. **Signature verification** -- if `PublicKeyPath` is set, verifies RS256 signature against the loaded RSA public key; if not set, parses without verification (trusted proxy mode)
3. **Standard claims** -- validates `iss` and `aud` if configured
4. **Role extraction** -- navigates the claim path (dot-notation for nested claims) and checks the value:
   - String claim: case-insensitive comparison with `OperatorRoleValue`
   - Array claim: checks if `OperatorRoleValue` is present in the array (Keycloak `realm_access.roles` pattern)
5. **Fallback** -- missing token, parse failure, or unmatched claim all default to `RoleViewer`

### Server Option

```go
// pkg/catalog/plugin/jwt_role_extractor.go
func WithJWTRoleExtractor(cfg JWTRoleExtractorConfig) ServerOption
```

This convenience option creates the JWT extractor and sets it on the server. On construction error (e.g. unreadable key file), the error is logged and the server continues with its existing extractor.

## SecretRef Resolution

### SecretRef Type

Source configurations may reference Kubernetes Secrets instead of inlining sensitive values:

```go
// pkg/catalog/plugin/management_types.go
type SecretRef struct {
    Name      string `json:"name" yaml:"name"`
    Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
    Key       string `json:"key" yaml:"key"`
}
```

A source config property using SecretRef looks like:

```yaml
sources:
  - id: my-source
    properties:
      token:
        name: my-secret      # Kubernetes Secret name
        namespace: catalog    # Optional; defaults to server namespace
        key: api-token        # Key within the Secret's data map
```

### K8sSecretResolver

```go
// pkg/catalog/plugin/secret_resolver.go
type SecretResolver interface {
    Resolve(ctx context.Context, ref SecretRef) (string, error)
}

type K8sSecretResolver struct { ... }
func NewK8sSecretResolver(client kubernetes.Interface, defaultNamespace string) *K8sSecretResolver
```

The resolver reads the referenced key from the Kubernetes Secret. If `ref.Namespace` is empty, the server's own namespace (`CATALOG_CONFIG_NAMESPACE`) is used as the default. The resolver is wired automatically when `CATALOG_CONFIG_STORE_MODE=k8s`.

### Resolution Flow

```
apply-source request
        |
        v
  Validation pipeline
        |
        v
  ResolveSecretRefs()          For each property value:
        |                        - IsSecretRef(v) checks if value is a map
        |                          with string "name" and "key" fields
        |                        - If yes, calls resolver.Resolve(ctx, ref)
        |                        - Returns a shallow copy with resolved values
        v
  Plugin.ApplySource()         Receives resolved (plain string) properties
        |
        v
  ConfigStore.Save()           Persists ORIGINAL input (SecretRefs intact)
```

The original SecretRef objects are persisted to the ConfigStore, not the resolved values. This means secrets are never written to the config file or ConfigMap -- they are resolved at runtime on each apply or refresh.

### IsSecretRef Detection

`IsSecretRef(v any)` identifies SecretRef-shaped property values by checking for a `map[string]any` with non-empty string `"name"` and `"key"` fields.

## Redaction

### Sensitive Key Detection

The system automatically detects sensitive properties by matching key names (case-insensitive) against a set of known patterns:

```go
// pkg/catalog/plugin/redact.go
var sensitiveKeyPatterns = []string{
    "password", "token", "secret", "apikey", "api_key", "credential",
}

func IsSensitiveKey(key string) bool
```

Any property key containing one of these substrings is treated as sensitive.

### Response Redaction

API responses from the `GET .../management/sources` endpoint redact sensitive property values before returning them:

```go
// pkg/catalog/plugin/redact.go
const RedactedValue = "***REDACTED***"

func RedactSensitiveProperties(props map[string]any) map[string]any
```

- Returns a shallow copy of the properties map
- Replaces plain string values for sensitive keys with `"***REDACTED***"`
- Preserves `map[string]any` values (SecretRef objects) unchanged, since they already hide the actual secret value behind a reference
- Prevents credential leakage through list and get endpoints

Example response after redaction:

```json
{
  "id": "hf-models",
  "properties": {
    "baseUrl": "https://huggingface.co",
    "token": "***REDACTED***",
    "org": "kubeflow"
  }
}
```

## FilterQuery Injection Safety

List endpoints accept a `filterQuery` parameter with SQL-like syntax (e.g. `name='example' AND toolCount>5`). Two separate mechanisms prevent injection:

### DB-Backed Plugins (MCP, Knowledge Sources)

DB-backed plugins use a formal grammar parser (`internal/db/filter/parser.go`) built with `participle`. The parser:

1. **Lexes** the input with a strict token grammar that only recognizes identifiers, numbers, quoted strings, and known operators (`=`, `!=`, `>`, `<`, `>=`, `<=`, `LIKE`, `ILIKE`, `IN`, `AND`, `OR`)
2. **Parses** into a typed AST (`WhereClause` -> `Expression` -> `OrExpression` -> `AndExpression` -> `Term`)
3. **Builds** GORM queries using parameterized placeholders (`?`) with bound arguments -- raw user input never appears in SQL strings
4. **Validates** property names against a registered allowlist (`RestEntityPropertyMap`) that enumerates permitted fields per entity type

Unrecognized tokens, malformed expressions, or unknown property names cause a parse error, rejecting the query before it reaches the database.

### In-Memory Plugins (Agents, Prompts, etc.)

In-memory plugins implement a `parseFilterConditions` function that splits the query on `AND`, then matches each part against a fixed set of operators (`=`, `!=`, `>=`, `<=`, `>`, `<`, `LIKE`). Values are string-compared against in-memory struct fields. No SQL is generated, so injection is not applicable.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/rbac.go` | `Role` type, `RoleExtractor`, `RequireRole` middleware, `DefaultRoleExtractor` |
| `pkg/catalog/plugin/jwt_role_extractor.go` | `JWTRoleExtractorConfig`, `NewJWTRoleExtractor`, `WithJWTRoleExtractor`, RS256 verification |
| `pkg/catalog/plugin/secret_resolver.go` | `SecretResolver` interface, `K8sSecretResolver`, `ResolveSecretRefs`, `IsSecretRef` |
| `pkg/catalog/plugin/redact.go` | `IsSensitiveKey`, `RedactSensitiveProperties`, `RedactedValue` constant |
| `pkg/catalog/plugin/management_handlers.go` | Management route wiring with per-endpoint RBAC, SecretRef resolution, redaction calls |
| `internal/db/filter/parser.go` | `participle`-based filter grammar lexer and parser |
| `internal/db/filter/query_builder.go` | Parameterized GORM query builder from parsed filter AST |
| `internal/db/filter/rest_entity_mapping.go` | `RestEntityPropertyMap` allowlist for filter field validation |

## Multi-Tenant Authorization (Phase 8)

Phase 8 introduces enterprise-grade authorization that delegates access control to Kubernetes RBAC via SubjectAccessReview (SAR). This replaces the simple viewer/operator role model for production deployments while preserving backward compatibility.

**Location:** `pkg/authz/`

### Authorization Modes

| Mode | Env Var | Behavior |
|------|---------|----------|
| `none` | `CATALOG_AUTHZ_MODE=none` | All requests allowed (default, backward compatible) |
| `sar` | `CATALOG_AUTHZ_MODE=sar` | SubjectAccessReview against K8s API server |

### Identity Extraction

Identity is extracted from standard auth-proxy headers via `IdentityMiddleware`:

| Header | Purpose | Default |
|--------|---------|---------|
| `X-Remote-User` | Authenticated username | `"anonymous"` |
| `X-Remote-Group` | Comma-separated group list | (empty) |

The service expects an authenticating reverse proxy (Istio, OAuth2-Proxy, Dex, etc.) to set these headers before the request reaches the catalog-server.

```go
// pkg/authz/identity.go
type Identity struct {
    User   string
    Groups []string
}
```

### Resource Mapping

Every HTTP request is mapped to a `(resource, verb)` tuple for SAR checks. The API group is `catalog.kubeflow.org`.

| Resource | Available Verbs | Example Endpoints |
|----------|----------------|-------------------|
| `plugins` | list | `GET /api/plugins` |
| `capabilities` | get | `GET /api/plugins/{name}/capabilities` |
| `catalogsources` | get, list, create, update, delete | Management source endpoints |
| `assets` | get, list, create, update, delete | Entity list/get endpoints |
| `actions` | list, execute | `:action` endpoints |
| `jobs` | get, list, create | Job status and cancel endpoints |
| `approvals` | list, approve, get | Governance approval endpoints |
| `audit` | list, get | Audit event endpoints |

The URL-to-resource mapping is implemented in `pkg/authz/mapper.go` and matches patterns from most specific to least specific (e.g., `:action` suffix before generic management routes).

### SAR Authorization Flow

```
Incoming Request
      |
      v
Identity Middleware              Extract X-Remote-User, X-Remote-Group
      |
      v
Tenancy Middleware               Resolve namespace from ?namespace= or X-Namespace
      |
      v
Authz Middleware (if sar)        Map (method, path) -> (resource, verb)
      |                          Build AuthzRequest{User, Groups, Resource, Verb, Namespace}
      +---> CachedAuthorizer
              |
              +---> Cache HIT? Return cached decision
              |
              +---> Cache MISS? Call SARAuthorizer
                       |
                       v
                    K8s API Server (SubjectAccessReview)
                       |
                    allowed=true  --> proceed to handler
                    allowed=false --> 403 Forbidden
```

### SAR Caching

SAR results are cached in-memory with a configurable TTL (default 10 seconds) to reduce load on the Kubernetes API server:

```go
// pkg/authz/cache.go
type CachedAuthorizer struct {
    inner Authorizer       // Wrapped SARAuthorizer
    ttl   time.Duration    // Default: 10s
    cache map[string]cacheEntry
}
```

Cache key: `user:groups:resource:verb:namespace`

### RBAC Manifest Examples

**AI Engineer role** (namespace-scoped, can manage sources and assets within their namespace):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: catalog-ai-engineer
  namespace: team-a
rules:
- apiGroups: ["catalog.kubeflow.org"]
  resources: ["catalogsources", "assets", "actions", "jobs"]
  verbs: ["get", "list", "create", "update"]
- apiGroups: ["catalog.kubeflow.org"]
  resources: ["plugins", "capabilities"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alice-ai-engineer
  namespace: team-a
subjects:
- kind: User
  name: alice
roleRef:
  kind: Role
  name: catalog-ai-engineer
  apiGroup: rbac.authorization.k8s.io
```

**Platform Operator role** (cluster-scoped, can manage all namespaces and view audit):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: catalog-platform-operator
rules:
- apiGroups: ["catalog.kubeflow.org"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ops-catalog-admin
subjects:
- kind: Group
  name: platform-ops
roleRef:
  kind: ClusterRole
  name: catalog-platform-operator
  apiGroup: rbac.authorization.k8s.io
```

**Read-only Viewer role** (namespace-scoped, browse only):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: catalog-viewer
  namespace: team-b
rules:
- apiGroups: ["catalog.kubeflow.org"]
  resources: ["plugins", "capabilities", "catalogsources", "assets"]
  verbs: ["get", "list"]
```

### Denied Response Format

All authorization denials return a consistent JSON error:

```json
{
  "error": "forbidden",
  "message": "insufficient permissions for catalogsources/create in namespace team-a"
}
```

HTTP status: 403 Forbidden.

## Audit Logging (Phase 8)

The audit system captures a structured event for every state-changing management operation. Events record who performed what action, when, in which namespace, and the outcome.

**Location:** `pkg/audit/`

### Audit Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_AUDIT_ENABLED` | `true` | Whether audit middleware is active |
| `CATALOG_AUDIT_RETENTION_DAYS` | `90` | Days to keep audit events before cleanup |
| `CATALOG_AUDIT_LOG_DENIED` | `true` | Whether to record 403 denied actions |

### Audit Event Schema

Each audit event captures:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | UUID of the event |
| `namespace` | string | Tenant namespace |
| `correlationId` | string | From `X-Correlation-ID` header or request ID |
| `eventType` | string | Category (e.g., `"management"`) |
| `actor` | string | Authenticated user |
| `requestId` | string | Chi middleware request ID |
| `plugin` | string | Plugin name (e.g., `"mcp"`) |
| `resourceType` | string | Resource type (e.g., `"sources"`, `"entities"`) |
| `resourceIds` | []string | Affected resource identifiers |
| `action` | string | Action verb (e.g., `"apply-source"`, `"refresh"`) |
| `outcome` | string | `"success"`, `"denied"`, or `"failure"` |
| `statusCode` | int | HTTP response status code |
| `metadata` | object | Method, path, duration, groups |
| `createdAt` | timestamp | Event creation time |

### Audit API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/audit/v1alpha1/events` | List events (paginated, filtered) |
| GET | `/api/audit/v1alpha1/events/{eventId}` | Get event by ID |

**Filter parameters:** `namespace`, `actor`, `plugin`, `action`, `eventType`, `pageSize`, `pageToken`

**Example response:**

```json
{
  "events": [
    {
      "id": "a1b2c3d4-...",
      "namespace": "team-a",
      "eventType": "management",
      "actor": "alice",
      "plugin": "mcp",
      "resourceType": "sources",
      "action": "apply-source",
      "outcome": "success",
      "statusCode": 200,
      "metadata": {
        "method": "POST",
        "path": "/api/mcp_catalog/v1alpha1/management/apply-source",
        "duration": "45ms"
      },
      "createdAt": "2026-02-17T10:30:00Z"
    }
  ],
  "nextPageToken": "",
  "totalSize": 1
}
```

### Audit Retention

The `RetentionWorker` runs daily and deletes audit events older than the configured retention period. It is designed to run as a leader-only singleton loop in HA deployments.

## Phase 8 Key Files

| File | Purpose |
|------|---------|
| `pkg/tenancy/` | Tenant context, middleware, resolvers |
| `pkg/authz/types.go` | `Authorizer` interface, resource/verb constants |
| `pkg/authz/sar.go` | SubjectAccessReview authorizer |
| `pkg/authz/identity.go` | Identity extraction middleware (X-Remote-User/Group) |
| `pkg/authz/cache.go` | Short-lived SAR result cache |
| `pkg/authz/mapper.go` | HTTP request to (resource, verb) mapping |
| `pkg/authz/middleware.go` | `RequirePermission` and `AuthzMiddleware` |
| `pkg/authz/noop.go` | No-op authorizer for development mode |
| `pkg/audit/middleware.go` | Audit event capture middleware |
| `pkg/audit/handlers.go` | Audit API list/get handlers |
| `pkg/audit/retention.go` | Daily audit event cleanup worker |
| `pkg/audit/router.go` | Audit API router with authz integration |
| `pkg/jobs/models.go` | RefreshJob GORM model and state machine |
| `pkg/jobs/store.go` | Job store with Enqueue, Claim, Complete, Fail, Cancel |
| `pkg/jobs/worker.go` | Worker pool with configurable concurrency |
| `pkg/jobs/handlers.go` | Job API list/get/cancel handlers |
| `pkg/jobs/router.go` | Job API router with authz integration |
| `pkg/cache/lru.go` | Thread-safe LRU cache with TTL |
| `pkg/cache/middleware.go` | HTTP response caching middleware |
| `pkg/cache/invalidation.go` | Cache manager with per-plugin invalidation |
| `pkg/ha/migration_lock.go` | Migration locking (PG advisory + table fallback) |
| `pkg/ha/leader_election.go` | K8s Lease-based leader election |

---

[Back to Operations](./README.md) | [Prev: Deployment](./deployment.md)
