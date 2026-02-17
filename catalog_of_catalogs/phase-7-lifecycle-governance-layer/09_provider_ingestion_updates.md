# 09 Provider and Ingestion Updates for Governance
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Ensure providers supply enough provenance data and support governance workflows without mutating sources.

## Provider contract updates
All providers return:
- source uri
- revision reference
- observedAt

Provider specifics:
- YAML: revision.id is sha256(file contents preferred)
- Git: revision.id is commit SHA
- HTTP: revision.id is ETag or digest
- OCI: revision.id is digest, optional signature verification

## Ingestion behavior
When a source changes:
- create a new version snapshot in draft (configurable per plugin)
- keep existing approved versions unchanged
- promotion bindings remain pinned unless explicitly changed

## Definition of Done
- All providers populate provenance fields
- At least one plugin demonstrates version snapshots created from source changes
- OCI verification hook available (even if only used by one plugin)

## Acceptance Criteria
- Git-sourced assets show correct repo and commit SHA
- Changing git commit yields new draft version without altering prod binding
