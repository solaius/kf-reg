# 02 Versioning and Promotion Flows
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Introduce a universal versioning model and environment promotion bindings across all asset types.

## Concepts
### Asset identity
An asset has a stable identity:
- assetUid (stable, internal)
- kind + name (user-facing)
- sourceId (optional, if sourced)

### Version
A version is an immutable snapshot of an asset record plus governance overlay.
Each version includes:
- versionId
- version (human label, recommended SemVer but not required)
- createdAt, createdBy
- sourceRevisionRef (if sourced)
- contentDigest (optional)

### Environment binding
Promotion is implemented as a binding from environment to version.
Binding key is plugin, kind, name, environment. Value is versionId.

Promotion updates bindings, not the version itself.

## Default promotion rules
- To bind a version to stage or prod, version must be approved unless policy overrides
- Deprecated versions can remain bound but trigger warning and require explicit allow flag for new binds
- Archived versions cannot be bound

## Universal actions
- version.create (creates new version snapshot from current asset view)
- promotion.bind (params: environment, versionId)
- promotion.promote (params: fromEnv, toEnv, versionId or latestApproved)
- promotion.rollback (params: environment, targetVersionId)
- version.list (read)
- version.get (read)
- alias.set (optional: stable aliases like latest, stable)

## Source-backed assets and versioning
- YAML and Git:
  - If source record includes a spec.version, capture it as version
  - Git provider captures commit SHA as sourceRevisionRef
  - When source changes, system can auto-create a new draft version snapshot (configurable)
- HTTP and OCI:
  - HTTP: use ETag or content hash as revision ref
  - OCI: use digest as revision ref

## UI requirements
Generic UI supports:
- Version selector in asset detail view
- Environment bindings panel showing dev, stage, prod version
- Promote and rollback actions (subject to policy)
- Indicator when viewing a non-bound version

## CLI requirements
Generic CLI supports:
- catalogctl <plugin> versions <name>
- catalogctl <plugin> promote <name> --to prod --version v1.2.3
- catalogctl <plugin> rollback <name> --env prod --to-version v1.2.2
- catalogctl <plugin> bindings <name>

## Definition of Done
- Version snapshots stored and retrievable for at least Agents and one additional plugin
- Promotion bindings persist, visible, and enforced
- Rollback updates bindings and is audited

## Acceptance Criteria
- Create v1 and v2 for an Agent, approve v2, promote v2 to stage then prod
- Roll back prod to v1 and verify UI and CLI reflect new binding
- Attempt to bind draft to prod is rejected with policy error unless allowed
