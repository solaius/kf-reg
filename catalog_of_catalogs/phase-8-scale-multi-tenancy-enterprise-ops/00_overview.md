# Phase 8: Scale, Multi-tenancy, and Enterprise Ops

## Goal

Make the catalog-server safe at organization scale and easy to run as a platform service:

- Multi-namespace and multi-tenant separation
- Strong RBAC model across plugins and actions
- Audit logging for all management actions
- Performance hardening (caching, consistent pagination/filtering, async refresh for large catalogs)
- HA readiness (safe migrations, leader election where needed, safe background workers)

## Non-goals

- Deployment lifecycle management for models/agents/MCPs
- Replacing registry semantics (MLflow registry remains the registry layer)
- Building a full IAM/IdP system (we delegate authn/authz to the platform)

## Key principles

1. Tenant isolation is enforced server-side, not implied by UI conventions
2. Authorization is policy-driven and consistently applied to every plugin and action
3. All management actions are auditable: who, what, when, where, why (request context)
4. Scale features must be measurable (SLOs, load tests) and operable (dashboards, runbooks)
5. HA must be deterministic: no double-migrations, no duplicate background jobs, safe failover

## Personas

- Ops for AI (platform operator): installs and operates the service, sets tenancy and RBAC, monitors health and performance
- AI engineer (tenant user): manages sources and assets within authorized tenant(s), uses UI and CLI

## What ships (Phase 8)

- Tenant context model and multi-namespace behavior for all APIs
- RBAC enforcement for:
  - plugin discovery
  - list/get for assets and sources
  - all management verbs/actions (apply, enable, disable, refresh, tag, promote, approve, etc.)
- Audit event capture for every state-changing operation
- Performance features:
  - response caching for discovery/capabilities endpoints
  - guaranteed pagination and server-side filtering semantics across plugins
  - async refresh jobs for large catalogs and predictable refresh concurrency
- HA readiness:
  - safe migrations with locking
  - leader election for singleton background workers (optional but supported)
  - idempotent workers and safe retry behavior

## Exit criteria

- Two teams use the platform concurrently with:
  - isolation (no cross-tenant visibility or actions)
  - correct RBAC enforcement (positive and negative tests)
  - stable performance under refresh load (defined SLOs met)

## Deliverables

See the numbered specs in this directory. Each includes:
- Definition of Done (DoD)
- Acceptance Criteria (AC)
- Verification plan

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

