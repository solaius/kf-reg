# Enterprise Ops Runbook

Operational guide for installing, operating, and troubleshooting the catalog-server as an organization-scale platform service.

## Day-0: Install

### 1. Choose Tenancy Mode

| Mode | Env Var | When to Use |
|------|---------|-------------|
| Single-tenant | `CATALOG_TENANCY_MODE=single` | Development, single-team, backward compatibility |
| Multi-tenant | `CATALOG_TENANCY_MODE=namespace` | Multiple teams, namespace isolation required |

Single-tenant is the default. All requests are scoped to the `"default"` namespace automatically.

### 2. Database Setup

PostgreSQL 14+ is recommended for production. SQLite can be used for development.

```bash
# PostgreSQL example
DATABASE_TYPE=postgres
DATABASE_DSN="postgres://catalog:PASSWORD@postgres-host:5432/catalog?sslmode=require"
```

Ensure the database user has permissions to create tables and indexes (for AutoMigrate).

### 3. Auth Proxy

Deploy an authenticating proxy in front of the catalog-server. The proxy must set:

| Header | Purpose |
|--------|---------|
| `X-Remote-User` | Authenticated username |
| `X-Remote-Group` | Comma-separated groups |

Common proxy choices: Istio with OIDC, OAuth2-Proxy, Envoy with ext-authz, Dex.

### 4. Authorization

Enable SAR-based authorization for production:

```bash
CATALOG_AUTHZ_MODE=sar
```

Create the ClusterRoleBinding for SAR:

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

### 5. RBAC Manifests

Create Roles and RoleBindings per namespace for each team:

```yaml
# AI Engineer role for team-a
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
  name: team-a-engineers
  namespace: team-a
subjects:
- kind: Group
  name: team-a-engineers
roleRef:
  kind: Role
  name: catalog-ai-engineer
  apiGroup: rbac.authorization.k8s.io
```

### 6. Verify Installation

```bash
# Check health
curl -s http://catalog-server:8080/readyz | python3 -m json.tool

# Verify plugins are loaded
curl -s http://catalog-server:8080/api/plugins | python3 -m json.tool

# Test namespace resolution (multi-tenant mode)
curl -s "http://catalog-server:8080/api/mcp_catalog/v1alpha1/mcpservers?namespace=team-a" \
  -H "X-Remote-User: alice" \
  -H "X-Remote-Group: team-a-engineers"
```

## Day-1: Operations

### Health Checks

| Endpoint | Purpose | Expected |
|----------|---------|----------|
| `/livez` | Process alive | Always 200 |
| `/readyz` | Ready for traffic | 200 when DB + plugins + initial load OK |

```bash
# Quick health check
curl -sf http://catalog-server:8080/readyz > /dev/null && echo "OK" || echo "NOT READY"

# Detailed health
curl -s http://catalog-server:8080/readyz | python3 -m json.tool
```

### Monitoring

Key metrics to watch:

| Metric | Source | Alert Threshold |
|--------|--------|----------------|
| Request latency (p95) | Structured logs, `duration` field | > 300ms for list/get |
| Error rate | HTTP status codes 5xx | > 1% |
| Refresh queue depth | `GET /api/jobs/v1alpha1/refresh?state=queued` | > 50 queued jobs |
| SAR deny rate | Audit events with `outcome=denied` | Sustained spike = config issue |
| Cache hit rate | `X-Cache: HIT/MISS` response headers | < 50% = TTL too short |
| DB pool saturation | PostgreSQL `pg_stat_activity` | Active connections near pool max |

### Operational Knobs

| Setting | Env Var | Default | Guidance |
|---------|---------|---------|----------|
| Refresh concurrency | `CATALOG_JOB_CONCURRENCY` | 3 | Increase for large catalogs; watch DB load |
| Job retry count | `CATALOG_JOB_MAX_RETRIES` | 3 | Increase for flaky remote sources |
| Discovery cache TTL | `CATALOG_CACHE_DISCOVERY_TTL` | 60s | Lower for fast plugin updates; higher for stability |
| Audit retention | `CATALOG_AUDIT_RETENTION_DAYS` | 90 | Align with org compliance requirements |
| SAR cache TTL | 10s (hardcoded) | 10s | RBAC changes take up to 10s to propagate |
| Leader lease duration | `CATALOG_LEADER_LEASE_DURATION` | 15s | Lower for faster failover; higher for stability |

### Viewing Audit Trail

```bash
# List recent audit events for a namespace
curl -s "http://catalog-server:8080/api/audit/v1alpha1/events?namespace=team-a&pageSize=10" \
  -H "X-Remote-User: ops-admin" | python3 -m json.tool

# Filter by actor
curl -s "http://catalog-server:8080/api/audit/v1alpha1/events?actor=alice" \
  -H "X-Remote-User: ops-admin" | python3 -m json.tool
```

### Viewing Job Status

```bash
# List all queued/running jobs
curl -s "http://catalog-server:8080/api/jobs/v1alpha1/refresh?state=queued" \
  -H "X-Remote-User: ops-admin" | python3 -m json.tool

# Check a specific job
curl -s "http://catalog-server:8080/api/jobs/v1alpha1/refresh/JOB_ID" \
  -H "X-Remote-User: ops-admin" | python3 -m json.tool
```

## Troubleshooting

### 403 Spikes

**Symptoms:** Users report "forbidden" errors accessing catalog resources.

**Diagnosis:**

1. Check audit events for denied actions:
   ```bash
   curl -s "http://catalog-server:8080/api/audit/v1alpha1/events?namespace=NAMESPACE" \
     -H "X-Remote-User: ops-admin" | python3 -m json.tool
   ```

