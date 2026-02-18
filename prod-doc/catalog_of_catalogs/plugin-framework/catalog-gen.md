# catalog-gen: Plugin Scaffolding and Build Tool

## Overview

`catalog-gen` is the developer-facing CLI tool for the catalog-of-catalogs ecosystem. It automates plugin scaffolding, validation, versioning, conformance testing, documentation generation, and server composition. After Phase 9, adding a new plugin follows a repeatable, machine-checkable workflow with zero bespoke code.

**Location:** `cmd/catalog-gen/`

## Commands

| Command | Description |
|---------|-------------|
| `catalog-gen init --name=<name>` | Scaffold a new plugin (Go code, catalog.yaml, plugin.yaml, conformance tests, docs) |
| `catalog-gen generate --config=<path>` | Regenerate non-editable files from catalog.yaml |
| `catalog-gen validate [dir]` | Validate plugin.yaml, catalog.yaml, conformance tests, and docs |
| `catalog-gen bump-version [major\|minor\|patch]` | Bump the version in plugin.yaml |
| `catalog-gen build-server --manifest=<path>` | Generate a custom catalog-server from a manifest |
| `catalog-gen compat-matrix --manifest=<path>` | Generate a compatibility matrix for a server manifest |

---

## init: Plugin Scaffolding

Generates a complete plugin directory structure with all files needed to pass governance checks.

```bash
catalog-gen init --name=widgets --display-name="Widgets" --entity-kind=Widget
```

### Generated Files

```
catalog/plugins/widgets/
├── register.go              # init() + plugin.Register()
├── plugin.go                # CatalogPlugin implementation
├── asset_mapper.go          # AssetMapperProvider
├── actions.go               # ActionProvider (builtin actions)
├── management.go            # CapabilitiesV2Provider, SourceManager
├── catalog.yaml             # Schema definition (entity fields, columns, filters)
├── plugin.yaml              # Distribution/governance metadata
├── data/
│   └── widgets.yaml         # Sample YAML data
├── tests/
│   └── conformance_test.go  # Conformance test importing pkg/catalog/conformance
└── docs/
    ├── README.md             # Plugin overview
    ├── provider-guide.md     # Provider configuration guide
    ├── schema-guide.md       # Schema field mapping guide
    ├── testing.md            # Testing instructions
    └── publishing.md         # Versioning and publishing guide
```

### File Categories

