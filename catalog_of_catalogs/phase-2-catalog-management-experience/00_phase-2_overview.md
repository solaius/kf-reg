# phase-2-catalog-management-experience

Date: 2026-02-15

## Purpose

Phase 1 proved the plugin based catalog server can host multiple asset catalogs in a single process (model catalog as a plugin plus a generated MCP plugin), merge OpenAPI across plugins, and expose plugin discovery through the UI BFF.

Phase 2 turns that foundation into an end to end product experience that lets people actually operate and use the catalogs:

- AI Engineer: discover assets, evaluate fit, pull details, compare variants, and feed downstream workflows
- Ops for AI: configure catalog sources, control access, monitor health, and troubleshoot ingestion

This phase focuses on UI and CLI parity, consistent contracts, and operational readiness.

## North Star Outcome

A user can use either the UI or CLI to:

- Discover assets across all installed catalog plugins
- Manage catalog sources (add, update, enable, disable, validate)
- Monitor plugin health and ingestion status
- Troubleshoot errors with actionable diagnostics
- Extend the catalog with a new asset type using catalog-gen with minimal custom code

## Non Goals (Phase 2)

- Rebuilding any existing registry systems
- Designing a full control plane for model deployment or serving
- Adding speculative asset types without a concrete usage story
- Breaking compatibility for existing Model Catalog API paths

## Deliverables

By the end of Phase 2, we ship:

- UI: a unified catalog experience with plugin aware navigation, list and detail views, and source management flows
- CLI: a catalog CLI that can list plugins, list and filter assets, manage sources, and show health and status
- Backend: stable management endpoints for plugins and sources plus ingestion status and diagnostics
- Contracts: OpenAPI for catalog management and consistent schemas across plugins
- Docs: contributor docs for adding asset types, plus ops docs for configuring and operating catalogs

## Definition of Done

- UI and CLI support the same core user workflows
- New plugin can be scaffolded with catalog-gen and integrated end to end (API, BFF, UI, CLI) with a documented path
- Operational basics exist: readiness, health, status, logs, and common failure diagnostics
- Tests cover the critical paths for at least Model and MCP catalogs plus the management plane
