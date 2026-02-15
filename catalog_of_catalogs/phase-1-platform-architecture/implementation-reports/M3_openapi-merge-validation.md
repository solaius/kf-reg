# M3: OpenAPI Merge and Validation

**Date**: 2026-02-15
**Status**: Complete
**Phase**: Phase 1: Platform Architecture

## Summary

This milestone delivers a deterministic OpenAPI merge pipeline that combines the main catalog spec with all plugin specs into a single unified specification (`catalog-spec.yaml`). A key bug fix prevents common schemas from plugin specs from corrupting the main catalog's schema definitions during deep merge.

## Motivation

- The catalog server exposes a unified API surface that spans multiple plugins (e.g., model catalog, MCP catalog). Frontends and documentation tooling need a single, authoritative OpenAPI spec.
- Without namespace isolation, plugin schemas with the same name as common schemas (e.g., `BaseResource`, `BaseResourceList`) would overwrite the main catalog's richer definitions during `yq` deep merge.
- **FR9** (Unified OpenAPI): Requires a deterministic merge producing no collisions, with CI validation.
- **AC4** (Merged spec includes both plugins): The merged output must contain paths and schemas from every registered plugin.

## What Changed

### Files Created

| File | Purpose |
|------|---------|
| `scripts/merge_catalog_specs.sh` | Shell script to discover, preprocess, and merge plugin OpenAPI specs into a single unified spec |
| `api/openapi/catalog-spec.yaml` | The merged output consumed by documentation and frontend tooling |

### Files Modified

| File | Change |
|------|--------|
| `catalog/plugins/mcp/api/openapi/openapi.yaml` | Plugin spec used as merge input; schemas include common `BaseResource` definitions |
| `catalog/plugins/mcp/api/openapi/src/openapi.yaml` | Source spec for MCP plugin paths (pre-component merge) |
| `catalog/plugins/mcp/api/openapi/src/generated/components.yaml` | Generated schemas from `catalog-gen`, including common schemas that triggered the merge bug |

## How It Works

### Common Schema Detection

The script loads schema names from the main catalog's shared library file (`api/openapi/src/lib/common.yaml`) and builds a pipe-delimited lookup string. This list is used throughout preprocessing to distinguish common schemas from plugin-specific ones.

```bash
COMMON_SCHEMAS=""
if [[ -f "api/openapi/src/lib/common.yaml" ]]; then
    COMMON_SCHEMAS=$($YQ eval '.components.schemas | keys | join("|")' api/openapi/src/lib/common.yaml 2>/dev/null || echo "")
fi

is_common_schema() {
    local name="$1"
    [[ -n "$COMMON_SCHEMAS" && "|${COMMON_SCHEMAS}|" == *"|${name}|"* ]]
}
```

The common schemas include `BaseModel`, `BaseResource`, `BaseResourceDates`, `BaseResourceList`, `Error`, `MetadataValue`, `SortOrder`, and all `Metadata*Value` types.

### Plugin Preprocessing (Namespace Isolation)

Before merging, each plugin spec is preprocessed to avoid naming collisions. The `preprocess_plugin_spec` function performs seven steps:

1. **Prefix plugin-specific schemas** -- Renames e.g. `McpServer` to `Mcp_McpServer`, while leaving common schemas untouched.
2. **Remove common schemas from the plugin spec** -- This is the critical bug fix. Common schemas like `BaseResource` in the plugin have simpler definitions than the main catalog. Without removal, the deep merge would overwrite the main catalog's `BaseResource` with the plugin's shallow version.
3. **Update `$ref` pointers** -- Rewrites all `$ref` references to use the prefixed names, then un-prefixes references to common schemas.
4. **Resolve external references** -- Converts `lib/common.yaml#/components/schemas/...` to local `#/components/schemas/...`.
5. **Prefix operation IDs** -- e.g., `listMcpServers` becomes `mcp_listMcpServers`.
6. **Convert relative paths to absolute** -- Combines the plugin's `servers[0].url` with each path, e.g., `/mcpservers` becomes `/api/mcp_catalog/v1alpha1/mcpservers`.
7. **Remove info section** -- Prevents the plugin's info block from overwriting the main catalog's title and description.

