# 10 Upgrade and backwards compatibility

## Objective

Introduce Phase 8 without breaking existing single-tenant users.

## Compatibility modes

- Single-tenant (default): no namespace required
- Namespace multi-tenant: namespace required or inferred, with server enforcement

## Data migration approach

- Add namespace columns with default value
- Backfill existing rows to default namespace
- Add composite unique constraints including namespace
- Prefer two-step migrations for non-nullable additions

## Rolling upgrade

- Migrations run once with locking
- Rollout tested with version n-1 to n

## Definition of Done

- Upgrade guide included
- Backwards compatibility tests pass (model and mcp)
- Rolling upgrade validated in staging

## Acceptance Criteria

- Existing config works unchanged in single-tenant mode after upgrade
- Multi-tenant mode can be enabled with explicit config and RBAC manifests

## References

The following upstream docs informed design choices. URLs are provided as code blocks to keep the spec self-contained.

```text
https://kubernetes.io/docs/concepts/security/multi-tenancy/
https://kubernetes.io/docs/concepts/security/rbac-good-practices/
https://kubernetes.io/docs/reference/access-authn-authz/authorization/
https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/
https://www.kubeflow.org/docs/components/central-dash/overview/
https://www.kubeflow.org/docs/components/central-dash/profiles/
https://www.kubeflow.org/docs/components/pipelines/operator-guides/multi-user/
https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/leaderelection
https://kubernetes.io/blog/2016/01/simple-leader-election-with-kubernetes/
https://gorm.io/docs/migration.html
```

