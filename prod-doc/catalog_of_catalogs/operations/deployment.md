# Deployment and Health

## Overview

The catalog-server is packaged as a multi-stage Docker image, orchestrated locally via Docker Compose, and deployed to Kubernetes using standard manifests with health probes. This document covers the container build, the Compose development stack, Kubernetes deployment, and the health endpoint contract.

```
                    Build and Deploy Flow
                    =====================

 Dockerfile.catalog-server           docker-compose.catalog.yaml
 +---------------------------+       +-----------------------------+
 | Stage 1: golang:1.24      |       | postgres:16                 |
 |   go build catalog-server |       |   pg_isready healthcheck    |
 |   go build catalog-cli    |       +----------+------------------+
 |   go build healthcheck    |                  |
 +----------+----------------+       depends_on (service_healthy)
            |                                   |
 +----------v----------------+       +----------v------------------+
 | Stage 2: alpine:3.21      |       | catalog-server              |
 |   git, ca-certificates    |       |   --listen=:8080            |
 |   /catalog-server         |       |   --sources=/config/...     |
 |   /healthcheck            |       |   --db-type=postgres        |
 +---------------------------+       +-----------------------------+
```

**Location:** `Dockerfile.catalog-server`, `docker-compose.catalog.yaml`, `deploy/catalog-server/`

## Dockerfile.catalog-server

The catalog-server uses a two-stage Docker build.

### Stage 1: Go Builder

The builder stage compiles three binaries from a `golang:1.24` base with cross-platform support via `BUILDPLATFORM`, `TARGETOS`, and `TARGETARCH` build args.

```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

WORKDIR /workspace

# Dependency caching
COPY go.mod go.sum ./
COPY pkg/openapi/go.mod pkg/openapi/
COPY catalog/pkg/openapi/go.mod catalog/pkg/openapi/
RUN go mod download

# Copy source
COPY main.go ./
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/
COPY catalog/ catalog/
COPY templates/ templates/
COPY patches/ patches/
COPY scripts/ scripts/
COPY Makefile .

# Build three binaries with CGO disabled for static linking
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -o catalog-server ./cmd/catalog-server

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -o catalog-cli ./cmd/catalog

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -o healthcheck ./cmd/healthcheck
```

| Binary | Source | Purpose |
|--------|--------|---------|
| `catalog-server` | `cmd/catalog-server` | Main HTTP server process |
| `catalog-cli` | `cmd/catalog` | CLI for catalog operations |
| `healthcheck` | `cmd/healthcheck` | Minimal HTTP health check binary for container probes |

### Stage 2: Alpine Runtime

The runtime was changed from `gcr.io/distroless` to `alpine:3.21` in Phase 6 to provide the `git` binary required by the Git source provider.

```dockerfile
FROM alpine:3.21
RUN apk add --no-cache git ca-certificates && \
    adduser -D -u 65532 nonroot && \
    mkdir -p /home/nonroot && \
    git config --system --add safe.directory '*'
WORKDIR /

COPY --from=builder /workspace/catalog-server .
COPY --from=builder /workspace/catalog-cli .
COPY --from=builder /workspace/healthcheck /usr/local/bin/healthcheck

COPY catalog/config/ /config/
COPY catalog/plugins/mcp/data/ /plugins/mcp/data/
COPY catalog/plugins/knowledge/data/ /plugins/knowledge/data/

USER nonroot
EXPOSE 8080
ENTRYPOINT ["/catalog-server"]
```

Key runtime details:

| Detail | Value |
|--------|-------|
| Base image | `alpine:3.21` |
| Extra packages | `git`, `ca-certificates` |
| Non-root user UID | 65532 |
| Healthcheck binary | `/usr/local/bin/healthcheck` |
| Config directory | `/config/` (sources.yaml and related files) |
| Default entrypoint | `/catalog-server` |
| Exposed port | 8080 |

The `healthcheck` binary is a minimal Go program that performs a GET request to a given URL and exits 0 on 2xx, exit 1 otherwise. It is used by both Docker and Kubernetes health probes.

## docker-compose.catalog.yaml

The Compose file defines two services: a PostgreSQL 16 database and the catalog-server.

