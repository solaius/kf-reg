# 08 UI and CLI: Generic Lifecycle and Governance Experience
**Date**: 2026-02-17  
**Status**: Spec

## UI changes (generic)
- List view: lifecycle column, archived hidden by default, risk and compliance filters
- Detail view panels:
  - Governance (owner, team, SLA, risk, compliance, intended use)
  - Lifecycle (state, transitions)
  - Versions (list, select)
  - Promotion (dev, stage, prod bindings, promote, rollback)
  - Approvals (pending requests, approval history)
  - Provenance (source, revision, signature status)
  - Audit history (events + diffs)

UX:
- Gated action shows requestId and pending status
- Denied action shows policy reason and remediation
- Async actions with clear progress feedback

## CLI changes (generic)
- catalogctl approvals list|approve|reject|get
- catalogctl governance get|set
- catalogctl versions list|get
- catalogctl promote|rollback
- catalogctl history

## Definition of Done
- UI renders governance panels for any governance-enabled entity
- CLI supports lifecycle, approvals, versions, bindings
- No plugin-specific UI or CLI code paths introduced

## Acceptance Criteria
- Approve draft Agent via UI then promote to stage then prod, bindings update
- Approve gated promotion via CLI, action executes and audited
- View audit history and see who changed lifecycle and when