2. Verify RoleBindings exist for the user/group in the target namespace:
   ```bash
   kubectl get rolebindings -n NAMESPACE
   kubectl describe rolebinding BINDING_NAME -n NAMESPACE
   ```

3. Test SAR directly:
   ```bash
   kubectl auth can-i list catalogsources.catalog.kubeflow.org \
     --as=USERNAME --namespace=NAMESPACE
   ```

4. Check SAR connectivity from the catalog-server pod:
   ```bash
   kubectl exec -it catalog-server-POD -- curl -sf http://localhost:8080/readyz
   ```

**Resolution:** Create or fix the RoleBinding. Changes take effect within 10 seconds (SAR cache TTL).

### Slow List/Get

**Symptoms:** List or get endpoints respond slowly (> 300ms p95).

**Diagnosis:**

1. Check page size -- large pages are expensive:
   ```
   ?pageSize=100   # Consider reducing to 20
   ```

2. Check filter usage -- unindexed filters cause full scans:
   ```
   ?filterQuery=name='example'   # name is indexed
   ?filterQuery=description LIKE '%test%'   # description may not be indexed
   ```

3. Check database performance:
   ```sql
   -- PostgreSQL: check slow queries
   SELECT pid, now() - pg_stat_activity.query_start AS duration, query
   FROM pg_stat_activity
   WHERE state != 'idle'
   ORDER BY duration DESC;
   ```

4. Check for missing indexes on namespace column.

**Resolution:** Reduce page size, use indexed filter fields, add database indexes if needed.

### Refresh Backlog

**Symptoms:** Many queued jobs, refreshes taking a long time.

**Diagnosis:**

1. Check queue depth:
   ```bash
   curl -s "http://catalog-server:8080/api/jobs/v1alpha1/refresh?state=queued" \
     -H "X-Remote-User: ops-admin"
   ```

2. Check for failed jobs:
   ```bash
   curl -s "http://catalog-server:8080/api/jobs/v1alpha1/refresh?state=failed" \
     -H "X-Remote-User: ops-admin"
   ```

3. Check provider latency (remote sources like Hugging Face may be rate-limited).

4. Check worker concurrency:
   ```
   CATALOG_JOB_CONCURRENCY=3   # Consider increasing
   ```

**Resolution:**
- Increase `CATALOG_JOB_CONCURRENCY` for more parallel workers
- Check remote source rate limits and add authentication tokens
- Cancel stuck jobs: `POST /api/jobs/v1alpha1/refresh/{id}:cancel`

### Migration Stuck

**Symptoms:** Server does not become ready. `/readyz` shows `"initial_load": {"status": "pending"}`.

**Diagnosis:**

1. Check server logs for migration lock messages:
   ```bash
   kubectl logs -l app=catalog-server --tail=50
   ```

2. For PostgreSQL, check for held advisory locks:
   ```sql
   SELECT * FROM pg_locks WHERE locktype = 'advisory';
   ```

3. For table-based locking, check the migration_lock table:
   ```sql
   SELECT * FROM migration_lock;
   ```

4. Check database connectivity:
   ```bash
   kubectl exec -it catalog-server-POD -- curl -sf http://localhost:8080/livez
   ```

**Resolution:**
- If a pod crashed while holding the lock, the advisory lock is released automatically on disconnect
- For table-based locks, stale locks older than 5 minutes are auto-cleaned
- Manual cleanup: `DELETE FROM migration_lock WHERE id = 'migration';`
- Restart pods after resolving DB connectivity issues

### Leader Election Issues

**Symptoms:** Singleton loops (audit retention, job workers) are not running.

**Diagnosis:**

1. Check which pod is the leader:
   ```bash
   kubectl get lease catalog-server-leader -n catalog-system -o yaml
   ```

2. Check pod logs for leader election messages:
   ```bash
   kubectl logs -l app=catalog-server | grep -i "leader"
   ```

3. Verify Lease RBAC:
   ```bash
   kubectl auth can-i update leases.coordination.k8s.io \
     --as=system:serviceaccount:catalog-system:catalog-server \
     -n catalog-system
   ```

**Resolution:** Fix Lease RBAC, ensure `CATALOG_LEADER_ELECTION_ENABLED=true`, and `POD_NAME` is unique per pod (set via Downward API).

## Backup and Restore

### PostgreSQL Backup

```bash
# Full logical dump
pg_dump -h postgres-host -U catalog -d catalog -Fc > catalog_backup_$(date +%Y%m%d).dump

# With specific tables
pg_dump -h postgres-host -U catalog -d catalog \
  --table='mcp_*' --table='audit_*' --table='refresh_*' \
  -Fc > catalog_partial_$(date +%Y%m%d).dump
```

### Restore

```bash
# Stop the catalog-server
kubectl scale deployment catalog-server --replicas=0

# Restore from backup
pg_restore -h postgres-host -U catalog -d catalog --clean --if-exists \
  catalog_backup_20260217.dump

# Restart
kubectl scale deployment catalog-server --replicas=3
```

### Post-Restore Validation

1. **Schema check:** Verify migrations complete:
   ```bash
   curl -s http://catalog-server:8080/readyz | python3 -m json.tool
   ```

2. **Plugin health:** Verify all plugins load:
   ```bash
   curl -s http://catalog-server:8080/api/plugins | python3 -m json.tool
   ```

3. **Tenant isolation:** Verify namespace scoping:
   ```bash
   curl -s "http://catalog-server:8080/api/mcp_catalog/v1alpha1/mcpservers?namespace=team-a" \
     -H "X-Remote-User: team-a-user"
   ```

4. **Conformance tests:** Run the conformance suite:
   ```bash
   CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1
   ```

---

[Back to Operations](./README.md) | [Deployment](./deployment.md) | [Security](./security.md)
