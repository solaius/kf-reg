# Goals, scope, and non-goals

## Goals
- Provide a unified catalog-server capable of hosting multiple asset-type catalogs in one runtime
- Ensure zero breaking changes for current Model Catalog API consumers
- Make it easy to add new asset types using a consistent plugin contract
- Provide consistent query semantics across asset types
  - filterQuery
  - pagination and ordering
  - standard resource envelopes and metadata
- Enable a unified OpenAPI contract for documentation and client generation
- Support multiple data source types per plugin (at minimum YAML, with a clear path to HTTP or registry-backed providers)
- Provide clear integration seams for UI and CLI consumers

## Scope for this project phase
- Implement and harden plugin architecture within catalog-server
- Validate at least one non-model plugin end-to-end
- Provide docs and examples showing how to add a new plugin
- Provide baseline UI and CLI integration guidance and contract expectations

## Explicit non-goals
- Building the full set of asset-type plugins listed in the motivation
- Implementing full lifecycle governance workflows for every asset type
- Designing a brand new filtering language or pagination scheme
- Solving every cross-asset relationship in v1
- Designing a new authentication and authorization model beyond what the platform already provides
  - The system should integrate with existing authn and authz patterns in the repo

## Compatibility constraints
- Existing Model Catalog routes, schemas, and behaviors must remain unchanged
- Configuration for existing model catalogs should continue to work, including legacy naming expectations where applicable
- The merged OpenAPI spec must remain additive and stable

