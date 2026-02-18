# 01 catalog-gen vNext: scaffolding + guardrails

## Objective

Upgrade catalog-gen from “generates boilerplate” to “creates a complete, consistent, supported plugin project”:
- scaffold plugin + providers + tests + docs
- encode compatibility metadata
- produce deterministic outputs
- enforce conventions automatically

This phase assumes the plugin framework already exists (init() registration, sources.yaml config, openapi merge).

## Inputs and outputs

### Inputs

- `catalog.yaml` (schema)
- `plugin.yaml` (new): plugin metadata + capabilities + UI hints + compatibility

Example `plugin.yaml` (conceptual):

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: agents
spec:
  displayName: Agents
  description: Catalog of agent definitions
  owners:
    - team: ai-platform
      contact: "#ai-platform"
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.x"
    frameworkApi: "v1alpha1"
  providers:
    - type: yaml
    - type: http
  ui:
    search:
      defaultQuery: "status = 'approved'"
      facets: ["owner", "riskLevel", "labels.env"]
```

### Outputs (generated)

- plugin skeleton (server, routes, repo, migrations)
- provider templates:
  - YAML provider (baseline)
  - HTTP provider (remote catalogs)
  - Git provider (catalog-as-code)
  - OCI provider (assets as artifacts) where applicable
- conformance test suite scaffold
- docs kit scaffold (README, provider guide, publish guide)
- OpenAPI with:
  - shared BaseResource composition
  - required pagination/filterQuery params
  - vendor extensions (x-*) for UI hints

## vNext commands

### `catalog-gen init`
Enhance init to generate:
- `plugin.yaml` populated with defaults
- conformance harness (tests + make targets)
- docs kit stubs

### `catalog-gen generate`
Must remain deterministic and “regen safe”:
- regenerate non-editable outputs
- never overwrite editable business logic
- verify openapi merge and conformance compilation

### `catalog-gen validate`
New: validates plugin meets baseline requirements:
- plugin.yaml fields present
- OpenAPI compiles
- UI hints schema validates
- conformance tests exist and are runnable
- compatibility fields follow semver rules

### `catalog-gen bump-version`
New: bumps plugin version and updates compatibility matrix entries.

## Definition of Done

- catalog-gen vNext supports `plugin.yaml`
- init produces a plugin that:
  - builds
  - runs
  - passes generated unit tests
  - passes conformance suite (when implementation stubs are filled)
- validate fails fast with actionable errors

## Acceptance Criteria

- A new team can run:
  - `catalog-gen init <name> ...`
  - implement a minimal provider
  - run `make conformance`
  - produce publish artifacts
  - with no changes required in UI/CLI repos

## Verification plan

- Golden tests for generator determinism
- “hello plugin” integration test in CI:
  - generate plugin
  - run unit tests + conformance
  - verify openapi merge includes plugin
