# 04 Provenance and Integrity
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Provide traceability and integrity guarantees for assets and their artifacts:
- where the asset came from
- what revision it represents
- what changed over time
- optional verification hooks for signed artifacts

## Provenance model
Every asset version includes a provenance block:
```yaml
provenance:
  source:
    type: "yaml|git|http|oci|manual"
    uri: "<file path, repo url, endpoint, oci ref>"
    sourceId: "<catalog source id>"
  revision:
    id: "<git sha, http etag, oci digest, file hash>"
    observedAt: "<rfc3339>"
  integrity:
    contentDigest: "<sha256 optional>"
    signature:
      verified: <bool>
      method: "cosign|pgp|x509|none"
      details: "<string optional>"
```

## Provider responsibilities
- YAML: uri is mounted path, revision.id is sha256(file contents preferred)
- Git: uri is repo url, revision.id is commit SHA
- HTTP: uri is endpoint, revision.id is ETag or digest
- OCI: uri is oci ref, revision.id is digest

## Audit trail requirements
All governance actions emit immutable audit events:
- actor, time, action
- asset uid and version
- outcome and reason
- diff (old/new)

## Signing and verification hooks
Phase 7 adds hooks, not a mandatory signing requirement.
- For OCI-based assets or artifacts, verify signatures using Sigstore Cosign where configured
- Store verification outcome in provenance.integrity.signature
- If verification fails and policy requires verification, deny promotion and approval transitions

## Definition of Done
- Every asset detail view shows source and revision information
- Audit events emitted for lifecycle changes, approvals, promotions, and metadata edits
- OCI verification hook exists and can be enabled in config, with at least one working example in tests

## Acceptance Criteria
- Git-sourced Agent shows repo and commit SHA
- After promote to prod, audit history shows promotion event with actor and timestamp
- With OCI verification enabled and signature invalid, asset is marked unverified and promotion denied by policy
