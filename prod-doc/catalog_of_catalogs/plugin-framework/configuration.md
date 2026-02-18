# Catalog Configuration

## Overview

The catalog server uses a declarative YAML configuration file (`sources.yaml`) to define which plugins are active, what data sources each plugin reads from, and how those sources are configured. A single file drives all plugins in the process: models, MCP servers, knowledge sources, agents, prompts, guardrails, policies, and skills.

Configuration flows through three layers:

```
sources.yaml          Environment Variables          CLI Flags
     │                        │                          │
     ▼                        ▼                          ▼
 LoadConfig()         os.Getenv() in main          flag.Parse() in main
     │                        │                          │
     └────────────────────────┼──────────────────────────┘
                              │
                              ▼
                     plugin.NewServer(cfg, paths, db, logger, opts...)
                              │
                              ▼
                     server.Init(ctx)
                              │
                     ┌────────┴────────┐
                     │  For each plugin │
                     │   resolve key    │
                     │   look up section│
                     │   call Init(cfg) │
                     └─────────────────┘
```

**Location:** `pkg/catalog/plugin/config.go`

## sources.yaml Format

The root structure has three fields: `apiVersion`, `kind`, and a `catalogs` map keyed by plugin config key:

```yaml
apiVersion: catalog/v1alpha1
kind: CatalogSources
catalogs:

  # ── Models ─────────────────────────────────────────────────
  # Config key "models" is read by the "model" plugin via SourceKeyProvider.
  models:
    sources:
      - id: organization_ai_models
        name: "Organization AI Models"
        type: yaml
        enabled: true
        labels:
          - Organization AI
        properties:
          yamlCatalogPath: "../plugins/model/data/dev-organization-models.yaml"

      - id: validated_ai_models
        name: "Validated AI Models"
        type: yaml
        enabled: true
        labels:
          - Validated AI
        properties:
          yamlCatalogPath: "../plugins/model/data/dev-validated-models.yaml"

      - id: community_custom_models
        name: "Community and Custom Models"
        type: yaml
        enabled: true
        labels:
          - Community and Custom
        properties:
          yamlCatalogPath: "../plugins/model/data/dev-community-models.yaml"

    namedQueries:
      large_models:
        parameterCount:
          operator: ">="
          value: 70000000000

  # ── MCP Servers ────────────────────────────────────────────
  mcp:
    sources:
      - id: mcp-default
        name: "MCP Servers"
        type: yaml
        enabled: true
        labels: ["MCP Servers"]
        properties:
          yamlCatalogPath: "../plugins/mcp/data/mcp-servers.yaml"
          loaderConfigPath: "./mcp-loader-config.yaml"

  # ── Knowledge Sources ──────────────────────────────────────
  knowledge:
    sources:
      - id: knowledge-default
        name: "Knowledge Sources"
        type: yaml
        enabled: true
        labels: ["Knowledge Sources"]
        properties:
          yamlCatalogPath: "../plugins/knowledge/data/sample-knowledge-sources.yaml"

  # ── Agents ─────────────────────────────────────────────────
  agents:
    sources:
      - id: agents-default
        name: "Sample Agents"
        type: yaml
        enabled: true
        labels: ["Agents"]
        properties:
          yamlCatalogPath: "../plugins/agents/data/sample-agents.yaml"

      - id: agents-git               # Git provider example
        name: "Git Agent Repository"
        type: git
        enabled: true
        labels: ["Agents", "Git"]
        properties:
          repoUrl: "file:///sample-repos/agents-repo"
          branch: "main"
          path: "**/*.yaml"
          syncInterval: "1h"

  # ── Prompts ────────────────────────────────────────────────
  prompts:
    sources:
      - id: prompts-default
        name: "Prompt Templates"
        type: yaml
        enabled: true
        labels: ["Prompt Templates"]
        properties:
          yamlCatalogPath: "../plugins/prompts/data/sample-prompts.yaml"

  # ── Guardrails ─────────────────────────────────────────────
  guardrails:
    sources:
      - id: guardrails-default
        name: "Guardrails"
        type: yaml
        enabled: true
        labels: ["Guardrails"]
        properties:
          yamlCatalogPath: "../plugins/guardrails/data/sample-guardrails.yaml"

  # ── Policies ───────────────────────────────────────────────
  policies:
    sources:
      - id: policies-default
        name: "Policies"
        type: yaml
        enabled: true
        labels: ["Policies"]
        properties:
          yamlCatalogPath: "../plugins/policies/data/sample-policies.yaml"

  # ── Skills ─────────────────────────────────────────────────
  skills:
    sources:
      - id: skills-default
        name: "Skills"
        type: yaml
        enabled: true
        labels: ["Skills"]
        properties:
          yamlCatalogPath: "../plugins/skills/data/sample-skills.yaml"
```

