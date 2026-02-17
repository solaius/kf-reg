# 01 Lifecycle State Machine
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Define a consistent lifecycle model for all assets that can be applied uniformly across plugins without plugin-specific UI or CLI behavior.

## Lifecycle states
- draft
  - default state for newly discovered assets unless the provider supplies an explicit state mapping
  - editable in governance overlay
  - not selectable by default for production bindings
- approved
  - safe for intended use, within governance tags
  - required for promotion to stage and prod unless policy config says otherwise
- deprecated
  - still readable and discoverable
  - not recommended for new work
  - can remain bound in prod but triggers warnings and policy checks
- archived
  - removed from default discovery surfaces
  - always retrievable for audit and history
  - cannot be promoted or approved
  - can be restored to deprecated or draft depending on policy

## State transition rules
### Allowed transitions (default)
| From | To | Requires approval by default |
|---|---|---|
| draft | approved | Yes (configurable) |
| approved | deprecated | No (configurable) |
| deprecated | archived | Yes (configurable) |
| approved | archived | Yes (configurable) |
| deprecated | approved | Yes (configurable) |
| archived | deprecated | Yes (configurable) |
| archived | draft | Yes (configurable) |

### Disallowed transitions (default)
- draft -> deprecated (must approve first, then deprecate)
- draft -> archived (must be deprecated or explicitly allowed by policy)
- archived -> approved (must restore to deprecated or draft, then approve)

Policies can override allowed transitions, but disallowed transitions must remain disallowed unless an explicit config flag enables them.

## Lifecycle enforcement
Lifecycle is enforced at three layers:
1. UI layer
   - Draft assets display warnings
   - Archived assets hidden by default, toggle to show
2. CLI layer
   - Default list excludes archived unless include-archived is set
   - Promotion commands refuse non-approved unless policy allows
3. Server layer (authoritative)
   - All lifecycle transitions go through the governance service
   - Direct edits to lifecycle fields via plugin APIs are rejected

## Canonical fields (universal contract)
All assets must expose lifecycle via universal governance fields:
```yaml
governance:
  lifecycle:
    state: draft|approved|deprecated|archived
    reason: "<optional string>"
    changedBy: "<principal>"
    changedAt: "<rfc3339 timestamp>"
```

## Actions (universal)
Lifecycle is modified through actions:
- lifecycle.setState (params: state, reason)
- lifecycle.deprecate (params: reason, replacementRef optional)
- lifecycle.archive (params: reason)
- lifecycle.restore (params: targetState, reason)

Actions may require approval depending on policy.

## Definition of Done
- Lifecycle state and history is visible for all assets in UI and CLI
- Server rejects invalid transitions with clear errors
- Policy config can require approvals per transition
- Archived assets excluded from default list endpoints unless requested

## Acceptance Criteria
- Given an asset in draft, a user cannot promote it to prod unless policy allows
- Given an asset in approved, user can set deprecated and see warning in UI
- Given an asset in archived, it does not appear in default lists but can be fetched directly
- Given an invalid transition, server returns 4xx with a machine-readable error code
