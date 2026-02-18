# 03 Audit logging and telemetry

## Objective

All state-changing operations must be auditable and diagnosable at org scale.

## Audit event requirements

Capture for every state-changing request:
- timestamp
- tenant namespace
- actor (user, groups)
- request id and correlation id
- operation (plugin, resource type, resource ids, action verb)
- outcome (success/failure, status code, error details)
- revision pointers (before/after, when available)

## Audit data model

Table: `audit_events`
Indexes:
- (namespace, ts desc)
- (actor, ts desc)
- (plugin, ts desc)

Retention:
- configurable TTL
- cleanup worker removes expired events

## API endpoints

- `GET /api/audit/v1alpha1/events` supports filtering and pagination
- `GET /api/audit/v1alpha1/events/{id}`

RBAC:
- `audit:list` and `audit:get`

## Telemetry

Metrics:
- latency and error rate by endpoint and plugin
- refresh queue depth and duration
- authz SAR latency and deny rate
- cache hit/miss for discovery endpoints

Logs:
- structured JSON logs with request id, namespace, plugin, action

Tracing:
- optional OpenTelemetry traces for request, authz, DB, job execution

## Definition of Done

- Every management endpoint writes an audit event on success and failure
- Audit listing supports pagination and filtering
- Retention cleanup runs and is tested
- Metrics and log fields documented

## Acceptance Criteria

- Executing any action produces exactly one audit event
- Denied actions can also be recorded (configurable)
- Ops can query audit across namespaces only if authorized

## Verification plan

- Unit tests: audit writer called on action paths
- E2E: execute actions, verify events, query back
- Load test: audit does not degrade p95 latency beyond target

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