```yaml
services:
  postgres:
    image: postgres:16
    container_name: catalog-postgres
    environment:
      POSTGRES_DB: catalog
      POSTGRES_USER: catalog
      POSTGRES_PASSWORD: catalog
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U catalog -d catalog"]
      interval: 5s
      timeout: 5s
      retries: 5

  catalog-server:
    build:
      context: .
      dockerfile: Dockerfile.catalog-server
    container_name: catalog-server
    command:
      - --listen=:8080
      - --sources=/config/sources.yaml
      - --db-type=postgres
      - --db-dsn=postgres://catalog:catalog@postgres:5432/catalog?sslmode=disable
    volumes:
      - ./catalog/config:/config:ro
      - ./catalog/plugins/model/data:/plugins/model/data:ro
      - ./catalog/plugins/mcp/data:/plugins/mcp/data:rw
      - ./catalog/plugins/knowledge/data:/plugins/knowledge/data:ro
      - ./catalog/plugins/prompts/data:/plugins/prompts/data:ro
      - ./catalog/plugins/agents/data:/plugins/agents/data:ro
      - ./catalog/sample-repos/agents-repo:/sample-repos/agents-repo:ro
      - ./catalog/plugins/guardrails/data:/plugins/guardrails/data:ro
      - ./catalog/plugins/policies/data:/plugins/policies/data:ro
      - ./catalog/plugins/skills/data:/plugins/skills/data:ro
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "/usr/local/bin/healthcheck", "http://localhost:8080/readyz"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
```

### Service: postgres

- **Image**: `postgres:16`
- **Health check**: `pg_isready -U catalog -d catalog` every 5 seconds, 5 retries
- **Credentials**: user `catalog`, password `catalog`, database `catalog`
- **Port**: Host 5432 mapped to container 5432

### Service: catalog-server

- **Build**: From `Dockerfile.catalog-server` in the repo root
- **Start condition**: Waits for `postgres` to report `service_healthy`
- **Health check**: Uses the `healthcheck` binary against `/readyz` with a 30-second startup grace period
- **Port**: Host 8080 mapped to container 8080

### Command-line Flags

| Flag | Value | Purpose |
|------|-------|---------|
| `--listen` | `:8080` | HTTP listen address |
| `--sources` | `/config/sources.yaml` | Path to source configuration |
| `--db-type` | `postgres` | Database backend type |
| `--db-dsn` | `postgres://catalog:catalog@postgres:5432/catalog?sslmode=disable` | Database connection string |

### Volume Mounts

| Host Path | Container Path | Mode | Purpose |
|-----------|---------------|------|---------|
| `catalog/config` | `/config` | ro | Source configuration files |
| `catalog/plugins/model/data` | `/plugins/model/data` | ro | Model catalog data |
| `catalog/plugins/mcp/data` | `/plugins/mcp/data` | rw | MCP server data (rw for refresh) |
| `catalog/plugins/knowledge/data` | `/plugins/knowledge/data` | ro | Knowledge source data |
| `catalog/plugins/prompts/data` | `/plugins/prompts/data` | ro | Prompt catalog data |
| `catalog/plugins/agents/data` | `/plugins/agents/data` | ro | Agent catalog data |
| `catalog/sample-repos/agents-repo` | `/sample-repos/agents-repo` | ro | Sample Git repo for agent sources |
| `catalog/plugins/guardrails/data` | `/plugins/guardrails/data` | ro | Guardrails catalog data |
| `catalog/plugins/policies/data` | `/plugins/policies/data` | ro | Policies catalog data |
| `catalog/plugins/skills/data` | `/plugins/skills/data` | ro | Skills catalog data |

### Environment Variables

The catalog-server accepts configuration through command-line flags rather than environment variables in the Compose stack. When running the server outside of Compose, the following environment variables are recognized:

