# catalogctl CLI and Conformance Suite

## Overview

This document covers two client-side tools that complete the universal asset
framework's developer and operator experience:

1. **catalogctl** -- A Cobra-based CLI that dynamically discovers catalog
   plugins at startup and builds per-plugin command trees at runtime, so that
   new asset-type plugins appear in the CLI with zero code changes.

2. **Conformance Suite** -- An integration test suite that validates every
   loaded plugin meets the Phase 5 universal framework contract. Tests run
   against a live catalog-server and cover capabilities, endpoints, actions,
   and filters.

Both tools rely on the same V2 capabilities schema that drives the generic
UI, ensuring consistency across all three consumer surfaces (UI, CLI, tests).

```
                  catalog-server (:8080)
                         |
         +---------------+---------------+
         |               |               |
         v               v               v
   Generic UI       catalogctl      Conformance
   (BFF + React)    (Cobra CLI)     Suite (go test)
         |               |               |
         +-------+-------+-------+-------+
                 |               |
                 v               v
        GET /api/plugins   GET /api/plugins/{name}/capabilities
        (discovery)        (V2 schema per plugin)
```

## catalogctl Overview

**Location:** `cmd/catalogctl/`

catalogctl is a capabilities-driven CLI for the unified catalog server. On
startup it calls `GET /api/plugins` to discover all registered plugins and
their V2 capabilities. For each plugin that exposes `capabilitiesV2`, it
dynamically generates Cobra subcommands for entity listing, detail retrieval,
action execution, source management, and action discovery.

Static commands (`plugins`, `health`) are always available. Dynamic commands
only appear when the server is reachable at startup; if the server cannot be
contacted, the CLI prints a warning and continues with static commands only.

```go
// cmd/catalogctl/main.go (simplified)
func main() {
    if err := discoverPlugins(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not discover plugins: %v\n", err)
    }
    rootCmd.Execute()
}
```

### Global Flags

| Flag       | Default                  | Description                        |
|------------|--------------------------|------------------------------------|
| `--server` | `http://localhost:8080`  | Catalog server URL                 |
| `--output` | `table`                  | Output format: `table`, `json`, `yaml` |

## Static Commands

### `catalogctl plugins`

Lists all registered plugins with their health, version, entity kinds, and
description. Calls `GET /api/plugins` and renders a table:

```
NAME     VERSION     HEALTHY  ENTITIES                DESCRIPTION
model    v1alpha1    yes      [CatalogModel]          Model catalog for ML models
mcp      v1alpha1    yes      [McpServer]             McpServer catalog
agents   v1alpha1    yes      [Agent]                 Agent catalog
```

If a plugin is unhealthy, the `HEALTHY` column shows the last error:

```
knowledge  v1alpha1  no (source timeout)  [KnowledgeSource]  Knowledge catalog
```

### `catalogctl health`

Checks server liveness and readiness by calling `/healthz` and `/readyz`:

```
CHECK       STATUS
Liveness    alive
Uptime      5m32s
Readiness   ready
```

## Dynamic Discovery

At startup `discoverPlugins()` fetches the plugin list and iterates:

```
                  GET /api/plugins
                        |
                        v
               +------------------+
               | pluginsResponse  |
               | {plugins: [...]} |
               +------------------+
                        |
                for each plugin
                        |
              Has capabilitiesV2?
              /                \
            yes                no
             |                  |
             v                 skip
    buildPluginCommand(p)
             |
    +--------+--------+-------+
    |        |        |       |
  entity   entity  sources  actions
  cmds     cmds    cmd      cmd
```

For each plugin with V2 capabilities, the following command tree is generated:

1. **Entity subcommands** -- one per entry in `capabilitiesV2.entities[]`
   - `{plural} list` -- list entities with pagination and filtering
   - `{plural} get <name>` -- get a single entity by name
   - `{plural} {actionID} <name>` -- one subcommand per action the entity supports

2. **Sources subcommand** (if `capabilitiesV2.sources` is non-nil)
   - `sources list` -- list configured data sources
   - `sources refresh [source-id]` -- refresh a single source or all sources (if `refreshable`)

3. **Actions subcommand** (if `capabilitiesV2.actions` is non-empty)
   - `actions` -- list all available actions with metadata

## Generated Command Structure

