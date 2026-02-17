# Knowledge Sources Plugin (Exit Criteria Plugin)

## Purpose
Prove Phase 5 exit criteria:
A brand-new plugin appears in UI and CLI with zero UI/CLI code changes.

## Scope
Minimal but real plugin managing KnowledgeSource assets:
- documents (files/urls)
- vector stores (connections)
- graph stores (connections)

Phase 5 does not require ingestion; it requires list/get + actions + discoverability.

## Entity definition (example)
Kind: KnowledgeSource
Suggested fields:
- spec.type: enum [document, url, vector_store, graph_store]
- spec.location: string (path or URL)
- spec.contentType: string (optional)
- metadata.tags
- status.health

## Providers
- YAML provider (required)
- HTTP provider (optional)

## Actions
- asset: tag, annotate, deprecate, link
- source: refresh

## Acceptance Criteria
- AC1: plugin created using catalog-gen and minimal custom code
- AC2: it appears in UI nav and has working list/detail screens
- AC3: CLI v2 can list/get and run at least one action
- AC4: conformance test suite passes

## Definition of Done
- Plugin scaffolded, registered, configured in sources.yaml
- Capabilities generated/overridden as needed
- Data loads from sample files and persists
- UI + CLI flows validated
