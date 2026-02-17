# Phase 5: Universal Asset Framework (Spec Pack)

Date: 2026-02-16

## Goal
Make UI and CLI truly generic across any catalog plugin, and make “management verbs” consistent, so new asset types (plugins) appear in UI and CLI without frontend or CLI code changes.

## Why this phase exists
Phases 1–4 established:
- Plugin-based catalog-server with multiple catalog types
- Management plane for sources.yaml (validate/apply/revisions/rollback)
- Early CLI and UI management experiences

Phase 5 turns the system into a platform by standardizing:
- Asset shape (contract)
- Actions (verbs) and how they’re discovered/executed
- Capabilities (how UI/CLI discover what each plugin supports)
- UI composition (generic components with plugin-provided hints)
- CLI composition (plugin-driven command surface)

## Primary outcomes
1) Universal Asset Contract that every plugin uses (or maps to) for display + automation
2) Universal Action Model that plugins opt into for consistent ops and lifecycle verbs
3) Plugin Capabilities Schema returned by the server that fully drives UI + CLI
4) Generic UI components library rendered from capabilities + OpenAPI + hints
5) CLI v2 that is plugin-driven, consistent, and testable
6) Exit criteria proved by adding Knowledge Sources plugin with zero UI/CLI changes

## Non-goals
- Building every possible asset plugin (that’s later phases)
- Replacing existing model_catalog API paths (must remain stable)
- Turning the catalog into a general-purpose workflow engine

## Personas
- Ops for AI: configures sources, validates, applies, refreshes, monitors health and drift
- AI Engineer: discovers and selects assets, inspects details/artifacts, links assets into solutions

## Exit Criteria (Phase 5)
- Add a brand-new plugin “knowledge sources” and it shows up in UI and CLI with no frontend or CLI code changes
- Model and MCP plugins expose capabilities that allow generic UI/CLI to render and operate them
- Action execution works end-to-end (UI + CLI) for a baseline set of verbs across sources and assets
- Conformance test suite passes for model, mcp, and knowledge sources plugins

## Definition of Done (Phase 5, overall)
- All acceptance criteria in the specs are met
- All required tests in the verification plan are green in CI
- Docs: developer guide for adding a plugin with Phase 5 standards (capabilities + actions + hints)

## References (inspiration)
- Kubernetes discovery + OpenAPI publishing patterns (client-driven schema-aware tooling) citeturn2view3
- Kubeflow Model Catalog REST API reference citeturn2view0
- Backstage catalog model semantics (metadata/spec pattern) citeturn2view2
