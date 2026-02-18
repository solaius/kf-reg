# Phase 9: Extensibility and Ecosystem

**Date**: 2026-02-17
**Status**: Complete
**Phase**: Phase 9 — Extensibility and Ecosystem

## Summary

Phase 9 transforms the catalog platform from "a platform with plugins" into "an ecosystem where adding new plugins is boring, repeatable, and well-governed." This phase delivers: `plugin.yaml` as a distribution/governance metadata format, upgraded `catalog-gen` with `validate`, `bump-version`, and `build-server` commands, formalized UI hints schema, an importable conformance test harness, server builder for custom plugin compositions, documentation kit generation, governance checks, and a supported plugin index.

## Motivation

- After 8 phases of building the runtime platform, the gap was developer experience: adding a new plugin required tribal knowledge about which files to create, how to structure tests, and what metadata was expected
- Plugin ecosystem scale requires machine-checkable governance: conformance, compatibility, ownership, and licensing
- The UI hints schema was informal — extending it to be versioned and validated ensures plugins render correctly without bespoke UI code
- Server composition (choosing which plugins to include) was manual blank-import editing; a builder pipeline automates this

## What Changed

### M9.1: plugin.yaml Schema and catalog-gen Foundation

#### Files Created
| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/plugin_metadata.go` | PluginMetadataSpec, CompatibilitySpec, VersionRange, OwnerRef types; LoadPluginMetadata, ValidatePluginMetadata, ParseSemver, BumpVersion functions |
| `pkg/catalog/plugin/plugin_metadata_test.go` | Table-driven tests for YAML parsing, validation, semver parsing, version bumping |
| `cmd/catalog-gen/validate.go` | `catalog-gen validate [dir]` command with `--governance` flag |
| `cmd/catalog-gen/validate_test.go` | Tests with temp directories for valid/invalid plugin configs |
| `cmd/catalog-gen/bump_version.go` | `catalog-gen bump-version [major|minor|patch]` command |
| `cmd/catalog-gen/bump_version_test.go` | Version bump tests |
| `cmd/catalog-gen/templates/plugin/plugin_yaml.gotmpl` | Template for plugin.yaml generation during init |

#### Files Modified
| File | Change |
|------|--------|
| `cmd/catalog-gen/main.go` | Added `validate` and `bump-version` subcommands |
| `cmd/catalog-gen/types.go` | Added PluginConfig types |
| `cmd/catalog-gen/helpers.go` | Added loadPluginConfig helper |
| `cmd/catalog-gen/templates.go` | Added TmplPluginYAML constant |

### M9.2: UI Hints Schema Formalization

#### Files Created
| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/ui_hints_types.go` | FieldDisplayType enum (11 types), ListViewHints, DetailViewHints, SearchHints, ActionDisplayHints, ColumnDisplay, SortHint, DetailSection, FieldDisplay, FacetHint, ActionConfirmation |
| `pkg/catalog/plugin/ui_hints_validator.go` | ValidateUIHints function with per-field validation |
| `pkg/catalog/plugin/ui_hints_validator_test.go` | Table-driven tests for valid/invalid hints, display type validation, secretRef safety |

#### Files Modified
| File | Change |
|------|--------|
| `pkg/catalog/plugin/capabilities_types.go` | Extended EntityUIHints with ListView, DetailView, Search, ActionHints pointer fields |
| `catalog/plugins/mcp/management.go` | Added extended UI hints to MCP plugin as reference implementation |

### M9.3: Importable Conformance Harness

#### Files Created
| File | Purpose |
|------|---------|
| `pkg/catalog/conformance/types.go` | Exported response types (PluginsResponse, PluginInfo, CapabilitiesV2, etc.) |
| `pkg/catalog/conformance/config.go` | HarnessConfig and ExpectedCaps |
| `pkg/catalog/conformance/helpers.go` | GetJSON, WaitForReady HTTP helpers |
| `pkg/catalog/conformance/report.go` | ConformanceResult, CategoryResult, TestResult with JSON/Summary output |
| `pkg/catalog/conformance/harness.go` | RunConformance entry point with 6 test categories |
| `pkg/catalog/conformance/category_a_capabilities.go` | Capability contract tests |
| `pkg/catalog/conformance/category_b_list_get.go` | List/get contract tests |
| `pkg/catalog/conformance/category_c_sources.go` | Source management tests |
| `pkg/catalog/conformance/category_d_security.go` | Security tests (skippable) |
| `pkg/catalog/conformance/category_e_observability.go` | Observability tests (skippable) |
| `pkg/catalog/conformance/category_f_openapi.go` | OpenAPI merge tests (skippable) |

