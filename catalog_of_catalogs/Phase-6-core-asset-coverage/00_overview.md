# Phase 6: Core Asset Coverage

## Goal
Land real catalog plugins and real providers for the major AI asset categories, so the system covers end-to-end workflows (discover, filter, manage actions) in both UI and CLI, and is extensible for future assets.

This phase assumes Phase 5 has delivered:
- Universal asset contract (shared fields and conventions)
- Universal action model and a consistent management API surface
- Plugin capabilities schema and discovery endpoints that drive UI and CLI automatically
- Generic UI components and CLI v2 that are plugin-driven
- Conformance suite for plugins

## Exit criteria
- At least 4 additional asset catalogs can be loaded from real sources, filtered, and managed from UI and CLI
- Model, MCP, and Knowledge Sources schemas are confirmed and updated where required to align with the Phase 5 universal contract and action model
- Provider ecosystem is expanded beyond YAML with at least:
  - HTTP provider usable by multiple plugins
  - Git provider (catalog-as-code) usable by multiple plugins
  - OCI provider usable for at least one plugin where "assets as artifacts" is the best fit

## Suggested delivery order
1) Prompt Templates
2) Agents Catalog (highest priority)
3) Guardrails
4) Policies
5) Datasets
6) Evaluators
7) Benchmarking
8) Notebooks
9) Skills (commands)

Rationale: Prompt Templates and Agents unlock the most visible end-to-end UX and become reference patterns for the remaining plugins.

## Non-goals (Phase 6)
- Building or replacing a full downstream "registry" for each asset type
- Implementing deployment or runtime execution of assets (beyond management actions like validate/apply/enable/disable/refresh and the universal asset actions defined in Phase 5)
- Solving organization-wide governance and policy enforcement beyond schema, validation, and metadata capture

## Dependencies
- Plugin-based catalog-server and catalog-gen scaffolding are available
- Capabilities schema and universal action model endpoints exist and are stable
- UI and CLI can render plugin-defined assets and actions using the capabilities schema
- A containerized dev environment for repeatable e2e validation exists (Phase 4 baseline)

## Source references used to guide schema design
This phase intentionally leans on widely adopted specs and de facto standards for schema fidelity:
- Kubeflow Model Catalog is a federated metadata aggregation layer for discovery across sources (Kubeflow docs)
- JSON Schema 2020-12 for embedded input/output schemas (JSON Schema, OpenAPI 3.1, MCP tool schemas)
- Dataset metadata standards: MLCommons Croissant, Hugging Face dataset cards, schema.org Dataset, Datasheets for Datasets
- Policy-as-code: Open Policy Agent (Rego and bundles)
- Notebooks: Jupyter nbformat
- Tools and skills schemas: MCP tool definitions (JSON Schema-based)

See 04_common-schema-standards.md for the consolidated reference list.