```
catalogctl
|-- plugins                         # Static: list registered plugins
|-- health                          # Static: server health check
|-- model                           # (dynamic) Model plugin
|   |-- models list                 # List catalog models
|   |-- models get <name>           # Get model detail
|   |-- models tag <name>           # Execute tag action
|   |-- models annotate <name>      # Execute annotate action
|   |-- sources list                # List data sources
|   |-- sources refresh [id]        # Refresh source(s)
|   +-- actions                     # List available actions
|-- mcp                             # (dynamic) MCP plugin
|   |-- mcpservers list             # List MCP servers
|   |-- mcpservers get <name>       # Get MCP server detail
|   |-- mcpservers tag <name>       # Execute tag action
|   |-- sources list                # List data sources
|   |-- sources refresh [id]        # Refresh source(s)
|   +-- actions                     # List available actions
|-- agents                          # (dynamic) Agents plugin
|   |-- agents list
|   |-- agents get <name>
|   +-- ...
|-- knowledge                       # (dynamic) Knowledge plugin
|   |-- knowledgesources list
|   |-- knowledgesources get <name>
|   +-- ...
+-- ...                             # One group per discovered plugin
```

## Output Formats

The `--output` (`-o`) flag controls output rendering globally:

| Format  | Behavior |
|---------|----------|
| `table` | Default. Prints uppercase headers and aligned columns via `text/tabwriter`. For list views, uses V2 `columnHint` definitions for headers and cell extraction. For detail views, groups fields by `section`. |
| `json`  | Pretty-printed JSON via `json.Encoder` with two-space indent. |
| `yaml`  | YAML via `gopkg.in/yaml.v3` (marshalled through JSON first for consistent key names). |

## Dynamic Table Rendering

Entity list tables are driven entirely by V2 `columnHint` definitions from
the capabilities response. This is the mechanism that lets new plugins appear
in the CLI without code changes.

Each `columnHint` provides:

```go
// cmd/catalogctl/types.go
type columnHint struct {
    Name        string `json:"name"`        // Internal name
    DisplayName string `json:"displayName"` // Table header text
    Path        string `json:"path"`        // Dot-separated JSON path
    Type        string `json:"type"`        // Value type (string, number, etc.)
    Sortable    bool   `json:"sortable"`    // Whether orderBy is supported
}
```

The `extractValue()` function in `discover.go` uses the `Path` field as a
dot-separated JSON path to extract values from each entity response:

```
Column Path: "status.state"           Column Path: "name"
                |                                    |
                v                                    v
    data["status"]["state"] -> "available"    data["name"] -> "llama3"
```

Supported value types in `extractValue`:

| Go Runtime Type | Rendering |
|-----------------|-----------|
| `string`        | Literal value |
| `float64` (integer) | Formatted as integer (e.g., `42`) |
| `float64` (decimal) | Formatted to 2 decimal places (e.g., `3.14`) |
| `bool`          | `"true"` or `"false"` |
| `[]any`         | Comma-separated (e.g., `"a, b, c"`) |
| `nil`           | Empty string |

When a plugin provides no column hints (fallback), `inferColumns()` creates
columns from up to 5 keys of the first item in the response.

### Detail View Rendering

The `get` subcommand uses `detailFields` from V2 capabilities, grouping
fields by the `section` attribute and rendering labeled key-value pairs:

```
--- Overview ---
  Name:                 llama3
  Description:          Meta's LLaMA 3 model
  Provider:             Meta

--- Technical ---
  Task:                 text-generation
  License:              Apache-2.0
```

## Entity List Flags

The `list` subcommand on every entity supports:

| Flag                | Type   | Description |
|---------------------|--------|-------------|
| `--filter`          | string | Filter query (e.g., `"name LIKE '%server%'"`) |
| `--order-by`        | string | Field to order results by |
| `--sort`            | string | Sort order: `ASC` or `DESC` |
| `--page-size`       | int    | Number of results per page |
| `--next-page-token` | string | Pagination token for the next page |
| `--all`             | bool   | Automatically fetch all pages |

When `--all` is set, the CLI follows `nextPageToken` values until no more
pages remain. Otherwise, if a token is present in the response, the CLI
prints a hint:

```
More results available. Use --next-page-token eyJvZmZzZXQiOjI1fQ==
```

## Action Execution

Actions are exposed as direct subcommands under each entity group. For
example, if the `tag` action is listed in `entity.Actions`, a `tag`
subcommand is created:

```bash
catalogctl mcp mcpservers tag filesystem --params "tags=production,verified" --dry-run
```

| Flag       | Type   | Description |
|------------|--------|-------------|
| `--params` | string | Comma-separated `key=value` pairs |
| `--dry-run`| bool   | Preview without applying (only if action supports it) |

The `--dry-run` flag is only added when the action definition has
`SupportsDryRun: true`.

