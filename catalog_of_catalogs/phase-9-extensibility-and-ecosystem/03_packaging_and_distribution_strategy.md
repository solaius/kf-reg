# 03 Plugin packaging and distribution strategy

## Objective

Define how plugins are delivered and kept compatible.

Constraints:
- Plugins register via blank imports (compile-time).
- We want “optional plugins” without turning integration into a bespoke effort each time.

## Packaging tiers

### Tier 1: Built-in plugins (in-repo)
- core, widely used assets
- shipped in the default catalog-server image
- versioned with catalog-server releases

### Tier 2: Optional supported plugins (out-of-repo, curated)
- built and shipped as:
  - separate Go modules
  - plus a published catalog-server image that includes them
- curated by conformance + governance rules
- optional plugins may have faster release cadence than core

### Tier 3: Experimental / community plugins
- no support guarantees
- still can run if built into a custom server image
- must not claim “supported” unless conformance/gov passes

## “Server builder” approach (recommended)

To make optional plugins easy:
- Maintain a `catalog-server-manifest.yaml` listing plugin modules and versions
- A builder pipeline generates a `main.go` with blank imports, then builds images

Example manifest (conceptual):

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogServerBuild
spec:
  base:
    image: catalog-server:0.9.0
    frameworkVersion: "v1alpha1"
  plugins:
    - name: agents
      module: github.com/acme/catalog-plugins/agents
      version: v0.3.1
    - name: guardrails
      module: github.com/acme/catalog-plugins/guardrails
      version: v0.2.0
```

Outputs:
- catalog-server image with those plugins compiled in
- SBOM and vulnerability scan results
- signed image (optional; see governance)

This is analogous to curated indices like Krew, but for server-side compiled plugins (not runtime).

## Compatibility rules

### Semver policy

- catalog-server follows semver
- the shared plugin framework API has a declared version (e.g., `frameworkApi: v1alpha1`)
- a plugin declares:
  - `minServerVersion`
  - `maxServerVersion` (or compatible range)
  - `frameworkApi` compatibility

### Break glass rules

If server introduces breaking plugin API changes:
- bump major version
- provide a migration guide
- keep a compatibility shim for core plugins when possible

## Definition of Done

- Clear tier model documented
- A server builder pipeline exists (CI-ready)
- Compatibility metadata required in plugin.yaml
- A compatibility matrix is generated automatically

## Acceptance Criteria

- Another team can ship an optional plugin by:
  - publishing module version
  - running builder pipeline
  - producing a signed image + conformance report
  - platform team only approves a manifest/index change

## Verification plan

- Demonstrate with a toy plugin in a separate repo:
  - publish module
  - build server image via manifest
  - run conformance + smoke tests
