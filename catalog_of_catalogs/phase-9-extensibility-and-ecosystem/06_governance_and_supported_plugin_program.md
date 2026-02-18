# 06 Governance and the “supported plugin” program

## Objective

Define the lightweight governance needed so “supported plugin” means something:
- compatibility
- security
- ownership
- quality

## Supported plugin requirements

A plugin can call itself “supported” only if:

1. Conformance suite passes (latest released server version in range)
2. Compatibility metadata exists and is correct
3. Ownership exists:
   - named owning team
   - support channel/contact
4. Security checks pass:
   - license scan
   - vuln scan for published images
   - SBOM available
5. Docs kit complete

## Publishing and indexing

Create a “supported plugin index”:
- a Git repo with:
  - plugin.yaml manifests
  - compatibility matrix
  - latest conformance report link
  - image/module coordinates

This is conceptually similar to curated indices like Krew for kubectl plugins, but for our server-side plugins.

## Optional signing hooks

If we ship optional plugins as container images:
- provide hooks for signing and verification (cosign)
- keep it optional in Phase 9, but design the pipeline to support it

## Definition of Done

- Supported plugin checklist exists and is automated where possible
- CI gating prevents “supported” designation without conformance + checks
- Index repo format is defined and documented

## Acceptance Criteria

- Another team can publish a supported plugin without bespoke review cycles:
  - they run the pipeline
  - produce artifacts + reports
  - submit a single PR to the index

## Verification plan

- Run the full process with a toy plugin:
  - publish module + image
  - run checks
  - PR into index
