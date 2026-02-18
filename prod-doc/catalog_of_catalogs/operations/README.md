# Operations

This section documents deployment, health monitoring, security, and enterprise operations for the catalog-of-catalogs system.

## Contents

| Document | Description |
|----------|-------------|
| [Deployment](./deployment.md) | Docker, Kubernetes, health probes, multi-tenant and HA deployment |
| [Security](./security.md) | RBAC, JWT authentication, SAR authorization, audit logging, SecretRef, injection safety |
| [Enterprise Ops Runbook](./enterprise-ops-runbook.md) | Day-0 install, day-1 operations, troubleshooting, backup and restore |
| [Upgrade Guide](./upgrade-guide.md) | Phase 8 upgrade procedure, migration, rollback |
| [Governance](./governance.md) | Plugin governance checks, supported plugin index, quality gates (Phase 9) |

## Quick Summary

- **Docker Compose** stack with PostgreSQL 16 and catalog-server
- **Kubernetes** deployment with startup/liveness/readiness probes
- **Health endpoints** at `/livez` and `/readyz` with component-level status
- **RBAC** with viewer and operator roles (pre-Phase 8) and SAR-based authorization (Phase 8)
- **Multi-tenancy** with namespace-based isolation and server-side enforcement
- **SAR authorization** delegating access control to Kubernetes RBAC
- **Audit logging** for all management actions with configurable retention
- **JWT authentication** with RS256 signature verification
- **SecretRef** for Kubernetes Secret-backed credentials
- **Redaction** of sensitive values in API responses
- **HA deployment** with migration locking and leader election
- **Async refresh jobs** with database-backed queue and worker pool
- **Plugin governance** with 7 machine-checkable quality gates (Phase 9)
- **Supported plugin index** at `deploy/plugin-index/` with 3 tiers (Phase 9)

---

[Back to Catalog of Catalogs](../README.md)
