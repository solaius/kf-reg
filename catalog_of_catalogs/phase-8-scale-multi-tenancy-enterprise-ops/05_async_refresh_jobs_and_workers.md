# 05 Async refresh jobs and workers

## Objective

Convert refresh operations into reliable async jobs that can be queued, rate-limited, retried, and observed.

## Job model

Table: `refresh_jobs`
- id (uuid)
- namespace
- plugin
- source_id (optional)
- requested_by
- requested_at
- state: queued, running, succeeded, failed, canceled
- progress (optional)
- message (optional)
- started_at, finished_at
- attempt_count
- last_error
- idempotency_key (optional)

## API

- `POST /api/{plugin}/sources:refresh` returns 202 + job id
- `GET /api/jobs/v1alpha1/refresh/{id}`
- `GET /api/jobs/v1alpha1/refresh?namespace=&plugin=&state=&limit=&pageToken=`

RBAC:
- enqueue: `jobs:create`
- read: `jobs:get`, `jobs:list`

## Worker model

- Worker pool processes queued jobs
- Safe HA strategies:
  - DB row locking (`FOR UPDATE SKIP LOCKED`) to claim jobs, or
  - leader-only job processing using leader election

Concurrency:
- global max
- per namespace max
- per plugin max

Retries:
- exponential backoff
- max attempts configurable
- retry only on transient errors

Cancellation:
- allow cancel for queued
- running cancellation is best-effort and cooperative

## Definition of Done

- Refresh endpoints enqueue jobs and return quickly
- Job status endpoints exist with RBAC
- Workers process jobs with concurrency limits
- Jobs are safe under multi-replica deployments (no duplicates)
- Metrics exist for queue depth and job durations

## Acceptance Criteria

- Refresh does not block HTTP longer than 1s
- Same idempotency key does not create duplicate work
- Under 3 replicas, each job runs exactly once

## Verification plan

- Unit tests: state transitions, retry policy, cancellation
- E2E: enqueue and observe processing, verify data updated
- HA: kill pods mid-job, ensure safe retry or continuation

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

