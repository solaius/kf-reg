# 06 Governance Management Plane: Storage and APIs
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Define persistent storage and API contracts that enable lifecycle, versioning, approvals, and audit across all plugins.

## Architecture
Add a governance service to the catalog-server process:
- shared DB connection
- shared authn/authz
- exposed under a management-plane API prefix
- consumed by UI, CLI, and plugin action framework

## Data model (tables)
- asset_governance
- asset_versions
- env_bindings
- approval_requests
- approval_decisions
- audit_events
- attestations (optional)

## API endpoints (v1alpha1)
Governance:
- GET /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}
- PATCH /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}
- GET /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/history
- GET /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/versions
- POST /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/versions
- GET /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/bindings
- PUT /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/bindings/{env}
- POST /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/actions/{action}

Approvals:
- GET /api/governance/v1alpha1/approvals
- GET /api/governance/v1alpha1/approvals/{id}
- POST /api/governance/v1alpha1/approvals/{id}/approve
- POST /api/governance/v1alpha1/approvals/{id}/reject
- POST /api/governance/v1alpha1/approvals/{id}/cancel

## Auth and authorization
- reuse existing jwt + role extraction patterns
- enforce: viewer, operator, approver, security-approver roles
- audit always includes actor and correlation id

## Definition of Done
- DB migrations added and idempotent
- Governance APIs implemented with unit and integration tests
- Actions integrate with approvals and audit
- No breaking changes to plugin catalog APIs

## Acceptance Criteria
- Governance update persists and appears in UI for any plugin asset
- Version list deterministic ordering and supports pagination
- Binding update audited and visible in history
- Approval endpoints enforce role checks and record decisions
