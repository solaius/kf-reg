# Phase 9: Extensibility and Ecosystem

## Goal

Make adding new asset types boring, repeatable, and well-governed.

A different team should be able to:
- scaffold a new plugin
- implement providers
- pass conformance
- publish it in a supported/consumable form
- integrate it into UI + CLI with minimal coordination

This phase turns “plugin architecture exists” into “ecosystem scale is sustainable”.

## What ships

1. **catalog-gen vNext**
   - plugin scaffolding improvements
   - provider scaffolding improvements
   - generated conformance tests + docs
   - deterministic outputs, “regen safe” behavior

2. **Plugin conformance suite**
   - a standard test harness
   - a “must pass to be supported” bar
   - compatibility checks across server versions

3. **Plugin packaging + distribution**
   - built-in vs optional plugins
   - version compatibility rules and a compatibility matrix
   - a repeatable “server builder” path for optional plugins

4. **UI hints spec**
   - field display types and layout hints
   - search/facet hints
   - default filters and list/detail view composition
   - drives UI and CLI without bespoke code

5. **Documentation kit**
   - how to build a plugin
   - how to build a provider
   - how to publish and get “supported” status

## Non-goals (Phase 9)

- New asset types themselves (that’s Phase 6+)
- Runtime loading of Go plugins via `plugin` package (avoided due to portability/versioning constraints)
- Full registry/deployment lifecycling (catalog-level concerns only)

## Key design choices

- **Contract-first**: capabilities + UI hints are the contract. UI/CLI are readers.
- **Build-time plugin integration**: Go blank-import registration is compile-time; “optional plugins” are solved with repeatable build pipelines, not runtime code loading.
- **Conformance as a gate**: “supported plugin” is an objective bar, not tribal knowledge.
- **Governance is lightweight but real**: versioning, compatibility, security checks, docs, and ownership.

## Exit criteria

- A new plugin can be created by a different team and integrated with minimal coordination:
  - they scaffold with catalog-gen vNext
  - implement providers
  - pass conformance locally + in CI
  - publish artifacts
  - platform team integrates by updating a single manifest (or pulling a published server image)

## Documents

- 01 catalog-gen vNext requirements
- 02 conformance suite specification
- 03 packaging and distribution strategy
- 04 UI hints specification
- 05 documentation kit spec
- 06 governance and “supported plugin” program
- 07 verification plan and acceptance matrix
- 08 Claude Code execution guide

## References (external)

Links are included as code blocks to keep this pack self-contained.

```text
OpenAPI vendor extensions (x-*) are explicitly supported:
https://swagger.io/docs/specification/v3_0/openapi-extensions/

Kubeflow Central Dashboard customization and integration patterns:
https://www.kubeflow.org/docs/components/central-dash/customize/
https://www.kubeflow.org/docs/components/central-dash/overview/

Kubeflow Profiles (namespace isolation model):
https://www.kubeflow.org/docs/components/central-dash/profiles/

Model Catalog (existing baseline behavior):
https://www.kubeflow.org/docs/components/model-registry/reference/model-catalog-rest-api/

Kubernetes kubectl plugin packaging via Krew (analogy for “supported plugin index” + distro rules):
https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
https://krew.sigs.k8s.io/docs/developer-guide/distributing-with-krew/

UI schema patterns (useful for defining plugin-driven UI hints):
https://rjsf-team.github.io/react-jsonschema-form/docs/api-reference/uiSchema/
https://jsonforms.io/docs/uischema/

Artifact signing and provenance in OCI registries (optional hooks):
https://github.com/sigstore/cosign
https://docs.sigstore.dev/cosign/signing/other_types/
https://oras.land/blog/oras-artifacts-draft-specification-release/
```

