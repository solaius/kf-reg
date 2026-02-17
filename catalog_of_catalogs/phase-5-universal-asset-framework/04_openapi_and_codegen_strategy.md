# OpenAPI + Codegen Strategy for Universal Framework

## Goal
Make the system repeatable and automated by using:
- catalog.yaml as the primary plugin schema declaration
- deterministic generation of OpenAPI, persistence models, filter mappings
- additive vendor extensions to link schema to UX and actions

## Current baseline
- Each plugin owns OpenAPI and a merge step creates a unified catalog spec
- Shared schemas live in a common.yaml (BaseResource etc.)

## Phase 5 upgrades
1) Add common universal asset structures to common schemas
- metadata (labels, annotations, tags, owner, sourceRef)
- status (lifecycle, health, conditions, links)

2) Add vendor extensions for UI hints and actions
OpenAPI supports vendor extensions via x-* fields. citeturn0search13

3) catalog-gen generates:
- OpenAPI for entity types
- Capabilities skeleton (EntityCapabilities + filter fields + columns defaults)
- Action skeletons for baseline verbs (implemented by shared framework handlers)

## Vendor extensions (proposed)
- x-kubeflow-ui: displayName, order, section, widget, column hints
- x-kubeflow-actions: list of action ids exposed
- x-kubeflow-filter: allowedOperators, optimized flag

## Guardrails
- OpenAPI is descriptive, not prescriptive for UI layout
- Capabilities decide what UI actually renders
- Generated defaults must be editable without being overwritten

## Acceptance Criteria
- AC1: Merged OpenAPI remains valid and passes CI validation
- AC2: For a generated plugin, capabilities and OpenAPI are both produced from catalog.yaml with deterministic output
- AC3: UI hints and actions can be overridden by plugin authors without patching core UI/CLI code
- AC4: Model and MCP can adopt these extensions without changing existing public API paths

## Definition of Done
- catalog-gen outputs capability manifests and OpenAPI extensions
- merge scripts preserve vendor extensions
- CI validates generated code matches committed artifacts

## Verification plan
- Unit: generator deterministic tests
- Integration: generate a sample plugin, run merge, validate spec
- E2E: add knowledge source plugin via generator and confirm UI/CLI discover it
