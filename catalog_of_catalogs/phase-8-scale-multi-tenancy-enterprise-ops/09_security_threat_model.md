# 09 Security and threat model

## Primary risks

- Cross-tenant data leakage
- Privilege escalation via RBAC mapping bugs
- DoS via refresh storms
- Provider supply chain risks (HTTP, Git, OCI)

## Mitigations

- Hard tenant pre-filters in repositories
- Central authz middleware with explicit mapping table and tests
- Per-namespace rate limits and quotas for refresh
- Async jobs with bounded concurrency
- Provider fetching safeguards:
  - TLS by default
  - timeouts and max payload sizes
  - allowlists where required

## Definition of Done

- Threats documented with mitigations implemented or explicitly deferred
- Negative test suite covers common bypass attempts

## Acceptance Criteria

- No cross-tenant access in negative tests
- RBAC mapping is complete and tested

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