### Source Type Reference

Each source's `type` field determines which provider loads data. Provider-specific parameters go in `properties`:

| Type | Description | Key Properties |
|------|-------------|----------------|
| `yaml` | Local YAML file | `yamlCatalogPath` |
| `git` | Git repository | `repoUrl`, `branch`, `path`, `syncInterval` |
| `http` | Remote HTTP/HTTPS endpoint | `url`, `headers`, `refreshInterval` |
| `hf` | Hugging Face Hub | `repoId`, `revision` |

### Include / Exclude Patterns

Every source supports glob-based item filtering:

```yaml
- id: selective-source
  name: "Filtered Models"
  type: yaml
  properties:
    yamlCatalogPath: "./all-models.yaml"
  includedItems:
    - "llama-*"
    - "mistral-*"
  excludedItems:
    - "*-deprecated"
```

`includedItems` is evaluated first, then `excludedItems` is subtracted from the result.

## Go Types

### CatalogSourcesConfig

Root configuration structure:

```go
// pkg/catalog/plugin/config.go
type CatalogSourcesConfig struct {
    // APIVersion identifies the config format version (e.g., "catalog/v1alpha1").
    APIVersion string `json:"apiVersion" yaml:"apiVersion"`

    // Kind identifies the config type (e.g., "CatalogSources").
    Kind string `json:"kind" yaml:"kind"`

    // Catalogs maps plugin config keys to their configurations.
    // The key is the plugin config key (e.g., "models", "mcp").
    Catalogs map[string]CatalogSection `json:"catalogs" yaml:"catalogs"`
}
```

### CatalogSection

Per-plugin configuration block:

```go
type CatalogSection struct {
    // Sources is the list of data sources for this catalog.
    Sources []SourceConfig `json:"sources" yaml:"sources"`

    // Labels defines custom labels available in this catalog.
    Labels []map[string]any `json:"labels,omitempty" yaml:"labels,omitempty"`

    // NamedQueries defines preset filter queries.
    NamedQueries map[string]map[string]FieldFilter `json:"namedQueries,omitempty" yaml:"namedQueries,omitempty"`
}
```

### SourceConfig

Individual data source within a catalog section:

```go
type SourceConfig struct {
    ID            string         `json:"id" yaml:"id"`             // Unique identifier
    Name          string         `json:"name" yaml:"name"`         // Human-readable display name
    Type          string         `json:"type" yaml:"type"`         // Provider type (yaml, git, http, hf)
    Enabled       *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`       // Defaults to true if nil
    Labels        []string       `json:"labels,omitempty" yaml:"labels,omitempty"`         // Tags for filtering
    Properties    map[string]any `json:"properties,omitempty" yaml:"properties,omitempty"` // Provider-specific config
    IncludedItems []string       `json:"includedItems,omitempty" yaml:"includedItems,omitempty"` // Include globs
    ExcludedItems []string       `json:"excludedItems,omitempty" yaml:"excludedItems,omitempty"` // Exclude globs
    Origin        string         `json:"-" yaml:"-"`               // Set programmatically to config file path
}
```

The `Enabled` field uses a pointer so that absence defaults to enabled:

```go
func (s SourceConfig) IsEnabled() bool {
    return s.Enabled == nil || *s.Enabled
}
```

### FieldFilter

Filter condition used by named queries:

```go
type FieldFilter struct {
    Operator string `json:"operator" yaml:"operator"` // e.g., ">=", "==", "contains"
    Value    any    `json:"value" yaml:"value"`        // Filter threshold or match value
}
```

## Config Loading and Merging

### Single File: LoadConfig

```go
func LoadConfig(path string) (*CatalogSourcesConfig, error)
```

Reads a single YAML file, strict-unmarshals it, and sets the `Origin` field on every `SourceConfig` to `path`.

### Multiple Files: LoadConfigs

```go
func LoadConfigs(paths []string) (*CatalogSourcesConfig, error)
```

Loads the first path as the base, then iteratively merges each subsequent file using `MergeConfigs`. An empty paths slice returns an empty config with an initialized `Catalogs` map.

### Merge Rules: MergeConfigs

```go
func MergeConfigs(base, override *CatalogSourcesConfig) *CatalogSourcesConfig
```

Merge priority (the **override** file wins):

```
Merge Precedence
─────────────────────────────────────────────────────────────
                                base        override    result
