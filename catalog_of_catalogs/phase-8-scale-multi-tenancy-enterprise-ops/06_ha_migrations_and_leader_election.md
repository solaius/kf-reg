# 06 HA readiness: migrations, leader election, safe background workers

## Objective

Run catalog-server with multiple replicas safely.

## Database requirements

- HA requires an external DB (PostgreSQL recommended)
- SQLite supported for dev and single-replica use

## Migrations

Rules:
- deterministic and idempotent
- only one instance applies migrations at a time

Strategy:
- migrations table and DB lock
- optional Postgres advisory lock during migration

Avoid concurrent AutoMigrate without locking.

## Leader election (optional)

Singleton loops that should run on one replica:
- periodic cleanup loops
- scheduled scans
- any non-job based background watchers

Implement Kubernetes Lease-based leader election.
- leader runs singleton loops
- non-leaders serve HTTP only

## Worker safety

Preferred: DB locking for job claims so multiple replicas can process jobs safely.
Use leader election only where DB locking is not applicable.

## Readiness and health

- `/readyz` fails if DB not reachable, config invalid, or migrations incomplete
- `/healthz` is liveness and should remain true unless process is broken

## Definition of Done

- Multi-replica deployment with Postgres works reliably
- Migrations run once with locking
- Leader election can be enabled and verified
- Background workers are safe and idempotent
- HA docs included

## Acceptance Criteria

- Scale from 1 to 3 replicas with load and jobs:
  - no data corruption
  - no duplicate job execution
  - no startup deadlocks
- Kill leader:
  - new leader elected within lease duration
  - singleton loops resume
  - service stays available

## Verification plan

- E2E HA in test cluster with 3 replicas and Postgres
- Chaos: random pod kills during refresh jobs
- Integration test for migration lock behavior

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

