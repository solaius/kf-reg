# 03 Governance Metadata
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Define a universal set of governance metadata fields that apply across all asset types, while allowing plugin-specific extensions.

## Universal governance schema
```yaml
governance:
  owner:
    principal: "<user or service principal>"
    displayName: "<optional>"
    email: "<optional>"
  team:
    name: "<team name>"
    id: "<optional>"
  sla:
    tier: "gold|silver|bronze|none"
    responseHours: <int optional>
  risk:
    level: "low|medium|high|critical"
    categories: ["pii","security","legal","safety","reliability","cost"]
  intendedUse:
    summary: "<string>"
    environments: ["dev","stage","prod"]
    restrictions: ["internal-only","no-pii","no-external-callouts"]
  compliance:
    tags: ["sox","hipaa","gdpr","export-control"]
    controls: ["<control-id>"]
  lifecycle:
    state: "draft|approved|deprecated|archived"
    reason: "<optional>"
    changedBy: "<principal>"
    changedAt: "<rfc3339>"
  audit:
    lastReviewedAt: "<rfc3339 optional>"
    reviewCadenceDays: <int optional>
```

## Labels and annotations
Labels and annotations remain universally supported and are used for selection, filtering, and policy targeting.

Governance metadata is distinct from free-form annotations:
- governance fields are structured and validated
- annotations can be arbitrary

## Validation rules
- risk.level must be one of allowed enums
- compliance.tags must come from a configured allow-list (configurable)
- owner.principal must be non-empty for approved assets unless policy overrides
- intendedUse.environments must be subset of configured environments

## Definition of Done
- Governance metadata stored in overlay and returned for all assets
- UI renders governance section in detail view generically
- Filtering supports key governance fields (risk.level, compliance.tags, owner and team)

## Acceptance Criteria
- Update governance metadata for an Agent and see it in UI and CLI
- Filter list by risk.level=high and compliance.tags includes gdpr
- Attempt approval with missing required fields fails with actionable message
