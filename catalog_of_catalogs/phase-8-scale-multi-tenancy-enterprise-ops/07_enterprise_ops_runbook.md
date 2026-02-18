# 07 Enterprise ops runbook (install, operate, troubleshoot)

## Day-0 install

- Choose tenancy mode: single or namespace
- Choose DB: sqlite (dev) or postgres (prod)
- Deploy behind auth proxy
- Apply RBAC manifests for ops and tenant roles

## Day-1 operations

Dashboards:
- request latency and error rate
- refresh queue depth and duration
- DB pool saturation
- SAR latency and deny rate
- cache hit rate

Operational knobs:
- refresh concurrency
- job retry policy
- cache TTL
- audit retention TTL

## Troubleshooting guide

- 403 spikes: RoleBindings, SAR connectivity
- Slow list/get: indexes, filter field usage, page size
- Refresh backlog: concurrency, provider latency, remote rate limits
- Migration stuck: DB lock holder, connectivity

## Backup and restore

- Postgres backup guidance
- Restore validation steps:
  - schema check
  - conformance tests
  - tenant isolation tests

## Definition of Done

- Runbook covers install, upgrade, backup, restore, incident patterns
- Healthcheck behavior documented and tested
- Operator checklist included

## Acceptance Criteria

- A new operator can install and operate the service using this runbook

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

