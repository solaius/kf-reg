# 11 Migration Plan: Existing Plugins and Data
**Date**: 2026-02-17  
**Status**: Spec

## Approach
- Keep plugin catalog APIs unchanged
- Governance surfaced via overlay fields plus management-plane APIs plus capabilities flags

## Backfill
On first run after migration:
- Create governance overlay records for all known assets with default state draft
- Optional allow-list to auto-approve trusted sources (configurable)

## Definition of Done
- Existing plugins continue to function without changes
- Overlay created for existing assets without manual steps
- No breaking migrations

## Acceptance Criteria
- Start stack on an existing DB and system backfills overlay records safely
- Model catalog list|get behavior unchanged