─────────────────────────────────────────────────────────────
APIVersion                      "v1"        "v2"        "v2"
APIVersion (empty override)     "v1"        ""          "v1"
Kind                            same as APIVersion rules
─────────────────────────────────────────────────────────────
Catalog sections                union of both maps
 Sources (same ID)              field-level merge by source ID
 Sources (new ID)               added to result
 Labels                         override wins if non-nil
 NamedQueries (same name)       field-level merge on filter keys
 NamedQueries (new name)        added to result
─────────────────────────────────────────────────────────────
```

Field-level source merge (`mergeSourceConfigs`): for two sources with the same `ID`, each non-zero field in the override replaces the base value. This lets an overlay file change just a single property (e.g., disable a source or update its `yamlCatalogPath`) without repeating the full definition.

### Example: Two-File Merge

```
base.yaml                          override.yaml
─────────────────                   ─────────────────
catalogs:                           catalogs:
  models:                             models:
    sources:                            sources:
      - id: org_models                    - id: org_models
        name: "Org Models"                  enabled: false    # disable this source
        type: yaml                        - id: hf_models     # add new source
        enabled: true                       name: "HF Models"
                                            type: hf

Result after MergeConfigs(base, override):
  models.sources = [
    { id: org_models, name: "Org Models", type: yaml, enabled: false },
    { id: hf_models,  name: "HF Models",  type: hf }
  ]
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_DSN` | (none) | Database connection string (fallback for `--db-dsn`) |
| `DATABASE_TYPE` | `postgres` | Database type: `postgres` or `mysql` (fallback for `--db-type`) |
| `CATALOG_CONFIG_STORE_MODE` | `file` | Config store backend: `file`, `k8s`, or `none` |
| `CATALOG_AUTH_MODE` | (header) | Auth mode: `jwt`, `header`, or empty (default header-based) |
| `CATALOG_JWT_PUBLIC_KEY_PATH` | (none) | Path to PEM-encoded public key for JWT verification |
| `CATALOG_JWT_ISSUER` | (none) | Expected JWT `iss` claim |
| `CATALOG_JWT_AUDIENCE` | (none) | Expected JWT `aud` claim |
| `CATALOG_JWT_ROLE_CLAIM` | `role` | JWT claim containing the user role |
| `CATALOG_JWT_OPERATOR_VALUE` | `operator` | Role value that grants operator (write) access |
| `CATALOG_CONFIG_NAMESPACE` | `default` | Kubernetes namespace for ConfigMap store |
| `CATALOG_CONFIG_CONFIGMAP_NAME` | `catalog-sources` | ConfigMap name for K8s config store |
| `CATALOG_CONFIG_CONFIGMAP_KEY` | `sources.yaml` | Data key inside the ConfigMap |

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | `:8080` | Address to listen on (host:port) |
| `--sources` | `/config/sources.yaml` | Path to catalog sources config file |
| `--db-type` | `postgres` | Database type (`postgres` or `mysql`) |
| `--db-dsn` | (none) | Database connection string |
| `--config-store` | (none) | Config store backend (overridden by `CATALOG_CONFIG_STORE_MODE`) |

Flag and environment variable resolution for database:

```
--db-dsn  ──▶  DATABASE_DSN env  ──▶  fatal error if both empty
--db-type ──▶  DATABASE_TYPE env ──▶  "postgres" default
```

Config store mode resolution:

```
--config-store flag  ──▶  CATALOG_CONFIG_STORE_MODE env  ──▶  "file" default
```

## SourceKey vs Name

A plugin's `Name()` is its identity (e.g., used in route paths and the `/api/plugins` listing). The config key that the server uses to look up the plugin's section in `sources.yaml` defaults to `Name()`, but can be overridden by implementing `SourceKeyProvider`:

```go
// pkg/catalog/plugin/plugin.go
type SourceKeyProvider interface {
    SourceKey() string
}
```

During `server.Init()`, the server resolves the config key for each plugin:

```go
configKey := p.Name()
if skp, ok := p.(SourceKeyProvider); ok {
    configKey = skp.SourceKey()
}
section, ok := s.config.Catalogs[configKey]
```

### Example: Model Plugin

The model plugin has `Name() = "model"` but reads from the `"models"` section:

```go
// catalog/plugins/model/plugin.go
const PluginName = "model"