Action output in table mode:

```
[dry-run] Action "tag" on "filesystem": completed
  would set 2 tags on filesystem
```

---

## Conformance Suite

### Purpose

The conformance suite validates that **all** plugins loaded on a catalog
server meet the Phase 5 universal framework contract. It discovers plugins
at test start, then runs a structured set of checks against each one. A
plugin that passes conformance is guaranteed to work with the generic UI
and the catalogctl CLI without any plugin-specific code.

**Location:** `tests/conformance/`

### Architecture

```
   go test ./tests/conformance/... -v
                   |
                   v
            TestMain()
            Set serverURL from CATALOG_SERVER_URL env
                   |
                   v
            waitForReady()
            Poll /readyz up to 30s
                   |
                   v
            GET /api/plugins
                   |
         +---------+---------+
         |         |         |
         v         v         v
      plugin A  plugin B  plugin C
         |         |         |
   +-----+-----+  |    +----+----+
   |     |     |  ...   |    |    |
   v     v     v        v    v    v
 caps  endp  acts     caps endp filters
 test  test  test     test test  test
```

### Test Files and Coverage

| Test File | What It Tests |
|-----------|---------------|
| `conformance_test.go` | Orchestration: discovers plugins, runs sub-tests per plugin. Also tests health endpoints (`/healthz`, `/livez`, `/readyz`), readyz component details, plugin count, basic fields (name, version, description, basePath), name/basePath uniqueness, pagination with `pageSize`, and 404 for unknown capabilities endpoint. |
| `capabilities_test.go` | V2 capabilities schema completeness: schemaVersion, plugin metadata, entity definitions (kind, plural, displayName, endpoints, columns with name/displayName/path/type), filter field and detail field validation, action-reference integrity (entity action IDs must resolve to real definitions), action definition fields (ID, displayName, description, scope must be `source` or `asset`), and cross-checks that inline capabilities match the dedicated `/capabilities` endpoint. |
| `endpoints_test.go` | All declared endpoints return proper status codes: list endpoints return 200 with valid JSON containing `items` array and `size` number; get endpoints resolve the first item by name and verify it returns the correct entity; get with a nonexistent name returns 404 (or 400). Handles multi-parameter get patterns by falling back to management entity routes. |
| `actions_test.go` | All declared asset-scoped actions can be invoked in dry-run mode against the first available entity: verifies response has `action` and `status` fields, status is one of `completed`, `dry-run`, or `error`. Builds minimal params for known action types (tag, annotate, deprecate). Sends `X-User-Role: operator` header for RBAC. |
| `filters_test.go` | FilterQuery parsing and application for every declared `filterField`: constructs type-appropriate queries (boolean, numeric, string), verifies 200 OK (zero results is acceptable), and flags 400 or 500 responses as failures. Also tests `orderBy` with `sortOrder=ASC` on every column marked `sortable`. |

### Per-Plugin Test Matrix

Every plugin discovered via `GET /api/plugins` is tested with this matrix:

```
TestConformance/{pluginName}
  |-- healthy              Plugin is reporting healthy=true
  |-- capabilities         Full V2 schema validation
  |-- endpoints            List and get for all entity types
  |-- actions              Dry-run invocation of all declared actions
  +-- filters              Filter and sort query acceptance
```

Additional standalone tests run once (not per-plugin):

| Test Function | What It Validates |
|---------------|-------------------|
| `TestHealthEndpoints` | `/healthz`, `/livez`, `/readyz` all return 200 with `status` field |
| `TestReadyzComponents` | `/readyz` response includes `components` object with `database`, `initial_load`, and `plugins` sub-objects |
| `TestPluginCount` | At least 1 plugin loaded; `count` matches array length |
| `TestPluginsHaveBasicFields` | Every plugin has non-empty name, version, description, basePath starting with `/api/` |
| `TestCapabilitiesEndpointNotFound` | `GET /api/plugins/nonexistent/capabilities` returns 404 |
| `TestPluginNamesUnique` | No two plugins share the same name |
| `TestBasePathsUnique` | No two plugins share the same basePath |
| `TestPagination` | `pageSize=1` parameter is accepted on all list endpoints |

## Running the Conformance Suite

```bash
# 1. Start the catalog stack
docker compose -f docker-compose.catalog.yaml up --build -d

# 2. Wait for readiness (optional -- the suite polls internally)
curl -s http://localhost:8080/readyz

# 3. Run conformance tests
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1
```

