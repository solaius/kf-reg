# 11 Claude Code execution guide (Phase 8)

## Objective for Claude

Implement Phase 8 features according to specs in this folder, with:
- contract-first API updates
- RBAC via Kubernetes SAR
- tenant scoping enforced server-side
- audit logging for management actions
- async refresh jobs
- HA safe migrations and optional leader election
- comprehensive tests and load validation

## Required reading

Read all files in:
- `phase-8-scale-multitenancy-enterprise-ops/`

## Suggested agent team

- Lead coordinator
- Authz and SAR specialist
- DB and migrations engineer
- Async jobs and worker engineer
- UI and CLI engineer
- Test engineer
- Docs and runbook engineer

## Recommended implementation order

1. Tenancy plumbing (namespace resolution, tenant filters)
2. Authz middleware and RBAC mapping
3. DB migrations and indexes
4. Audit completion and retention worker
5. Async refresh jobs and job API
6. Performance caching for discovery endpoints
7. HA readiness: migration lock and leader election wiring
8. UI and CLI namespace selection
9. E2E, HA, and load tests
10. Acceptance evidence and docs sync

## Completion promise

Do not declare Phase 8 complete until:
- Exit criteria tests pass in a multi-tenant staging environment
- Evidence artifacts are produced (reports, logs, metrics)
- Docs match implementation

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

