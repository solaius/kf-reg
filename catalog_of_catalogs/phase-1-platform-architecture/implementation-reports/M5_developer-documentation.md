# M5: Developer Documentation

**Date**: 2026-02-15
**Status**: Complete
**Phase**: Phase 1: Platform Architecture

## Summary

This milestone delivers three developer-facing documentation guides that explain how to create catalog plugins, integrate them with UI/CLI clients, and use the `catalog-gen` scaffolding tool. Together they provide the complete onboarding path for developers adding new catalog types to the system.

## Motivation

- New contributors need a clear, end-to-end guide for creating plugins without reading every source file in the codebase.
- Frontend and CLI developers need to understand the plugin discovery API contract so they can build dynamic, plugin-aware interfaces.
- The `catalog-gen` tool has multiple commands and a nuanced editable-vs-regenerated file model that must be documented to prevent accidental data loss.
- **AC6** (Docs for adding plugins + UI/CLI wiring): Requires documentation covering the full developer workflow from plugin creation through frontend integration.
- **FR11** (Developer ergonomics): The documentation must make it straightforward for a developer to add a new catalog type in under a day.

## What Changed

### Files Created

| File | Purpose |
|------|---------|
| `docs/plugin-developer-guide.md` | Step-by-step walkthrough for creating a new catalog plugin using `catalog-gen` |
| `docs/generic-ui-cli-integration.md` | Guide for frontend and CLI integration via the plugin discovery API |
| `docs/catalog-gen-guide.md` | Reference documentation for all `catalog-gen` commands, flags, and configuration |

### Files Modified

None. This milestone is documentation-only.

## How It Works

### Plugin Developer Guide

Located at `docs/plugin-developer-guide.md`, this guide walks through the eight steps to create a plugin, using the MCP plugin as the reference example:

1. **Scaffold** -- Run `catalog-gen init` with entity name and Go package path
2. **Edit `catalog.yaml`** -- Define entity properties, providers, and API settings
3. **Regenerate code** -- Run `catalog-gen generate` and `make gen/openapi-server`
4. **Implement service layer** -- Write list/get logic in `api_*_service_impl.go`
5. **Implement YAML provider** -- Parse entity data from YAML sources
6. **Create test data** -- Populate `testdata/` with sample YAML files
7. **Wire into catalog-server** -- Add blank import in `cmd/catalog-server/main.go`
8. **Build and test** -- Compile and verify with curl

The guide includes the full `catalog.yaml` for the MCP plugin:

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogConfig
metadata:
  name: mcp
spec:
  package: github.com/kubeflow/model-registry/catalog/plugins/mcp
  entity:
    name: McpServer
    properties:
      - name: serverUrl
        type: string
        required: true
        description: URL of the MCP server
      - name: transportType
        type: string
      - name: toolCount
        type: integer
      - name: resourceCount
        type: integer
      - name: promptCount
        type: integer
  providers:
    - type: yaml
  api:
    basePath: /api/mcp_catalog/v1alpha1
    port: 8081
```

It also documents the file reference table showing which files are editable vs regenerated, property type mappings, and procedures for adding properties and artifact types after initial scaffolding.

### Generic UI/CLI Integration Guide

Located at `docs/generic-ui-cli-integration.md`, this guide documents the plugin discovery API and how clients consume it:

**Discovery endpoint** -- `GET /api/plugins` returns metadata about all registered plugins:

```json
{
  "plugins": [
    {
      "name": "mcp",
      "version": "v1alpha1",
      "basePath": "/api/mcp_catalog/v1alpha1",
      "healthy": true,
      "entityKinds": ["McpServer"],
      "capabilities": {
        "listEntities": true,
        "getEntity": true,
        "listSources": true,
        "artifacts": false
      }
    }
  ],
  "count": 1
}
```

**BFF proxy** -- The frontend accesses this via `GET /api/v1/model_catalog/plugins`, which follows the same middleware chain as other catalog routes.

**Frontend extension points** -- The guide provides a four-step pattern for dynamic UI integration:

1. Discover plugins at startup
2. Build navigation dynamically from plugin metadata
3. Construct API URLs using `basePath`
4. Check `capabilities` to conditionally render UI components

It includes a TypeScript example showing how to fetch plugins and render cards:

```typescript
const response = await fetch('/api/v1/model_catalog/plugins?namespace=kubeflow');
const { data } = await response.json();