#### Files Modified
| File | Change |
|------|--------|
| `tests/conformance/conformance_test.go` | Added TestConformanceV2 delegating to new harness |

### M9.4: catalog-gen Conformance Scaffold and Documentation Kit

#### Files Created
| File | Purpose |
|------|---------|
| `cmd/catalog-gen/gen_conformance.go` | Conformance scaffold generation |
| `cmd/catalog-gen/gen_docs.go` | Documentation kit generation |
| `cmd/catalog-gen/golden_test.go` | Golden tests for deterministic generation |
| `cmd/catalog-gen/templates/conformance/conformance_test.gotmpl` | Conformance test scaffold template |
| `cmd/catalog-gen/templates/docs/readme.gotmpl` | Plugin README template |
| `cmd/catalog-gen/templates/docs/provider_guide.gotmpl` | Provider guide template |
| `cmd/catalog-gen/templates/docs/schema_guide.gotmpl` | Schema guide template |
| `cmd/catalog-gen/templates/docs/testing.gotmpl` | Testing guide template |
| `cmd/catalog-gen/templates/docs/publishing.gotmpl` | Publishing guide template |

### M9.5: Server Builder and Packaging Pipeline

#### Files Created
| File | Purpose |
|------|---------|
| `cmd/catalog-gen/server_builder.go` | `catalog-gen build-server` command with ServerManifest types |
| `cmd/catalog-gen/server_builder_test.go` | Tests for manifest validation and file generation |
| `cmd/catalog-gen/compat_matrix.go` | Compatibility matrix generator |
| `cmd/catalog-gen/compat_matrix_test.go` | Matrix generation tests |
| `cmd/catalog-gen/templates/server/main.gotmpl` | Generated main.go template |
| `cmd/catalog-gen/templates/server/go_mod.gotmpl` | Generated go.mod template |
| `cmd/catalog-gen/templates/server/dockerfile.gotmpl` | Generated Dockerfile template |
| `cmd/catalog-gen/templates/server/compat_matrix.gotmpl` | Compatibility matrix Markdown template |
| `examples/catalog-server-manifest.yaml` | Example manifest with all 8 built-in plugins |

### M9.6: Governance Checks and Supported Plugin Index