| Category | Files | Overwrite on `generate` |
|----------|-------|------------------------|
| Editable | plugin.go, actions.go, management.go, asset_mapper.go, data/*.yaml | Never |
| Regenerable | register.go, catalog.yaml stubs, conformance test scaffold | Yes |
| Metadata | plugin.yaml | Only via `bump-version` |
| Documentation | docs/*.md | Generated once by `init`, never overwritten |

---

## validate: Plugin Validation

Validates a plugin directory for completeness and correctness.

```bash
# Basic validation
catalog-gen validate catalog/plugins/widgets/

# With governance checks (ownership, license, docs completeness)
catalog-gen validate --governance catalog/plugins/widgets/
```

### Validation Checks

| Check | Description |
|-------|-------------|
| plugin.yaml exists | Distribution metadata file present |
| plugin.yaml fields | apiVersion, kind, metadata.name, spec.version, spec.displayName, spec.description required |
| Semver format | spec.version must be valid semver (e.g., "1.0.0") |
| catalog.yaml exists | Schema definition file present |
| Conformance tests | Tests directory with at least one `_test.go` file |
| Documentation | docs/ directory with required sections (README, provider guide, testing) |

### Governance Checks (`--governance`)

When the `--governance` flag is set, additional checks run:

| Check | Description |
|-------|-------------|
| Ownership | `spec.owners` must have at least one entry with team and contact |
| License | `spec.license` must be set (e.g., "Apache-2.0") |
| Compatibility | `spec.compatibility.catalogServer` and `spec.compatibility.frameworkApi` must be set |
| Repository | `spec.repository` should be set for published plugins |

---

## bump-version: Version Management

Bumps the semver version in `plugin.yaml`.

```bash
# Bump patch version (0.9.0 -> 0.9.1)
catalog-gen bump-version patch --dir=catalog/plugins/widgets/

# Bump minor version (0.9.0 -> 0.10.0)
catalog-gen bump-version minor --dir=catalog/plugins/widgets/

# Bump major version (0.9.0 -> 1.0.0)
catalog-gen bump-version major --dir=catalog/plugins/widgets/
```

The command reads `plugin.yaml`, increments the specified version component, and writes the file back atomically.

---

## build-server: Custom Server Composition

Generates a compilable catalog-server from a declarative manifest listing which plugins to include.

```bash
# Generate server files
catalog-gen build-server --manifest=manifest.yaml --output=build/server

# Validate compatibility without generating
catalog-gen build-server --manifest=manifest.yaml --validate-only
```

### Server Manifest Format

```yaml
# CatalogServerBuild manifest
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogServerBuild
spec:
  base:
    module: github.com/kubeflow/model-registry
    version: v0.9.0
    goVersion: "1.22"
  plugins:
    - name: model
      module: github.com/kubeflow/model-registry/catalog/plugins/model
      version: v0.9.0
    - name: mcp
      module: github.com/kubeflow/model-registry/catalog/plugins/mcp
      version: v0.9.0
    - name: knowledge
      module: github.com/kubeflow/model-registry/catalog/plugins/knowledge
      version: v0.9.0
    # ... additional plugins
```

### Generated Output

```
build/server/
├── main.go          # Go entry point with blank imports for each plugin
├── go.mod           # Module definition with required dependencies
└── Dockerfile       # Multi-stage build for distroless container image
```

The generated `main.go` follows the same blank-import pattern used by `cmd/catalog-server/main.go`:

```go
package main

import (
    _ "github.com/kubeflow/model-registry/catalog/plugins/model"
    _ "github.com/kubeflow/model-registry/catalog/plugins/mcp"
    // ... one import per plugin in manifest
)

func main() {
    // Standard catalog-server startup
}
```

### Flags

| Flag | Description |
|------|-------------|
| `--manifest` | Path to CatalogServerBuild manifest YAML |
| `--output` | Output directory for generated files (default: `build/server`) |
| `--validate-only` | Check compatibility without generating files |
| `--compile` | Run `go build` after generation (not yet implemented) |

---

## compat-matrix: Compatibility Matrix

Generates a Markdown compatibility matrix showing which plugin versions work with which server versions.

```bash
catalog-gen compat-matrix --manifest=manifest.yaml --output=compat-matrix.md
```

Reads each plugin's `plugin.yaml` compatibility spec and produces a table:

```markdown
| Plugin | Min Server | Max Server | Framework API |
|--------|-----------|-----------|---------------|
| model  | 0.9.0     | 1.x       | v1alpha1      |
| mcp    | 0.9.0     | 1.x       | v1alpha1      |
| ...    | ...       | ...       | ...           |
```

---

## plugin.yaml Schema

The `plugin.yaml` file is the distribution and governance metadata for a plugin. It is separate from `catalog.yaml` (which defines the entity schema).

```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: CatalogPlugin
metadata:
  name: mcp
spec:
  displayName: MCP Servers
  description: Catalog of MCP server configurations for AI tool integration
  version: "0.9.0"
  owners:
    - team: ai-platform
      contact: "#ai-platform"
  compatibility:
    catalogServer:
      minVersion: "0.9.0"
      maxVersion: "1.x"
    frameworkApi: v1alpha1
  providers:
    - yaml
  license: Apache-2.0
  repository: https://github.com/kubeflow/model-registry
```

### Go Types

```go
// pkg/catalog/plugin/plugin_metadata.go

type PluginMetadataSpec struct {
    APIVersion string             `yaml:"apiVersion" json:"apiVersion"`
    Kind       string             `yaml:"kind" json:"kind"`
    Metadata   PluginMetadataName `yaml:"metadata" json:"metadata"`
    Spec       PluginMetadataBody `yaml:"spec" json:"spec"`
}

type PluginMetadataBody struct {
    DisplayName   string            `yaml:"displayName" json:"displayName"`
    Description   string            `yaml:"description" json:"description"`
    Version       string            `yaml:"version" json:"version"`
    Owners        []OwnerRef        `yaml:"owners" json:"owners"`
    Compatibility CompatibilitySpec `yaml:"compatibility" json:"compatibility"`
    Providers     []string          `yaml:"providers,omitempty" json:"providers,omitempty"`
    License       string            `yaml:"license,omitempty" json:"license,omitempty"`
    Repository    string            `yaml:"repository,omitempty" json:"repository,omitempty"`
}

type CompatibilitySpec struct {
    CatalogServer VersionRange `yaml:"catalogServer" json:"catalogServer"`
    FrameworkAPI  string       `yaml:"frameworkApi" json:"frameworkApi"`
}

type VersionRange struct {
    MinVersion string `yaml:"minVersion" json:"minVersion"`
    MaxVersion string `yaml:"maxVersion" json:"maxVersion"`
}
```

### Loading and Validation

```go
// Load from file
spec, err := plugin.LoadPluginMetadata("catalog/plugins/mcp/plugin.yaml")

// Validate
errs := plugin.ValidatePluginMetadata(spec)

// Parse semver
major, minor, patch, err := plugin.ParseSemver("1.2.3")

// Bump version
newVersion, err := plugin.BumpVersion("1.2.3", "minor") // "1.3.0"
```

---

## Golden Tests

The `catalog-gen` tool includes golden tests that verify deterministic generation. Running `init` twice with the same parameters produces identical output for all non-editable files. This prevents regressions in template rendering.

```bash
go test ./cmd/catalog-gen/... -run Golden -count=1
```

---

## End-to-End Plugin Development Workflow

```
1. Scaffold          catalog-gen init --name=widgets
2. Implement         Edit plugin.go, actions.go, management.go
3. Add data          Create data/widgets.yaml
4. Validate          catalog-gen validate catalog/plugins/widgets/
5. Test              CATALOG_SERVER_URL=http://localhost:8080 go test ./catalog/plugins/widgets/tests/...
6. Governance check  catalog-gen validate --governance catalog/plugins/widgets/
7. Version           catalog-gen bump-version patch --dir=catalog/plugins/widgets/
8. Build server      catalog-gen build-server --manifest=manifest.yaml
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/catalog-gen/main.go` | Entry point with all subcommands |
| `cmd/catalog-gen/init.go` | Plugin scaffolding (init command) |
| `cmd/catalog-gen/validate.go` | Validation with optional governance checks |
| `cmd/catalog-gen/bump_version.go` | Semver version bumping |
| `cmd/catalog-gen/server_builder.go` | Server composition from manifest |
| `cmd/catalog-gen/compat_matrix.go` | Compatibility matrix generation |
| `cmd/catalog-gen/gen_conformance.go` | Conformance test scaffold generation |
| `cmd/catalog-gen/gen_docs.go` | Documentation kit generation |
| `cmd/catalog-gen/golden_test.go` | Deterministic generation tests |
| `cmd/catalog-gen/templates/` | Go templates for all generated files |
| `pkg/catalog/plugin/plugin_metadata.go` | PluginMetadataSpec types, load, validate |
| `examples/catalog-server-manifest.yaml` | Example server build manifest |

---

[Back to Plugin Framework](./README.md) | [Prev: Configuration](./configuration.md)