data.plugins.forEach(plugin => {
  if (plugin.healthy) {
    renderPluginCard({
      title: plugin.description,
      basePath: plugin.basePath,
      entityKinds: plugin.entityKinds,
      hasArtifacts: plugin.capabilities?.artifacts ?? false,
    });
  }
});
```

**CLI integration** -- Shows shell patterns for discovering plugins, listing entities, and using `filterQuery` for structured queries.

**Health monitoring** -- Documents how `/api/plugins` serves as a per-plugin health check, with `healthy: false` and `status.lastError` for failed plugins.

### catalog-gen Guide

Located at `docs/catalog-gen-guide.md`, this is the reference manual for the code generation tool. It documents:

**Commands:**

| Command | Purpose |
|---------|---------|
| `catalog-gen init <name>` | Scaffold a new plugin with all directories and files |
| `catalog-gen generate` | Regenerate code from `catalog.yaml` |
| `catalog-gen add-artifact <Name>` | Add a new artifact type to the plugin |
| `catalog-gen add-provider <type>` | Add a data provider (yaml or http) |
| `catalog-gen gen-testdata` | Generate sample test data files |

**Editable vs non-editable file model** -- Clearly separates files that are safe to edit (service layer, providers, API impl, `catalog.yaml`) from files that are overwritten on every `generate` run (plugin.go, register.go, models, specs, loader, OpenAPI spec).

**`catalog.yaml` configuration** -- Documents the full schema including `apiVersion`, `metadata`, `spec.entity.properties`, `spec.artifacts`, `spec.providers`, and `spec.api` with base path and port.

**Typical workflow** -- An eight-step copy-pasteable workflow from plugin creation through adding properties and artifacts.

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Three separate docs instead of one monolith | Each doc serves a different audience (plugin author, frontend dev, tool user) and can be linked independently | Single guide (rejected -- too long, mixes concerns) |
| Use MCP plugin as the running example | It is a real, complete, non-trivial plugin that exercises all features (properties, providers, filtering) | Hypothetical example (rejected -- harder to verify accuracy) |
| Include TypeScript and shell examples in UI/CLI guide | Reduces time-to-first-integration for frontend and CLI developers | Reference-only API docs (rejected -- too abstract for practical use) |
| Document editable vs regenerated distinction prominently | This is the most common source of confusion -- developers losing custom code by running `generate` | Footnote or FAQ (rejected -- too important to bury) |

## Testing

- Documentation accuracy was verified by cross-referencing every code snippet and file path against the actual source code.
- The `catalog-gen` commands documented in the guide match the implemented CLI interface.
- API response formats match the `CatalogPlugin` and `CatalogPluginList` structs in `clients/ui/bff/internal/models/catalog_plugin.go`.

## Verification

```bash
# Verify all three docs exist
ls docs/plugin-developer-guide.md docs/generic-ui-cli-integration.md docs/catalog-gen-guide.md

# Verify the catalog-gen commands referenced in docs are available
go build -o catalog-gen ./cmd/catalog-gen
./catalog-gen --help
./catalog-gen init --help
./catalog-gen generate --help
./catalog-gen add-artifact --help
./catalog-gen add-provider --help
./catalog-gen gen-testdata --help

# Verify the plugin discovery endpoint documented in the UI/CLI guide
# (requires running catalog server or BFF in mock mode)
cd clients/ui/bff && make dev-bff &
curl -s http://localhost:4000/api/v1/model_catalog/plugins?namespace=kubeflow | jq .plugins[].name
# Expected output: "model" and "mcp"
```

## Dependencies & Impact

- **Upstream**: Depends on M1 (plugin framework), M2 (catalog-gen tool), M3 (OpenAPI merge), and M4 (BFF plugin discovery) being complete, since the documentation references all of these.
- **Downstream**: Enables external contributors and partner teams to create new catalog plugins without deep codebase knowledge. Satisfies the AC6 acceptance criterion for developer documentation.
- **Backward compatibility**: Documentation-only change; no code impact.

## Open Items

- No automated link-checking or doc-testing harness to verify code snippets stay in sync with source code over time.
- The UI/CLI integration guide describes frontend extension points conceptually but does not include a working React component example (deferred to a future frontend milestone).
- The `catalog-gen` guide does not yet document error messages and troubleshooting for common failures (e.g., missing `catalog.yaml`, invalid property types).
