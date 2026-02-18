# 04 Performance hardening: caching, pagination, filtering

## Objective

Keep the service stable under large catalogs, concurrent tenants, and frequent refresh.

## Caching strategy

Cache only safe, read-only, low-entropy endpoints:
- plugin discovery: `/api/plugins`
- capabilities: `/api/plugins/{plugin}/capabilities`
- openapi merged spec (if exposed)

Do not cache tenant-scoped lists unless tenant-scoped and invalidated on refresh/apply.

Implementation:
- in-memory LRU per replica with TTL
- optional Redis later if needed

Invalidation:
- on sources apply/refresh completion for that tenant and plugin
- on governance state changes if those states impact list views

## Pagination consistency

All list endpoints MUST:
- accept `pageSize` with bounds
- accept `pageToken`
- return `nextPageToken`
- provide stable ordering with an explicit default `orderBy`

No offset pagination.

## Server-side filtering consistency

All list endpoints MUST:
- support shared `filterQuery`
- map filter fields to indexed columns when possible
- enforce tenant constraint as a hard pre-filter, never as part of `filterQuery`

## Async refresh expectations

Large catalog refresh MUST be async by default:
- refresh returns job id quickly
- list endpoints expose last refresh status and age

## SLO targets (initial)

- p95 list/get latency (no refresh): <= 300ms
- p95 discovery endpoint latency: <= 150ms
- refresh queue start time: <= 10s under normal load

## Definition of Done

- Discovery endpoints cached with TTL and metrics
- All list endpoints paginated and ordered consistently
- filterQuery cannot bypass tenant scoping
- Load test demonstrates stability under concurrent users and refresh load

## Acceptance Criteria

- With 2 tenants, 5 plugins, 10k assets per tenant:
  - list endpoints stay within SLOs with 20 concurrent users
  - refresh does not cause 5xx spikes
- Pagination tokens work across multiple pages without duplicates or missing items

## Verification plan

- Unit tests for pagination envelopes
- E2E for filterQuery correctness
- Load tests: list workload, refresh workload, mixed workload

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

