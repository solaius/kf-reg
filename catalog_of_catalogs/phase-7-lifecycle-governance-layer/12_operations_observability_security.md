# 12 Operations, Observability, and Security
**Date**: 2026-02-17  
**Status**: Spec

## Observability
Metrics:
- approvals_pending_count
- lifecycle_transition_total by kind and outcome
- promotion_total by env and outcome
- audit_events_total
- signature_verification_total

Logs:
- structured logs for governance actions with correlationId

## Security
- lifecycle changes require operator
- approvals require approver roles
- prod binding can require security-approver via policy
- audit events append-only
- reuse Phase 5 safety patterns for filterQuery and action param validation

## Definition of Done
- Governance readiness and health endpoints
- Key metrics exported
- Audit events immutable and queryable with pagination

## Acceptance Criteria
- Denied action logged with reason and actor
- Metrics reflect actions taken during E2E tests
