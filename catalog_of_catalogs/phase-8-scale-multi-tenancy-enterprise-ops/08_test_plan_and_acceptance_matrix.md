# 08 Test plan and acceptance matrix

## Required test categories

1. Tenancy isolation
2. RBAC allow and deny
3. Audit logging
4. Performance and load
5. HA and failover
6. Regression (single-tenant)

## Environments

- Dev: docker-compose or kind
- CI: ephemeral k8s + Postgres
- Staging: multi-replica + auth proxy

## Exit criteria validation

Isolation:
- team-a cannot see or act on team-b

RBAC:
- validate endpoint-by-endpoint mapping against roles

Audit:
- every action emits one audit event
- listing supports pagination and filtering

Performance:
- defined SLOs met under mixed load

HA:
- no double migrations
- no duplicate job execution
- availability maintained during failover

## Definition of Done

- CI includes automated tenancy, RBAC, and audit tests
- Staging playbook exists for HA and full load validation
- Evidence artifacts produced (reports, logs, metrics snapshots)

## Acceptance Criteria

- All exit criteria pass in staging

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

