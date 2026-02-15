# Suggested workplan and milestones

This is a suggested sequencing guide, not a hard requirement.

## Milestone 1: Baseline plugin hosting hardening
- Ensure plugin lifecycle and config loading are stable
- /api/plugins endpoint returns useful metadata
- Model plugin is wrapped with zero behavior change
Outputs
- green CI
- docs for enabling plugins via sources.yaml

## Milestone 2: First non-model plugin end-to-end
- Scaffold or implement an asset-type plugin
- YAML ingestion
- list and get endpoints
Outputs
- integration tests
- example test data

## Milestone 3: Unified OpenAPI merge and validation
- Deterministic merged spec includes all plugin specs
- CI check mode enforced
Outputs
- docs for adding plugin OpenAPI

## Milestone 4: Generic UI and CLI enablement seams
- Document and implement minimum contract for generic rendering
- CLI can list plugins and basic entities
Outputs
- demo-level UX and CLI commands

## Milestone 5: Developer ergonomics
- catalog-gen or equivalent workflow is usable and documented
- Regeneration behavior and editable vs non-editable separation is proven
Outputs
- guide and example plugin

