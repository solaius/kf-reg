# 01 Tenancy model and namespace isolation

## Problem

We need to run a single catalog-server as a shared platform service while ensuring:
- tenant isolation (team A does not see or manage team B assets/sources)
- multi-namespace operation aligned with Kubeflow's Profiles/Namespaces model

## Decision: Tenant = Namespace (Profile namespace)

Tenant context is a required part of all management operations and list/get visibility rules.
- Tenant identifier: Kubernetes namespace string (for Kubeflow, this is typically a Profile namespace)
- Cross-namespace access is possible only if granted by RBAC

### Request tenant resolution

1. Prefer explicit request tenant:
   - Query param: `?namespace=<ns>` for list/get
   - Body field: `namespace` for action requests that mutate state
2. If omitted:
   - UI/BFF may inject a default from the user session (active profile/namespace)
   - CLI defaults to current kube-context namespace unless overridden
3. Server MUST reject state-changing requests without a resolved tenant context

### Tenant scoping rules

- All sources are namespaced (a source belongs to exactly one tenant)
- All management state is namespaced (lifecycle, approvals, audit events, refresh jobs)
- Imported assets MUST be logically partitioned:
  - add `namespace` columns, or
  - include namespace in composite unique keys,
  - always apply tenant filters in repositories

## Data model changes

For each plugin that persists entities:
- Add a `namespace` (tenant) column to entity rows
- Update uniqueness constraints to include namespace

For shared governance tables:
- `namespace` is REQUIRED on:
  - lifecycle state rows
  - promotion bindings
  - approvals
  - audit events
  - async jobs

## API updates (backward compatible)

- Keep existing model and MCP APIs unchanged for paths.
- Add optional `namespace` query param to list endpoints for multi-tenant mode.
- Add an API endpoint to discover namespaces available to the user:
  - `GET /api/tenancy/v1alpha1/namespaces`

Backward compatibility modes:
- Single-tenant mode: `TENANCY_MODE=single` uses a fixed namespace, e.g. `default`
- Multi-tenant mode: `TENANCY_MODE=namespace` and namespace is required (or inferred) per request.

## Definition of Done

- All list/get endpoints accept tenant context and enforce tenant filtering server-side
- All management operations require a resolved tenant and persist tenant-scoped state
- CLI and UI can select namespace and propagate it consistently
- No cross-tenant data leakage is possible via list/get, filterQuery, or openapi merges
- Migration scripts update existing tables without breaking single-tenant deployments

## Acceptance Criteria

- Given two namespaces `team-a`, `team-b` with different sources:
  - AI engineer in `team-a` cannot see any `team-b` sources or assets in UI or CLI
  - AI engineer in `team-a` cannot execute any action against `team-b`
  - Ops role can see both if granted cluster-level RBAC
- Attempted access to unauthorized namespace returns 403 with a consistent error envelope
- Unit tests cover repository tenant filtering for at least 3 plugins (model, mcp, agents)

## Verification plan

- E2E:
  - Create two namespaces and two service accounts with distinct RoleBindings
  - Seed sources in both namespaces
  - Validate isolation across list/get and management actions
- Security:
  - Fuzz filterQuery inputs to ensure tenant constraint cannot be bypassed
- Regression:
  - Single-tenant mode tests pass unchanged

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

