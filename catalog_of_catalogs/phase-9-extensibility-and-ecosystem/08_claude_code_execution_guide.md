# 08 Claude Code execution guide (Phase 9)

## Objective for Claude

Implement Phase 9 across:
- catalog-gen vNext
- conformance suite
- packaging/distribution strategy (server builder)
- UI hints spec implementation
- documentation kit templates
- governance/index repo format

## Required reading

Read all files in:
- `phase-9-extensibility-and-ecosystem/`

Also align changes with existing:
- plugin framework and catalog-gen (previous phases)
- capabilities schema and universal asset contract (Phase 5)
- multi-tenancy/RBAC/audit/job semantics (Phase 8)

## Recommended agent team

- Coordinator / integration lead
- catalog-gen engineer (templates, determinism, golden tests)
- conformance harness engineer (test harness + CI)
- packaging engineer (server builder, manifest format)
- UI contract engineer (UI hints schema + validation)
- docs engineer (templates + usability)
- governance engineer (supported plugin index + checks)

## Implementation order

1. Define plugin.yaml schema and add catalog-gen support
2. Define UI hints schema and add to capabilities endpoint + OpenAPI vendor extensions
3. Build conformance harness and make core plugins pass
4. Implement server builder pipeline and manifest format
5. Generate docs kit from catalog-gen and validate in CI
6. Implement supported plugin index format + governance checks
7. End-to-end demo: external team plugin → integrated build → shows in UI/CLI

## Completion promise

Do not declare Phase 9 complete until:
- A plugin created out-of-repo can be integrated by changing one manifest/index entry
- UI and CLI display it with no bespoke code changes
- Conformance is the gating mechanism for “supported”