func (p *ModelCatalogPlugin) Name() string      { return PluginName }
func (p *ModelCatalogPlugin) SourceKey() string  { return "models" }
```

This means in `sources.yaml` the key is `models` (plural), while the API routes use the plugin name `model`:

```
sources.yaml key:     catalogs.models.sources[...]
API base path:        /api/model_catalog/v1alpha1/
Plugin list name:     "model"
```

Plugins that do not implement `SourceKeyProvider` use their `Name()` directly as the config key. For example, the MCP plugin has `Name() = "mcp"` and reads from `catalogs.mcp`.

## ConfigStore Interface

The `ConfigStore` interface abstracts persistent storage for runtime configuration mutations (management API). The server uses it for the reconcile loop and management endpoints:

```go
// pkg/catalog/plugin/configstore.go
type ConfigStore interface {
    Load(ctx context.Context) (*CatalogSourcesConfig, string, error)
    Save(ctx context.Context, cfg *CatalogSourcesConfig, version string) (string, error)
    Watch(ctx context.Context) (<-chan ConfigChangeEvent, error)
    ListRevisions(ctx context.Context) ([]ConfigRevision, error)
    Rollback(ctx context.Context, version string) (*CatalogSourcesConfig, string, error)
}
```

Two implementations are provided:

| Implementation | Backend | Mode Flag | Key Features |
|----------------|---------|-----------|--------------|
| `FileConfigStore` | Local filesystem | `file` | Atomic writes, SHA-256 versioning, `.history/` revision snapshots (max 20) |
| `K8sSourceConfigStore` | Kubernetes ConfigMap | `k8s` | `RetryOnConflict`, revision annotations, max 10 revisions |

When mode is `none`, no `ConfigStore` is wired and mutations are not persisted.

## Phase 8 Configuration Options

Phase 8 introduces configuration for multi-tenancy, authorization, audit, async jobs, caching, and high availability. All options have backward-compatible defaults.

### Tenancy

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_TENANCY_MODE` | `single` | `single`: all requests scoped to `"default"` namespace. `namespace`: namespace required per request via `?namespace=` or `X-Namespace` header. |
| `CATALOG_NAMESPACES` | (none) | Comma-separated list of allowed namespaces (multi-tenant mode). |

### Authorization

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_AUTHZ_MODE` | `none` | `none`: all requests allowed. `sar`: Kubernetes SubjectAccessReview checks. |

When `sar` mode is enabled, the catalog-server creates SubjectAccessReview requests against the K8s API server using identity from `X-Remote-User` and `X-Remote-Group` headers.

### Audit

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_AUDIT_ENABLED` | `true` | Whether audit middleware captures events for management endpoints. |
| `CATALOG_AUDIT_RETENTION_DAYS` | `90` | Number of days to retain audit events before cleanup. |
| `CATALOG_AUDIT_LOG_DENIED` | `true` | Whether to record audit events for denied (403) actions. |

