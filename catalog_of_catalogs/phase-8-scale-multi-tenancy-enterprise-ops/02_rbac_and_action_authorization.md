# 02 RBAC and action authorization

## Objective

Enforce consistent authorization across:
- Plugins
- Resources (sources, assets, artifacts, jobs, approvals)
- Actions (apply, refresh, promote, approve, etc.)

## Decision: Delegate authorization to Kubernetes RBAC via SubjectAccessReview

The catalog-server checks authorization by calling Kubernetes SubjectAccessReview (SAR).
This aligns with existing cluster identity and RBAC policies.

## Identity requirements

The service must run behind an auth proxy (or equivalent) that:
- authenticates requests (OIDC or platform auth)
- forwards user identity to the service (username and groups)
- allows the service to create SAR requests using that identity

## Authorization model

### Resource types (logical)

- plugins
- capabilities
- catalogsources
- assets
- actions
- jobs
- approvals
- audit

### Verbs

- read-only: get, list
- management: create, update, delete, execute (actions), approve

### Mapping to Kubernetes RBAC

Define an API group and resources for RBAC checks:
- apiGroup: `catalog.kubeflow.org`
- resources: `plugins`, `capabilities`, `catalogsources`, `assets`, `actions`, `jobs`, `approvals`, `audit`

Namespace scope:
- tenant-specific resources are namespace-scoped
- cluster-wide visibility is reserved for ops roles

### Authorization flow

For every request:
1. Resolve tenant namespace (if applicable)
2. Compute required permission tuple
3. Call SAR
4. Deny if SAR is not allowed

Caching:
- Optional short-lived caching for SAR results (default TTL 10s)

## Definition of Done

- Every API endpoint has an explicit authorization check
- Every action execution checks authorization for that action and plugin
- Denies return 403 with consistent error schema
- RBAC role examples are included
- Unit tests cover mapping for list sources, execute refresh, approve promotion

## Acceptance Criteria

- Removing a RoleBinding prevents the user from executing the action within cache TTL
- Multi-tenant users cannot cross namespace boundaries even if they guess namespace names
- Ops can perform cluster-wide reads only with the correct ClusterRoleBinding

## Verification plan

- Unit tests: authz mapping table and SAR allow/deny cases
- E2E: deploy with two namespaces and distinct role bindings, validate allowed and denied flows

## Appendix: RBAC manifest snippet

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: catalog-ai-engineer
  namespace: team-a
rules:
- apiGroups: ["catalog.kubeflow.org"]
  resources: ["catalogsources", "assets", "actions", "jobs"]
  verbs: ["get", "list", "create", "update"]
```

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

