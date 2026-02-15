# Contract and API Strategy

This project must remain contract first.

## Principles

- OpenAPI is the source of truth for API surface
- Contracts are versioned and validated in CI
- Code generation is used where it reduces drift
- Backward compatibility for existing Model Catalog endpoints is preserved

## Required contracts

1. Catalog plugin discovery contract

- List plugins
- Plugin metadata
- Plugin health and capability hints

2. Source management contract

- List sources
- Validate proposed changes
- Apply changes (create, update, enable, disable, delete) where supported
- Trigger refresh and read refresh status

3. Diagnostics contract

- Structured status for plugin and sources
- Structured errors

## How to design the contracts

- Prefer additive changes
- Avoid per plugin bespoke management endpoints where possible
- Use shared schemas in a common module and reference them from plugin specs
- Treat plugin capabilities as discoverable metadata so UI and CLI can adapt without redeploy

## OpenAPI generation and merge

- Keep plugin owned OpenAPI specs for plugin specific entity APIs
- Keep a shared OpenAPI module for
  - base resource schemas
  - management plane schemas
- Merge into a single unified spec for documentation and client generation

## Required quality gates

- OpenAPI validation in CI
- Generated files in sync check in CI
- Linting and formatting checks in CI

## Output of this phase

- A unified spec that includes both
  - plugin entity APIs
  - management plane APIs
- Generated client libraries for UI BFF and for CLI

## Notes for implementers

- Follow the conventions and constraints in PROGRAMMING_GUIDELINES.md
- Avoid leaking internal database details into the API
- Make pagination and filtering consistent across all plugins
