# 07 Verification plan and acceptance matrix

## Phase 9 acceptance matrix

### catalog-gen vNext
- [ ] init produces plugin.yaml + docs + conformance scaffold
- [ ] generate is deterministic and regen safe
- [ ] validate catches missing/invalid metadata with clear errors

### Conformance suite
- [ ] core plugins pass conformance (model, mcp, knowledge sources)
- [ ] conformance failures are actionable
- [ ] conformance report artifact produced in CI

### Packaging/distribution
- [ ] server builder builds an image from a manifest
- [ ] compatibility matrix auto-generated
- [ ] optional plugin built outside repo can be integrated with minimal coordination

### UI hints spec
- [ ] capabilities endpoint exposes UI hints
- [ ] UI renders a new plugin with no UI code changes
- [ ] CLI can discover plugin fields and supported actions

### Documentation kit
- [ ] generated docs are complete and usable
- [ ] a new dev can follow docs to publish a plugin

## Exit criteria verification

Demonstration:
- Team B creates a plugin in a separate repo
- They pass conformance and publish artifacts
- Platform integrates plugin by:
  - updating a manifest or plugin index entry
- UI and CLI show plugin immediately in the new server build

## Definition of Done

- All above checks are green in CI and validated in a staging cluster
