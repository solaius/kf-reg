# Phase 7: Lifecycle and Governance Layer
**Date**: 2026-02-17  
**Status**: Spec  
**Owner**: Product  
**Applies to**: catalog-server, plugins, providers, BFF, UI, CLI, conformance suite  
**Dependencies**: Phase 5 Universal Asset Framework, Phase 6 Core Asset Coverage

## Goal
Move from discovery and day-2 operations into enterprise-grade lifecycle control across all assets, consistently across every plugin, without introducing plugin-specific UI or CLI code.

In Phase 6, the Agents plugin proved cross-asset linking across skills, knowledge sources, guardrails, policies, and prompt templates. That makes governance non-optional: one asset often becomes a dependency of many others. Phase 7 adds lifecycle, versioning, approvals, and provenance to manage that dependency graph safely.

## What ships
1. Asset lifecycle state machine
   - draft, approved, deprecated, archived
2. Versioning and promotion flows
   - environment bindings and promotion patterns (dev -> stage -> prod)
3. Governance metadata
   - owner, team, SLA, risk level, intended use, compliance tags (plus extensible metadata)
4. Provenance and integrity
   - source-of-truth references, revision history, audit trail
   - optional signing and verification hooks for artifacts
5. Lightweight approval workflows
   - configurable approval gates by asset type, action, label, and risk and compliance metadata

## Principles
- Universal by default: lifecycle and governance apply to all assets and plugins via shared contracts and services
- Backward compatible: no breaking changes to existing plugin APIs (especially model catalog)
- Source-safe: do not mutate source YAML or Git content. Use an overlay and audit trail for lifecycle, approvals, and promotions
- Capabilities-driven UX: UI and CLI render lifecycle and governance based on plugin capabilities
- Policy as configuration: approval requirements and gates are configured, not hard-coded
- Test-first: conformance expands to verify lifecycle correctness across all plugins

## Scope
### In scope
- Governance overlay persistence model, APIs, and actions
- Lifecycle transitions and enforcement
- Versioning model and environment promotion bindings
- Approval requests and approvals
- Audit events and revision history APIs
- Provider provenance capture (YAML, Git, HTTP, OCI)
- Optional signature verification hooks for OCI artifacts (framework and a working example)
- UI and CLI generic lifecycle and governance features

### Out of scope (Phase 7)
- Full workflow engine or BPM system
- Long-running human task systems beyond lightweight approvals
- Runtime execution of agents (catalog remains discovery and management)
- Organization-wide identity integration beyond pluggable authn and authz contract (use existing JWT role extractor patterns)

## Exit criteria
- Lifecycle state machine is enforced across all assets:
  - invalid transitions are rejected
  - lifecycle visible in UI and CLI
- Versioning and promotion works for at least:
  - Agents (YAML + Git)
  - One additional plugin of your choice
- Approval workflows:
  - at least one approval gate by label and by asset type is enforced
  - UI and CLI can submit and approve
- Provenance:
  - every asset shows a source-of-truth reference and revision
  - audit trail records every governance action
- Conformance suite:
  - includes lifecycle, approvals, promotion, and audit tests
  - runs in CI and passes for all shipped plugins

## Deliverables checklist
- [ ] Governance data model and migrations
- [ ] Management-plane API (governance, approvals, audit, version bindings)
- [ ] Action framework extensions (promote, archive, approve, etc.)
- [ ] Provider provenance adapters for YAML, Git, HTTP, OCI
- [ ] Capabilities schema extensions (lifecycle, versioning, provenance)
- [ ] Generic UI enhancements (lifecycle badges, version selector, approvals, history)
- [ ] CLI enhancements (generic governance verbs)
- [ ] Conformance extensions and E2E verification

## Files in this spec pack
See this folder's numbered markdown files. The Execution Guide contains the recommended PR order and commands.