#### Files Created
| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/governance_checks.go` | RunGovernanceChecks with 7 check functions |
| `pkg/catalog/plugin/governance_checks_test.go` | Tests with fixture directories |
| `deploy/plugin-index/README.md` | Index format documentation |
| `deploy/plugin-index/schema.yaml` | JSON Schema for PluginIndexEntry |
| `deploy/plugin-index/plugins/model.yaml` | Model plugin index entry |
| `deploy/plugin-index/plugins/mcp.yaml` | MCP plugin index entry |
| `deploy/plugin-index/plugins/knowledge.yaml` | Knowledge plugin index entry |
| `deploy/plugin-index/plugins/prompts.yaml` | Prompts plugin index entry |
| `deploy/plugin-index/plugins/agents.yaml` | Agents plugin index entry |
| `deploy/plugin-index/plugins/guardrails.yaml` | Guardrails plugin index entry |
| `deploy/plugin-index/plugins/policies.yaml` | Policies plugin index entry |
| `deploy/plugin-index/plugins/skills.yaml` | Skills plugin index entry |

### M9.7: End-to-End Integration and Documentation

#### Files Created
| File | Purpose |
|------|---------|
| `catalog/plugins/model/plugin.yaml` | Model plugin metadata |
| `catalog/plugins/mcp/plugin.yaml` | MCP plugin metadata |
| `catalog/plugins/knowledge/plugin.yaml` | Knowledge plugin metadata |
| `catalog/plugins/prompts/plugin.yaml` | Prompts plugin metadata |
| `catalog/plugins/agents/plugin.yaml` | Agents plugin metadata |
| `catalog/plugins/guardrails/plugin.yaml` | Guardrails plugin metadata |
| `catalog/plugins/policies/plugin.yaml` | Policies plugin metadata |
| `catalog/plugins/skills/plugin.yaml` | Skills plugin metadata |

## How It Works

### plugin.yaml Schema
```go
type PluginMetadataSpec struct {
    APIVersion string             `yaml:"apiVersion" json:"apiVersion"`
    Kind       string             `yaml:"kind" json:"kind"`
    Metadata   PluginMetadataName `yaml:"metadata" json:"metadata"`
    Spec       PluginMetadataBody `yaml:"spec" json:"spec"`
}
```
Loaded via `LoadPluginMetadata(path)`, validated via `ValidatePluginMetadata(spec)`.

### UI Hints Extension
```go
type EntityUIHints struct {
    Icon           string              `json:"icon,omitempty"`
    Color          string              `json:"color,omitempty"`
    NameField      string              `json:"nameField,omitempty"`
    DetailSections []string            `json:"detailSections,omitempty"`
    ListView       *ListViewHints      `json:"listView,omitempty"`
    DetailView     *DetailViewHints    `json:"detailView,omitempty"`
    Search         *SearchHints        `json:"search,omitempty"`
    ActionHints    *ActionDisplayHints `json:"actionHints,omitempty"`
}
```
All new fields are optional pointers for backward compatibility.

### Conformance Harness
```go
func RunConformance(t *testing.T, cfg HarnessConfig) ConformanceResult
```
6 categories (A-F): capabilities, list/get, sources, security, observability, OpenAPI.

### Server Builder
```bash
catalog-gen build-server --manifest=manifest.yaml --output=build/server
```
Reads `CatalogServerBuild` manifest, generates `main.go` + `go.mod` + `Dockerfile`.

### Governance Checks
```go
func RunGovernanceChecks(pluginDir string) GovernanceReport
```
7 checks: plugin.yaml, catalog.yaml, compatibility, ownership, license, conformance tests, docs.

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| plugin.yaml separate from catalog.yaml | catalog.yaml is schema definition, plugin.yaml is distribution/governance metadata | Merge into one file — rejected for separation of concerns |
| All UI hints fields optional pointers | Backward compatibility — existing plugins don't break | Required fields — rejected as it would break all 8 plugins |
| Conformance as importable library | External plugin developers can import and run against their plugin | Shell script harness — rejected for Go ecosystem integration |
| Server builder generates code, doesn't compile | Keep build step optional, support `--compile` flag later | Direct compilation — deferred, not all environments have Go toolchain |
| Governance checks in pkg/catalog/plugin | Reusable from catalog-gen and CI pipelines | Separate package — rejected for simplicity |

## Testing

- `go test ./pkg/catalog/plugin/... -count=1` — all plugin package tests (metadata, UI hints, governance)
- `go test ./cmd/catalog-gen/... -count=1` — all catalog-gen tests (validate, bump-version, server builder, golden)
- `go test ./tests/conformance/... -run "Phase8|HA|Job" -count=1` — Phase 8 regression tests

## Verification

```bash
# Build all affected packages
go build ./cmd/catalog-gen/
go build ./cmd/catalog-server/
go build ./pkg/catalog/plugin/...
go build ./pkg/catalog/conformance/...

# Run all tests
go test ./pkg/catalog/plugin/... -count=1
go test ./cmd/catalog-gen/... -count=1
go test ./tests/conformance/... -run "Phase8|HA|Job" -count=1

# Check plugin.yaml files exist
ls catalog/plugins/*/plugin.yaml

# Check plugin index entries
ls deploy/plugin-index/plugins/
```

## Dependencies & Impact

- Enables external plugin developers to scaffold, test, and publish plugins with zero bespoke code
- Governance checks automate the "supported plugin" designation process
- Server builder enables custom server compositions
- UI hints schema enables richer plugin rendering without UI code changes
- Conformance harness enables programmatic validation of plugin compliance

## Open Items

- `catalog-gen build-server --compile` flag not yet implemented (generates but doesn't build)
- OCI provider support in server builder deferred
- Cosign/Sigstore image signing hooks are designed but not implemented
- Extended UI hints consumed by frontend require frontend changes in a future phase
- Conformance suite requires live server for full execution
