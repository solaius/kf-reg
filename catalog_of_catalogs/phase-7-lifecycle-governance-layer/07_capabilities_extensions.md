# 07 Plugin Capabilities Extensions for Lifecycle and Governance
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Extend the Phase 5 capabilities schema so UI and CLI can render lifecycle, versions, approvals, and provenance generically.

## Backward compatibility
All new fields are optional and additive.

## Capability additions
Example:
```json
{
  "entities": [{
    "kind": "Agent",
    "governance": {
      "supported": true,
      "lifecycle": {
        "states": ["draft","approved","deprecated","archived"],
        "actions": ["lifecycle.setState","lifecycle.deprecate","lifecycle.archive","lifecycle.restore"],
        "defaultState": "draft"
      },
      "versioning": {
        "supported": true,
        "labelFormat": "semver|freeform",
        "environments": ["dev","stage","prod"],
        "actions": ["version.create","promotion.bind","promotion.rollback"]
      },
      "approvals": {
        "supported": true,
        "actionsRequiringApprovalMayReturn": true
      },
      "provenance": {
        "supported": true,
        "fields": ["source.type","source.uri","revision.id","integrity.contentDigest","integrity.signature.verified"]
      }
    }
  }]
}
```

## Definition of Done
- Capabilities schema updated and versioned
- UI and CLI read capability flags to enable governance features
- Conformance includes capabilities contract validation

## Acceptance Criteria
- governance.supported=false renders as today
- governance.supported=true shows lifecycle badge, versions panel, approvals UI without plugin-specific code