| Variable | Example | Purpose |
|----------|---------|---------|
| `DATABASE_TYPE` | `postgres` | Database backend (`postgres`, `mysql`) |
| `DATABASE_DSN` | `postgres://user:pass@host:5432/db` | Connection string |
| `CATALOG_CONFIG_STORE_MODE` | `file` | Config persistence mode (`file`, `k8s`, `none`) |
| `CATALOG_TENANCY_MODE` | `single` | Tenancy mode: `single` (backward compat) or `namespace` (multi-tenant) |
| `CATALOG_AUTHZ_MODE` | `none` | Authorization mode: `none` (all allowed) or `sar` (K8s SubjectAccessReview) |
| `CATALOG_AUDIT_ENABLED` | `true` | Whether audit middleware is active |
| `CATALOG_AUDIT_RETENTION_DAYS` | `90` | Days to retain audit events before cleanup |
| `CATALOG_AUDIT_LOG_DENIED` | `true` | Whether to record 403 denied actions in audit |
| `CATALOG_JOB_CONCURRENCY` | `3` | Max concurrent refresh worker goroutines |
| `CATALOG_JOB_MAX_RETRIES` | `3` | Max retry attempts per refresh job |
| `CATALOG_JOB_ENABLED` | `true` | Whether the async job system is active |
| `CATALOG_CACHE_ENABLED` | `true` | Whether discovery caching is active |
| `CATALOG_CACHE_DISCOVERY_TTL` | `60` | Discovery endpoint cache TTL in seconds |
| `CATALOG_CACHE_CAPABILITIES_TTL` | `30` | Capabilities endpoint cache TTL in seconds |
| `CATALOG_CACHE_MAX_SIZE` | `1000` | Max entries per cache instance |
| `CATALOG_LEADER_ELECTION_ENABLED` | `false` | Enable K8s Lease-based leader election for HA |
| `CATALOG_MIGRATION_LOCK_ENABLED` | `true` | Enable DB migration locking (safe for single-replica) |
| `CATALOG_NAMESPACES` | (none) | Comma-separated list of allowed namespaces (multi-tenant mode) |
| `POD_NAME` | hostname | Instance identity for leader election |
| `POD_NAMESPACE` | `catalog-system` | Namespace for Lease resources |

## Development Stack Commands

```bash
# Start the full catalog stack (builds from source)
docker compose -f docker-compose.catalog.yaml up --build -d

# Check health
curl -s http://localhost:8080/readyz | python3 -m json.tool

# View logs
docker compose -f docker-compose.catalog.yaml logs -f catalog-server

# Stop and clean up (remove volumes)
docker compose -f docker-compose.catalog.yaml down -v
```

To verify the stack is fully operational after startup:

```bash
# Wait for readiness (polls until 200)
until curl -sf http://localhost:8080/readyz > /dev/null 2>&1; do
  echo "Waiting for catalog-server..."
  sleep 2
done
echo "Catalog server is ready"

# List loaded plugins
curl -s http://localhost:8080/api/plugins | python3 -m json.tool

# Test a plugin endpoint
curl -s http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers | python3 -m json.tool
```

## Running Services

| Service | Address | Container | Notes |
|---------|---------|-----------|-------|
| PostgreSQL | localhost:5432 | catalog-postgres | Database backend |
| Catalog Server | localhost:8080 | catalog-server | Main API server |
| BFF (optional) | localhost:4000 | `go run ./cmd/` | Backend for Frontend, set `CATALOG_SERVER_BASE_URL` |
| Frontend (optional) | localhost:9000 | `npm run start:dev` | React/PatternFly UI |

```
  Browser (:9000)
      |
      v
  BFF (:4000)  <--- CATALOG_SERVER_BASE_URL=http://localhost:8080
      |
      v
  Catalog Server (:8080)
      |
      v
  PostgreSQL (:5432)
```

## Kubernetes Deployment

### deployment.yaml

The Kubernetes deployment uses three probe types to manage the server lifecycle:

```yaml
# deploy/catalog-server/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: catalog-server
  namespace: catalog-system
  labels:
    app: catalog-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: catalog-server
  template:
    metadata:
      labels:
        app: catalog-server
    spec:
      serviceAccountName: catalog-server
      containers:
        - name: catalog-server
          image: catalog-server:latest
          ports:
            - containerPort: 8080
          startupProbe:
            httpGet:
              path: /readyz
              port: 8080
            failureThreshold: 30
            periodSeconds: 2
          livenessProbe:
            httpGet:
              path: /livez
              port: 8080
            periodSeconds: 10
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            periodSeconds: 10
            failureThreshold: 3
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

### Probe Strategy

```
Container Start
      |
      v
Startup Probe (/readyz)         <-- Up to 30 attempts x 2s = 60s max startup
      | (passes once)
      v
