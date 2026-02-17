# 14 Open Questions and Decision Log
**Date**: 2026-02-17  
**Status**: Spec

## Decisions needed
1. Version label policy: SemVer required vs optional
2. Environment list: fixed dev, stage, prod vs configurable
3. Default lifecycle for discovered assets: always draft vs trusted-source allow-list
4. Approval policy language: YAML rules only vs optional OPA adapter
5. Signature verification defaults: off by default vs on for OCI sources
6. Audit retention defaults and configuration

## Risks
- versioning complexity
- approvals UX complexity
- provider provenance inconsistency

## Mitigations
- keep governance additive and server-enforced
- avoid plugin-specific UI and CLI
- conformance is the guardrail
