# 10 Conformance and Verification Plan
**Date**: 2026-02-17  
**Status**: Spec

## Test layers
- Unit: lifecycle validation, policy evaluation, versions, bindings, audit events, provider provenance
- Integration: migrations, governance APIs, approval gating, audit history pagination
- UI E2E: approvals flow, version selector, promotion updates
- CLI: golden outputs, paging, error codes
- Conformance: capabilities, lifecycle, approvals, promotion, provenance, audit, backward compatibility

## Minimum acceptance matrix
| Area | Must Pass |
|---|---|
| Lifecycle | valid transitions succeed, invalid rejected |
| Approvals | gated action creates request, approvals execute |
| Promotion | bind and rollback work, enforce approved requirement |
| Provenance | every asset has source + revision |
| Audit | every governance change creates audit event |
| Backward compatibility | model catalog endpoints unchanged |

## Definition of Done
- Conformance runs in CI and gates merges
- E2E passes on docker-compose stack
- Matrix covered for Agents and one additional plugin

## Acceptance Criteria
- CI shows governance conformance passing for all governance-enabled plugins
- Local bring-up provides a single command verification path