+-----+-----+
|           |
v           v
Liveness    Readiness
(/livez)    (/readyz)
every 10s   every 10s
```

| Probe | Endpoint | Period | Failure Threshold | Max Downtime Before Action |
|-------|----------|--------|-------------------|---------------------------|
| Startup | `/readyz` | 2s | 30 | 60s (then container is killed) |
| Liveness | `/livez` | 10s | 3 | 30s (then container is restarted) |
| Readiness | `/readyz` | 10s | 3 | 30s (then removed from Service endpoints) |

- **Startup probe** gates liveness and readiness. The server has up to 60 seconds to complete initialization (database migration, plugin init, initial data load) before Kubernetes kills the container.
- **Liveness probe** hits `/livez`, which always returns 200 with uptime. A failure here indicates the process is hung, not merely degraded.
- **Readiness probe** hits `/readyz`, which checks database connectivity, initial load completion, and plugin health. If any component is degraded, the pod is removed from Service endpoints but not restarted.

### rbac.yaml

The catalog-server needs RBAC permissions when running with `CATALOG_CONFIG_STORE_MODE=k8s` to read and write ConfigMaps for source configuration persistence, and to read Secrets for SecretRef resolution.

```yaml
# deploy/catalog-server/rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: catalog-server
  namespace: catalog-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: catalog-server-configmap-manager
  namespace: catalog-system
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]
    # Required for resolving SecretRef values in source configuration.
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: catalog-server-configmap-manager
  namespace: catalog-system
subjects:
  - kind: ServiceAccount
    name: catalog-server
    namespace: catalog-system
roleRef:
  kind: Role
  name: catalog-server-configmap-manager
  apiGroup: rbac.authorization.k8s.io
```

| Resource | Verbs | Purpose |
|----------|-------|---------|
| ConfigMaps | get, list, watch, create, update, patch | Persist and reload source configuration |
| Secrets | get | Resolve `SecretRef` values in source config (e.g., API tokens) |

## Health Endpoints

| Endpoint | Method | Status Codes | Checks |
|----------|--------|-------------|--------|
| `/healthz` | GET | 200 | Always returns alive status with uptime |
| `/livez` | GET | 200 | Always returns alive status with uptime |
| `/readyz` | GET | 200 / 503 | Database connectivity, initial load completion, plugin health |

### Liveness Response (`/healthz`, `/livez`)

These endpoints always return HTTP 200:

```json
{
  "status": "alive",
  "uptime": "2m35s"
}
```

### Readiness Response (`/readyz`)

Returns HTTP 200 when all components are healthy, HTTP 503 when any component is degraded.

**Healthy (200 OK):**

```json
{
  "status": "ready",
  "components": {
    "database": {
      "status": "up"
    },
    "initial_load": {
      "status": "complete"
    },
    "plugins": {
      "status": "healthy",
      "details": "all 8 plugins healthy"
    }
  }
}
```

**Degraded (503 Service Unavailable):**

```json
{
  "status": "not_ready",
  "components": {
    "database": {
      "status": "down",
      "error": "dial tcp 127.0.0.1:5432: connect: connection refused"
    },
    "initial_load": {
      "status": "pending"
    },
    "plugins": {
      "status": "degraded",
      "details": "6 of 8 plugins healthy"
    }
  }
}
```

### Readiness Component Details

| Component | Possible Status Values | Description |
|-----------|----------------------|-------------|
| `database` | `up`, `down`, `not_configured` | Pings the SQL database; `not_configured` when no DB is wired |
| `initial_load` | `complete`, `pending` | Set to `complete` after first successful config load and plugin init |
| `plugins` | `healthy`, `degraded` | Counts healthy plugins vs total (active + failed) |

## Multi-Tenant Deployment (Phase 8)

To deploy the catalog-server in multi-tenant mode, configure the following:

### 1. Set Tenancy Mode

```bash
CATALOG_TENANCY_MODE=namespace
```

In namespace mode, every request must include a namespace via the `?namespace=` query parameter or the `X-Namespace` HTTP header. Requests without a namespace receive a 400 error.

### 2. Deploy Behind an Auth Proxy

The catalog-server expects identity to be injected by an upstream authenticating proxy. Configure your proxy (Istio, OAuth2-Proxy, Dex, etc.) to set:

| Header | Content |
|--------|---------|
| `X-Remote-User` | Authenticated username |
| `X-Remote-Group` | Comma-separated group memberships |

### 3. Enable SAR Authorization

```bash
CATALOG_AUTHZ_MODE=sar
```

The catalog-server service account needs permission to create SubjectAccessReviews:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: catalog-server-sar-creator
rules:
- apiGroups: ["authorization.k8s.io"]
  resources: ["subjectaccessreviews"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: catalog-server-sar-creator
subjects:
- kind: ServiceAccount
  name: catalog-server
  namespace: catalog-system
roleRef:
  kind: ClusterRole
  name: catalog-server-sar-creator
  apiGroup: rbac.authorization.k8s.io
```