The suite uses `waitForReady()` which polls `/readyz` every second for up
to 30 seconds before failing. If `CATALOG_SERVER_URL` is not set, it
defaults to `http://localhost:8080`.

### Expected Output (all plugins passing)

```
=== RUN   TestConformance
    conformance_test.go:180: discovered 3 plugin(s)
=== RUN   TestConformance/model
=== RUN   TestConformance/model/healthy
=== RUN   TestConformance/model/capabilities
=== RUN   TestConformance/model/endpoints
=== RUN   TestConformance/model/actions
=== RUN   TestConformance/model/filters
=== RUN   TestConformance/mcp
...
=== RUN   TestConformance/knowledge
...
--- PASS: TestConformance (2.34s)
=== RUN   TestHealthEndpoints
--- PASS: TestHealthEndpoints (0.01s)
=== RUN   TestPluginNamesUnique
--- PASS: TestPluginNamesUnique (0.01s)
...
PASS
```

## catalogctl Unit Tests

The CLI has its own unit test suite in `cmd/catalogctl/catalogctl_test.go`
that uses `httptest.Server` to mock the catalog server:

| Test | What It Verifies |
|------|------------------|
| `TestExtractValue` | Dot-path extraction for strings, nested maps, integers, floats, booleans, arrays, nil, and missing keys |
| `TestExtractItems` | Item array extraction by plural key, fallback to `items`/`results`/`data`, and nil for non-matches |
| `TestTruncate` | String truncation with `...` suffix at various lengths |
| `TestToMapSlice` | Conversion of `[]any` to `[]map[string]any` with type filtering |
| `TestBuildPluginCommand` | Plugin command has entity, sources, and actions subcommands |
| `TestBuildEntityCommand` | Entity command has list, get, and per-action subcommands |
| `TestBuildEntityActionCommand_*` | Nil for missing action; `--dry-run` flag when supported |
| `TestBuildPluginCommand_SkipsSources*` | Sources subcommand omitted when `Sources` is nil |
| `TestBuildPluginCommand_SkipsActions*` | Actions subcommand omitted when `Actions` is empty |
| `TestPluginsListHTTP` | HTTP integration: plugins list round-trip |
| `TestHealthHTTP` | HTTP integration: health and readiness round-trip |
| `TestEntityListHTTP` | HTTP integration: entity list with item extraction |
| `TestEntityGetHTTP` | HTTP integration: entity get by name |
| `TestActionHTTP` | HTTP integration: action POST with dry-run |
| `TestSourcesListHTTP` | HTTP integration: sources list |
| `TestClientErrorHandling` | 500 responses produce descriptive errors |
| `TestClientNotFoundHandling` | 404 responses produce descriptive errors |
| `TestDiscoverPluginsIntegration` | Full discovery: 2 plugins produce correct command tree |
| `TestInferColumns` | Fallback column inference from first item keys |

## Key Files

| File | Purpose |
|------|---------|
| `cmd/catalogctl/main.go` | Entry point: calls `discoverPlugins()` then `rootCmd.Execute()` |
| `cmd/catalogctl/root.go` | Root Cobra command with `--server` and `--output` global flags |
| `cmd/catalogctl/client.go` | HTTP client wrapper: `getJSON`, `getRaw`, `postJSON` with 30s timeout |
| `cmd/catalogctl/discover.go` | Plugin discovery, command tree building, `extractValue()` for JSON path extraction |
| `cmd/catalogctl/entity.go` | Entity list/get commands with pagination, V2 column-driven table rendering |
| `cmd/catalogctl/action.go` | Action subcommand builder with `--dry-run` and `--params` flags |
| `cmd/catalogctl/plugins.go` | Static `plugins` command |
| `cmd/catalogctl/health.go` | Static `health` command (liveness + readiness) |
| `cmd/catalogctl/output.go` | Output formatting: `printTable`, `printJSON`, `printYAML`, `truncate` |
| `cmd/catalogctl/types.go` | CLI-local type definitions mirroring server response structures |
| `cmd/catalogctl/catalogctl_test.go` | Unit and HTTP integration tests (21 test functions) |
| `tests/conformance/conformance_test.go` | Conformance orchestration, health, pagination, uniqueness tests |
| `tests/conformance/capabilities_test.go` | V2 capabilities schema validation |
| `tests/conformance/endpoints_test.go` | List and get endpoint validation |
| `tests/conformance/actions_test.go` | Action invocation (dry-run) validation |
| `tests/conformance/filters_test.go` | Filter and sort query acceptance validation |

---

[Back to Clients](./README.md) | [Prev: Generic UI](./generic-ui.md)