```bash
# Step 2: Remove common schemas to avoid corrupting main catalog definitions
if [[ -n "$COMMON_SCHEMAS" ]]; then
    IFS='|' read -ra COMMON_ARRAY <<< "$COMMON_SCHEMAS"
    local del_expr=""
    for schema in "${COMMON_ARRAY[@]}"; do
        if $YQ eval ".components.schemas[\"${schema}\"]" "$temp_file" 2>/dev/null | grep -q -v '^null$'; then
            if [[ -n "$del_expr" ]]; then
                del_expr="${del_expr} | "
            fi
            del_expr="${del_expr}del(.components.schemas[\"${schema}\"])"
        fi
    done
    if [[ -n "$del_expr" ]]; then
        $YQ eval -i "$del_expr" "$temp_file" 2>/dev/null || true
    fi
fi
```

### Deep Merge and Final Ordering

After preprocessing, each plugin is merged into the accumulating output using `yq eval-all`:

```bash
$YQ eval-all '. as $item ireduce ({}; . * $item)' "$OUT_FILE" "$temp_preprocessed" > "$temp_merged"
```

After all plugins are merged, the script re-orders top-level keys and sorts paths, schemas, responses, and parameters alphabetically for deterministic output.

### CI Validation Mode

The `--check` flag generates the merged spec into a temporary file and diffs it against the committed version, failing if they diverge:

```bash
if [[ "$CHECK" == "true" ]]; then
    diff -u "api/openapi/$BASENAME" "$OUT_FILE"
    exit $?
fi
```

### Merged Output

The resulting `catalog-spec.yaml` contains both main catalog and plugin paths with proper namespacing:

```yaml
paths:
  /api/mcp_catalog/v1alpha1/mcpservers:
    get:
      operationId: mcp_listMcpServers
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Mcp_McpServer List'
  /api/model_catalog/v1alpha1/labels:
    # ... main catalog paths ...
```

Plugin schemas appear with their prefix (`Mcp_McpServer`, `Mcp_McpServerList`) while common schemas (`BaseResource`, `BaseResourceList`) appear once with the main catalog's full definitions.

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Remove common schemas from plugin before merge | Deep merge (`*`) replaces maps key-by-key; the plugin's simpler `BaseResource` would overwrite the main catalog's `allOf`-composed version | Tried prefixing all schemas including common ones, but that broke `$ref` chains across plugins |
| Prefix plugin schemas with `Mcp_` | Avoids name collisions when multiple plugins define types with the same name | Per-plugin sub-namespaces in components (rejected -- OpenAPI 3.0 does not support nested schema groups) |
| Auto-discover plugins via `find` | New plugins are included automatically without editing the script | Hardcoded list (rejected -- violates open/closed principle) |
| `--check` mode for CI | Ensures committed spec is always up to date | Git hooks (too intrusive), separate validation script (redundant) |

## Testing

- **Manual verification**: Run `scripts/merge_catalog_specs.sh catalog-spec.yaml` and inspect the output.
- **CI check mode**: `scripts/merge_catalog_specs.sh --check catalog-spec.yaml` exits non-zero if output differs from committed spec.
- **Schema validation**: The merged spec can be validated with any OpenAPI 3.0 linter (e.g., `spectral lint api/openapi/catalog-spec.yaml`).

## Verification

```bash
# Generate the merged spec
scripts/merge_catalog_specs.sh catalog-spec.yaml

# Verify no drift from committed version
scripts/merge_catalog_specs.sh --check catalog-spec.yaml

# Confirm plugin schemas are prefixed
grep 'Mcp_McpServer' api/openapi/catalog-spec.yaml

# Confirm common schemas are NOT duplicated with plugin prefix
grep -c 'Mcp_BaseResource' api/openapi/catalog-spec.yaml  # should be 0

# Confirm BaseResource retains main catalog's full definition (allOf)
yq '.components.schemas.BaseResource' api/openapi/catalog-spec.yaml

# Confirm x-catalog-plugins extension lists the merged plugins
yq '.["x-catalog-plugins"]' api/openapi/catalog-spec.yaml
```

## Dependencies & Impact

- **Upstream**: Depends on M1 (plugin framework) and M2 (catalog-gen) having generated `catalog/plugins/mcp/api/openapi/openapi.yaml`.
- **Downstream**: Enables M4 (BFF plugin discovery) and M5 (developer documentation) to reference a single authoritative API contract. Frontend and CLI tooling consume this merged spec for auto-generation.
- **Backward compatibility**: The merge script is additive. Removing a plugin simply removes its paths and schemas from the next merge run.

## Open Items

- No automated OpenAPI 3.0 structural validation step in CI yet (only diff-based check).
- If two plugins define a schema with the same name (after removing the plugin prefix), the second merge would overwrite the first. This is unlikely with the current naming convention but not enforced.
- The script depends on `yq` v4+ being available as `bin/yq` or via the `$YQ` environment variable.