### 4. Create Tenant RBAC Roles

See [Security: RBAC Manifest Examples](./security.md#rbac-manifest-examples) for Role/RoleBinding templates.

### 5. Data Isolation

In multi-tenant mode:
- All entity tables include a `namespace` column
- List/get queries are automatically scoped to the request namespace
- Management state (sources, lifecycle, approvals, audit, jobs) is namespace-scoped
- Cross-namespace queries return only data within the authorized namespace

## HA Deployment (Phase 8)

The catalog-server supports multi-replica deployments with PostgreSQL for high availability.

### Migration Locking

When multiple replicas start simultaneously, migration locking prevents concurrent schema changes:

```bash
CATALOG_MIGRATION_LOCK_ENABLED=true   # Default: true
```

| Database | Locking Strategy |
|----------|-----------------|
| PostgreSQL | Advisory lock (`pg_advisory_lock`) -- blocks until acquired |
| MySQL/SQLite | Table-based lock (`migration_lock` table) with stale lock cleanup (5 min) |

Migration locking is safe for single-replica deployments (minimal overhead).

### Leader Election

For singleton background loops (config reconciliation, audit retention, job workers), enable Kubernetes Lease-based leader election:

```bash
CATALOG_LEADER_ELECTION_ENABLED=true
CATALOG_LEADER_LEASE_NAME=catalog-server-leader       # Default
CATALOG_LEADER_LEASE_NAMESPACE=catalog-system          # From POD_NAMESPACE
CATALOG_LEADER_LEASE_DURATION=15                       # Seconds
CATALOG_LEADER_RENEW_DEADLINE=10                       # Seconds
CATALOG_LEADER_RETRY_PERIOD=2                          # Seconds
```

The service account needs Lease RBAC:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: catalog-server-leader-election
  namespace: catalog-system
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "create", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: catalog-server-leader-election
  namespace: catalog-system
subjects:
- kind: ServiceAccount
  name: catalog-server
  namespace: catalog-system
roleRef:
  kind: Role
  name: catalog-server-leader-election
  apiGroup: rbac.authorization.k8s.io
```

**Behavior:**
- Only the leader replica runs singleton loops (config reconciliation, audit retention, job workers)
- Non-leader replicas serve HTTP requests normally
- On leader loss, a new leader is elected within the lease duration
- `ReleaseOnCancel: true` ensures fast failover on graceful shutdown

### Multi-Replica Considerations

- Scale replicas using standard `spec.replicas` in the Deployment
- All replicas share the same PostgreSQL database
- Job workers use `FOR UPDATE SKIP LOCKED` for safe concurrent job processing
- `/readyz` reports `not_ready` until migrations are complete and initial load succeeds

### Readiness Behavior with Migrations

```
Replica starts
    |
    v
Acquire migration lock (blocks if another replica holds it)
    |
    v
Run AutoMigrate (idempotent)
    |
    v
Release migration lock
    |
    v
Plugin Init + Initial Load
    |
    v
/readyz returns 200
```

Until `/readyz` returns 200, the pod is excluded from Service endpoints, preventing traffic to partially-initialized replicas.

## Key Files

| File | Purpose |
|------|---------|
| `Dockerfile.catalog-server` | Multi-stage container build (Go builder + Alpine runtime) |
| `docker-compose.catalog.yaml` | Development stack: PostgreSQL + catalog-server |
| `deploy/catalog-server/deployment.yaml` | Kubernetes Deployment with startup/liveness/readiness probes |
| `deploy/catalog-server/rbac.yaml` | ServiceAccount, Role (ConfigMaps + Secrets), RoleBinding |
| `cmd/healthcheck/main.go` | Minimal HTTP healthcheck binary for container probes |
| `cmd/catalog-server/main.go` | Server entry point with plugin imports and startup |
| `pkg/catalog/plugin/server.go` | Server lifecycle, route mounting, health endpoint handlers |

---

[Back to Operations](./README.md) | [Next: Security](./security.md)
