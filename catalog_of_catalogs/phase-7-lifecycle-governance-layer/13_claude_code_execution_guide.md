# 13 Claude Code Execution Guide (Phase 7)
**Date**: 2026-02-17  
**Status**: Spec

## Read first
- Phase 7 specs 00-12
- Phase 5 contracts (universal asset contract, action model, capabilities schema)
- PROGRAMMING_GUIDELINES.md

## Recommended milestone order
- M7.1 Governance data model and APIs
- M7.2 Lifecycle state machine and actions
- M7.3 Approvals engine and endpoints
- M7.4 Versioning and promotion bindings
- M7.5 Provider provenance adapters and OCI verification hook
- M7.6 Capabilities, UI, CLI generic enhancements
- M7.7 Conformance and E2E

## Verification commands (examples)
```bash
go test ./...
docker compose -f docker-compose.catalog.yaml up --build -d
curl -s http://localhost:8080/api/plugins | python3 -m json.tool
```

## Completion promise
Iterate until:
- All acceptance criteria in this spec pack are met
- Conformance suite and E2E tests pass
- Bring-up shows governance working for Agents and one additional plugin
