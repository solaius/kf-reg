# Upgrade Guide: Phase 8 (Multi-Tenancy and Enterprise Ops)

This guide covers upgrading from a pre-Phase 8 catalog-server (single-tenant, no authz) to the Phase 8 release with multi-tenancy, authorization, audit, and HA support.

## Pre-Upgrade Checklist

- [ ] Back up the PostgreSQL database (see [Enterprise Ops Runbook](./enterprise-ops-runbook.md#backup-and-restore))
- [ ] Record current environment variables and CLI flags
- [ ] Verify current deployment is healthy (`/readyz` returns 200)
- [ ] Plan maintenance window (migration adds columns; duration depends on table sizes)
- [ ] Prepare RBAC manifests if enabling multi-tenant mode (see [Security](./security.md#rbac-manifest-examples))
- [ ] Ensure Kubernetes API server is accessible if enabling SAR authorization

## Data Migration

### What Changes

Phase 8 adds a `namespace` column to all entity, governance, and management tables:

| Change | Details |
|--------|---------|
| New column | `namespace VARCHAR NOT NULL DEFAULT 'default'` on all entity tables |
| Backfill | Existing rows get `namespace = 'default'` automatically |
| Constraints | Uniqueness constraints updated to include namespace |
| New tables | `audit_events`, `refresh_jobs`, `migration_lock` |

### Migration Behavior

- Migrations are applied automatically on server startup via GORM `AutoMigrate`
- Migration locking prevents concurrent schema changes when multiple replicas start
- The migration is **idempotent** -- re-running is safe
- Existing data is preserved; the `DEFAULT 'default'` ensures backward compatibility

### Estimated Impact

| Table Size | Expected Migration Time |
|------------|------------------------|
| < 10k rows | < 5 seconds |
| 10k-100k rows | 5-30 seconds |
| 100k-1M rows | 30 seconds - 5 minutes |

## Configuration Changes

### New Environment Variables

All new variables have backward-compatible defaults. No configuration changes are required for single-tenant mode.

| Variable | Default | Required? |
|----------|---------|-----------|
| `CATALOG_TENANCY_MODE` | `single` | No -- existing behavior preserved |
| `CATALOG_AUTHZ_MODE` | `none` | No -- all requests allowed by default |
| `CATALOG_AUDIT_ENABLED` | `true` | No -- audit starts automatically |
| `CATALOG_AUDIT_RETENTION_DAYS` | `90` | No |
| `CATALOG_AUDIT_LOG_DENIED` | `true` | No |
| `CATALOG_JOB_CONCURRENCY` | `3` | No |
| `CATALOG_JOB_MAX_RETRIES` | `3` | No |
| `CATALOG_JOB_ENABLED` | `true` | No |
| `CATALOG_CACHE_ENABLED` | `true` | No |
| `CATALOG_CACHE_DISCOVERY_TTL` | `60` | No |
| `CATALOG_LEADER_ELECTION_ENABLED` | `false` | No -- opt-in for HA |
| `CATALOG_MIGRATION_LOCK_ENABLED` | `true` | No -- safe for single-replica |

### No Breaking Changes

- All existing API paths, schemas, and behaviors are unchanged
- The `namespace` query parameter is optional in single-tenant mode
- Existing clients (UI, CLI, BFF) work without modification

## Upgrade Procedure

### Step 1: Single-Tenant Upgrade (Zero Config Change)

The simplest upgrade path: deploy the new image with no configuration changes.

```bash
# Update the container image
kubectl set image deployment/catalog-server \
  catalog-server=catalog-server:phase8

# Watch rollout
kubectl rollout status deployment/catalog-server
```

What happens:
1. New pods start with migration locking enabled
2. One pod acquires the migration lock and runs schema changes
3. Other pods wait for the lock, then proceed
4. All pods report ready once migrations and initial load complete
5. Everything works as before -- single tenant, no authz, audit runs in background

### Step 2: Enable Multi-Tenant Mode (Optional)

After verifying the upgrade works in single-tenant mode, switch to multi-tenant:

```bash
# Update environment
kubectl set env deployment/catalog-server \
  CATALOG_TENANCY_MODE=namespace
```

Now:
- All requests must include `?namespace=` or `X-Namespace` header
- Existing data remains in the `"default"` namespace
- Update clients to pass namespace (UI, CLI, BFF)

### Step 3: Enable Authorization (Optional)

```bash
# Deploy auth proxy if not already present
# Create SAR ClusterRoleBinding (see Security docs)

# Enable SAR authorization
kubectl set env deployment/catalog-server \
  CATALOG_AUTHZ_MODE=sar

# Create Roles and RoleBindings per namespace
kubectl apply -f rbac-manifests/
```

### Step 4: Enable HA Features (Optional)

For multi-replica deployments:

```bash
# Ensure POD_NAME is set via Downward API in the Deployment spec
# Create Lease RBAC (see Deployment docs)

kubectl set env deployment/catalog-server \
  CATALOG_LEADER_ELECTION_ENABLED=true

# Scale up
kubectl scale deployment/catalog-server --replicas=3
```

## Rolling Upgrade

Phase 8 supports rolling upgrades with the following behavior:

1. New pods acquire the migration lock and apply schema changes
2. Old pods continue serving with the existing schema (new columns are nullable/defaulted)
3. New pods report ready and join the Service
4. Old pods are terminated gracefully

**Important:** The migration adds columns with defaults, which is backward compatible. Old pods can continue to read and write data while new pods are starting. There is no need to drain traffic or stop old pods before starting new pods.

## Rollback Procedure

If the upgrade causes issues:

### Quick Rollback (Image Revert)

```bash
kubectl rollout undo deployment/catalog-server
```

The previous image will start and work with the new schema (extra columns are ignored by old code that doesn't reference them).

### Full Rollback (Schema Revert)

If you need to remove the new columns (not recommended unless there are data issues):

1. Stop the catalog-server:
   ```bash
   kubectl scale deployment/catalog-server --replicas=0
   ```

2. Connect to the database and drop the new columns manually. This is database-specific and should be done with extreme caution.

3. Restore from backup (recommended instead of manual column drops):
   ```bash
   pg_restore -h postgres-host -U catalog -d catalog --clean --if-exists backup.dump
   ```

4. Deploy the previous version:
   ```bash
   kubectl rollout undo deployment/catalog-server
   kubectl scale deployment/catalog-server --replicas=1
   ```

## Post-Upgrade Verification

### 1. Health Check

```bash
curl -s http://catalog-server:8080/readyz | python3 -m json.tool
# Should show all components "up"/"complete"/"healthy"
```

### 2. Plugin Verification

```bash
curl -s http://catalog-server:8080/api/plugins | python3 -m json.tool
# Should show all 8 plugins with healthy=true
```

### 3. Namespace Verification (Multi-Tenant Mode)

```bash
# This should work
curl -s "http://catalog-server:8080/api/mcp_catalog/v1alpha1/mcpservers?namespace=default"

# This should fail in multi-tenant mode (no namespace)
curl -s http://catalog-server:8080/api/mcp_catalog/v1alpha1/mcpservers
# Expected: 400 "namespace is required in multi-tenant mode"
```

### 4. Authorization Verification (SAR Mode)

```bash
# Should succeed for authorized user
curl -s "http://catalog-server:8080/api/plugins?namespace=team-a" \
  -H "X-Remote-User: alice" \
  -H "X-Remote-Group: team-a-engineers"

# Should return 403 for unauthorized user
curl -s "http://catalog-server:8080/api/mcp_catalog/v1alpha1/management/apply-source?namespace=team-b" \
  -H "X-Remote-User: alice" \
  -H "X-Remote-Group: team-a-engineers" \
  -X POST
```

### 5. Audit Verification

```bash
# Perform an action, then check audit
curl -s "http://catalog-server:8080/api/audit/v1alpha1/events?pageSize=5" \
  -H "X-Remote-User: ops-admin"
```

### 6. Conformance Tests

```bash
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1
```

---

[Back to Operations](./README.md) | [Deployment](./deployment.md) | [Security](./security.md) | [Enterprise Ops Runbook](./enterprise-ops-runbook.md)
