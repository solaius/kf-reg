# Plugin Governance and Supported Plugin Index

## Overview

Plugin governance provides machine-checkable quality gates for the catalog plugin ecosystem. It ensures that plugins meet minimum standards for metadata, compatibility, ownership, licensing, testing, and documentation before being designated as "supported."

**Location:** `pkg/catalog/plugin/governance_checks.go`, `deploy/plugin-index/`

## Governance Checks

The `RunGovernanceChecks` function inspects a plugin directory and produces a structured report with pass/fail results for 7 check categories.

```go
// pkg/catalog/plugin/governance_checks.go
func RunGovernanceChecks(pluginDir string) GovernanceReport
```

### Check Categories

| # | Check | What It Verifies |
|---|-------|------------------|
| 1 | **plugin.yaml** | File exists, parses correctly, required fields present (apiVersion, kind, name, version, displayName, description) |
| 2 | **catalog.yaml** | Schema definition file exists in the plugin directory |
| 3 | **Compatibility** | `spec.compatibility.catalogServer` has minVersion and maxVersion; `spec.compatibility.frameworkApi` is set |
| 4 | **Ownership** | `spec.owners` has at least one entry with team and contact fields |
| 5 | **License** | `spec.license` is set (e.g., "Apache-2.0") |
| 6 | **Conformance Tests** | A `tests/` directory exists containing at least one `_test.go` file |
| 7 | **Documentation** | A `docs/` directory exists with a README.md |

### GovernanceReport

```go
type GovernanceReport struct {
    PluginDir   string              `json:"pluginDir"`
    PluginName  string              `json:"pluginName"`
    PassedAll   bool                `json:"passedAll"`
    Checks      []GovernanceCheck   `json:"checks"`
    Summary     string              `json:"summary"`
}

type GovernanceCheck struct {
    Name    string `json:"name"`
    Passed  bool   `json:"passed"`
    Message string `json:"message"`
}
```

### Running Governance Checks

Via `catalog-gen`:

```bash
# Run governance checks on a single plugin
catalog-gen validate --governance catalog/plugins/mcp/

# Expected output for a compliant plugin:
# PASS  plugin.yaml valid
# PASS  catalog.yaml exists
# PASS  compatibility fields present
# PASS  ownership declared
# PASS  license present (Apache-2.0)
# PASS  conformance tests exist
# PASS  documentation present
# All 7 governance checks passed
```

Via Go code:

```go
import "github.com/kubeflow/model-registry/pkg/catalog/plugin"

report := plugin.RunGovernanceChecks("catalog/plugins/mcp/")
if !report.PassedAll {
    for _, check := range report.Checks {
        if !check.Passed {
            fmt.Printf("FAIL: %s - %s\n", check.Name, check.Message)
        }
    }
}
```

---

## Supported Plugin Index

The plugin index is a Git-based registry of plugin metadata at `deploy/plugin-index/`. It serves as the canonical list of known plugins with their compatibility, ownership, and tier information.

### Directory Structure

```
deploy/plugin-index/
├── README.md                    # Index format documentation
├── schema.yaml                  # JSON Schema for PluginIndexEntry
└── plugins/
    ├── model.yaml               # Model plugin entry
    ├── mcp.yaml                 # MCP plugin entry
    ├── knowledge.yaml           # Knowledge plugin entry
    ├── prompts.yaml             # Prompts plugin entry
    ├── agents.yaml              # Agents plugin entry
    ├── guardrails.yaml          # Guardrails plugin entry
    ├── policies.yaml            # Policies plugin entry
    └── skills.yaml              # Skills plugin entry
```

### Index Entry Format

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: PluginIndexEntry
metadata:
  name: mcp
spec:
  displayName: MCP Servers
  description: Catalog of MCP server configurations for AI tool integration
  tier: built-in
  module: github.com/kubeflow/model-registry/catalog/plugins/mcp
  version: v1alpha1
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.x"
    frameworkApi: v1alpha1
  owners:
    - team: ai-platform
      contact: "#ai-platform"
  security:
    license: Apache-2.0
```

### Plugin Tiers

| Tier | Description | Governance Requirements |
|------|-------------|------------------------|
| **built-in** | Ships with the default catalog-server binary | All 7 governance checks must pass |
| **supported** | Maintained by the project team but optionally included | All 7 governance checks must pass |
| **community** | Third-party plugins with varying support levels | plugin.yaml and basic validation |

### Adding a Plugin to the Index

1. Create `plugin.yaml` in your plugin directory
2. Pass all governance checks: `catalog-gen validate --governance <plugin-dir>/`
3. Create an index entry YAML in `deploy/plugin-index/plugins/<name>.yaml`
4. Submit a PR with the index entry

---

## Built-in Plugin Status

All 8 built-in plugins have plugin.yaml metadata and pass governance checks:

| Plugin | Version | License | Providers | Framework API |
|--------|---------|---------|-----------|---------------|
| model | 0.9.0 | Apache-2.0 | yaml, hf | v1alpha1 |
| mcp | 0.9.0 | Apache-2.0 | yaml | v1alpha1 |
| knowledge | 0.9.0 | Apache-2.0 | yaml | v1alpha1 |
| prompts | 0.9.0 | Apache-2.0 | yaml | v1alpha1 |
| agents | 0.9.0 | Apache-2.0 | yaml, git | v1alpha1 |
| guardrails | 0.9.0 | Apache-2.0 | yaml | v1alpha1 |
| policies | 0.9.0 | Apache-2.0 | yaml | v1alpha1 |
| skills | 0.9.0 | Apache-2.0 | yaml | v1alpha1 |

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/governance_checks.go` | RunGovernanceChecks with 7 check functions |
| `pkg/catalog/plugin/governance_checks_test.go` | Tests with fixture directories |
| `pkg/catalog/plugin/plugin_metadata.go` | PluginMetadataSpec types, load, validate, semver |
| `cmd/catalog-gen/validate.go` | `catalog-gen validate` with `--governance` flag |
| `deploy/plugin-index/README.md` | Index format documentation |
| `deploy/plugin-index/schema.yaml` | JSON Schema for PluginIndexEntry |
| `deploy/plugin-index/plugins/*.yaml` | Index entries for all 8 built-in plugins |
| `catalog/plugins/*/plugin.yaml` | Plugin metadata files for all 8 plugins |

---

[Back to Operations](./README.md) | [Prev: Upgrade Guide](./upgrade-guide.md)