### Async Refresh Jobs

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_JOB_ENABLED` | `true` | Whether the async job system is active. |
| `CATALOG_JOB_CONCURRENCY` | `3` | Maximum number of concurrent refresh worker goroutines. |
| `CATALOG_JOB_MAX_RETRIES` | `3` | Maximum retry attempts per failed job. |
| `CATALOG_JOB_POLL_INTERVAL_SECONDS` | `5` | How often workers poll for new jobs (seconds). |
| `CATALOG_JOB_CLAIM_TIMEOUT_MINUTES` | `10` | Maximum time a job can be in `running` state before considered stuck and re-queued. |
| `CATALOG_JOB_RETENTION_DAYS` | `7` | How long to keep completed/failed jobs before cleanup. |

### Discovery Caching

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_CACHE_ENABLED` | `true` | Whether in-memory caching is active for discovery endpoints. |
| `CATALOG_CACHE_DISCOVERY_TTL` | `60` | TTL in seconds for `/api/plugins` cache. |
| `CATALOG_CACHE_CAPABILITIES_TTL` | `30` | TTL in seconds for `/api/plugins/{name}/capabilities` cache. |
| `CATALOG_CACHE_MAX_SIZE` | `1000` | Maximum entries per cache instance. |

### High Availability

| Variable | Default | Description |
|----------|---------|-------------|
| `CATALOG_MIGRATION_LOCK_ENABLED` | `true` | Whether to use database locking during migrations. Safe for single-replica. |
| `CATALOG_LEADER_ELECTION_ENABLED` | `false` | Whether to use Kubernetes Lease-based leader election for singleton loops. |
| `CATALOG_LEADER_LEASE_NAME` | `catalog-server-leader` | Name of the Kubernetes Lease resource. |
| `CATALOG_LEADER_LEASE_NAMESPACE` | from `POD_NAMESPACE` or `catalog-system` | Namespace of the Lease resource. |
| `CATALOG_LEADER_LEASE_DURATION` | `15` | Lease duration in seconds. |
| `CATALOG_LEADER_RENEW_DEADLINE` | `10` | Lease renew deadline in seconds. |
| `CATALOG_LEADER_RETRY_PERIOD` | `2` | Leader election retry period in seconds. |
| `POD_NAME` | hostname | Instance identity for leader election (set via Downward API). |
| `POD_NAMESPACE` | `catalog-system` | Namespace for the Lease resource (set via Downward API). |

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/config.go` | `CatalogSourcesConfig`, `CatalogSection`, `SourceConfig`, `FieldFilter` types; `LoadConfig`, `LoadConfigs`, `MergeConfigs` functions |
| `pkg/catalog/plugin/config_test.go` | Unit tests for config loading and merging |
| `pkg/catalog/plugin/configstore.go` | `ConfigStore` interface, `ConfigRevision`, `ConfigChangeEvent` types |
| `pkg/catalog/plugin/file_config_store.go` | File-backed `ConfigStore` with atomic writes and revision history |
| `pkg/catalog/plugin/k8s_config_store.go` | Kubernetes ConfigMap-backed `ConfigStore` with `RetryOnConflict` |
| `cmd/catalog-server/main.go` | Server entry point; flag parsing, env var resolution, config store setup |
| `catalog/config/sources.yaml` | Default development configuration file |
| `pkg/tenancy/config.go` | `TenancyMode` type (Phase 8) |
| `pkg/authz/config.go` | `AuthzMode` type (Phase 8) |
| `pkg/audit/config.go` | `AuditConfig` with env var loading (Phase 8) |
| `pkg/jobs/config.go` | `JobConfig` with env var loading (Phase 8) |
| `pkg/cache/config.go` | `CacheConfig` with env var loading (Phase 8) |
| `pkg/ha/config.go` | `HAConfig` with env var loading (Phase 8) |

---

[Back to Plugin Framework](./README.md) | [Prev: Creating Plugins](./creating-plugins.md)
