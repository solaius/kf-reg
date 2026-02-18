# Asset-Type Plugins

## Overview

The catalog-of-catalogs system ships with **eight** asset-type plugins. Two (Model, MCP) are the original Phase 1-3 plugins documented separately. This document covers the remaining six plugins introduced in Phase 5 and Phase 6, plus the reusable Git provider used by the Agents plugin.

Every plugin listed below follows the same structural pattern: it implements `CatalogPlugin` and opts into the Phase 5 universal asset framework by also implementing `CapabilitiesV2Provider`, `AssetMapperProvider`, and `ActionProvider`. This means each one is automatically rendered in the generic UI and CLI with zero frontend or CLI code changes.

```
+-----------------------------------------------------------------------+
|                       catalog-server process                          |
|                                                                       |
|  Phase 5 (Knowledge)     Phase 6 (5 new plugins + Git provider)      |
|  +-------------------+   +-------------------+  +-----------------+  |
|  | knowledge         |   | prompts           |  | agents          |  |
|  | KnowledgeSource   |   | PromptTemplate    |  | Agent           |  |
|  | yaml              |   | yaml              |  | yaml + git      |  |
|  +-------------------+   +-------------------+  +-----------------+  |
|                           +-------------------+  +-----------------+  |
|                           | guardrails        |  | policies        |  |
|                           | Guardrail         |  | Policy          |  |
|                           | yaml              |  | yaml            |  |
|                           +-------------------+  +-----------------+  |
|                           +-------------------+                       |
|                           | skills            |                       |
|                           | Skill             |                       |
|                           | yaml              |                       |
|                           +-------------------+                       |
+-----------------------------------------------------------------------+
```

All six plugins share the same set of implemented optional interfaces:

| Interface | Implemented |
|-----------|-------------|
| `CatalogPlugin` | Yes (required) |
| `BasePathProvider` | Yes |
| `CapabilitiesProvider` (V1) | Yes |
| `CapabilitiesV2Provider` | Yes |
| `SourceManager` | Yes |
| `RefreshProvider` | Yes |
| `DiagnosticsProvider` | Yes |
| `UIHintsProvider` | Yes |
| `CLIHintsProvider` | Yes |
| `AssetMapperProvider` | Yes |
| `ActionProvider` | Yes |

---

## Knowledge Sources Plugin (Phase 5)

**Package:** `catalog/plugins/knowledge/`
**Plugin Name:** `knowledge`
**API Version:** `v1alpha1`
**Base Path:** `/api/knowledge_catalog/v1alpha1`
**Entity Kind:** `KnowledgeSource`
**Description:** Knowledge source catalog for documents, vector stores, and graph stores

The Knowledge Sources plugin was the first proof of the universal asset framework. It appeared in the generic UI navigation and the CLI entity listing with absolutely no changes to `clients/ui/frontend/` or `cmd/catalogctl/`. This validated Phase 5's core claim: zero-code-change extensibility.

### Entity Schema: KnowledgeSourceEntry

```go
// catalog/plugins/knowledge/plugin.go
type KnowledgeSourceEntry struct {
    Name             string         `yaml:"name" json:"name"`
    ExternalId       string         `yaml:"externalId" json:"externalId,omitempty"`
    Description      *string        `yaml:"description" json:"description,omitempty"`
    SourceType       *string        `yaml:"sourceType" json:"sourceType,omitempty"`
    Location         *string        `yaml:"location" json:"location,omitempty"`
    ContentType      *string        `yaml:"contentType" json:"contentType,omitempty"`
    Provider         *string        `yaml:"provider" json:"provider,omitempty"`
    Status           *string        `yaml:"status" json:"status,omitempty"`
    DocumentCount    *int32         `yaml:"documentCount" json:"documentCount,omitempty"`
    VectorDimensions *int32         `yaml:"vectorDimensions" json:"vectorDimensions,omitempty"`
    IndexType        *string        `yaml:"indexType" json:"indexType,omitempty"`
    CustomProperties map[string]any `yaml:"customProperties" json:"customProperties,omitempty"`
    SourceId         string         `yaml:"-" json:"sourceId,omitempty"`
}
```

### Data Providers

| Provider | Source Type | Properties |
|----------|-----------|------------|
| YAML | `yaml` | `yamlCatalogPath` or `path` -- path to a YAML file containing `knowledgesources:` array |

### Routes

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/knowledgesources` | List all knowledge sources (supports `filterQuery`) |
| `GET` | `/knowledgesources/{name}` | Get a single knowledge source by name |

### Capabilities V2

```
Display Name:  Knowledge Sources
Icon:          database
Entity Kind:   KnowledgeSource (plural: knowledgesources)

List Columns:  name, sourceType, provider, status, documentCount, contentType
Filter Fields: name (text), sourceType (select), provider (text), status (select)
Detail Sections: Overview, Connection, Statistics

Source Types:  yaml
Actions:       tag, annotate, deprecate (asset scope) + refresh (source scope)
```

### Asset Mapper

Maps `KnowledgeSourceEntry` to `AssetResource` with kind `KnowledgeSource`. The spec map includes: `sourceType`, `location`, `contentType`, `provider`, `status`, `documentCount`, `vectorDimensions`, `indexType`.

### Actions

| Action | Scope | Dry Run | Idempotent | Description |
|--------|-------|---------|------------|-------------|
| `refresh` | source | No | Yes | Reload entries from YAML source |
| `tag` | asset | Yes | Yes | Add or remove tags via overlay store |
| `annotate` | asset | Yes | Yes | Add or update annotations via overlay store |
| `deprecate` | asset | Yes | Yes | Mark entity as deprecated via overlay store |

---

## Prompt Templates Plugin (Phase 6)

**Package:** `catalog/plugins/prompts/`
**Plugin Name:** `prompts`
**API Version:** `v1alpha1`
**Base Path:** `/api/prompts_catalog/v1alpha1`
**Entity Kind:** `PromptTemplate`
**Description:** Prompt template catalog for reusable AI prompts

### Entity Schema: PromptTemplateEntry

```go
// catalog/plugins/prompts/plugin.go
type PromptTemplateEntry struct {
    Name             string           `yaml:"name" json:"name"`
    ExternalId       string           `yaml:"externalId" json:"externalId,omitempty"`
    Description      *string          `yaml:"description" json:"description,omitempty"`
    Format           *string          `yaml:"format" json:"format,omitempty"`
    Template         *string          `yaml:"template" json:"template,omitempty"`
    ParametersSchema map[string]any   `yaml:"parametersSchema" json:"parametersSchema,omitempty"`
    OutputSchema     map[string]any   `yaml:"outputSchema" json:"outputSchema,omitempty"`
    ModelConstraints map[string]any   `yaml:"modelConstraints" json:"modelConstraints,omitempty"`
    Examples         []map[string]any `yaml:"examples" json:"examples,omitempty"`
    TaskTags         []string         `yaml:"taskTags" json:"taskTags,omitempty"`
    Version          *string          `yaml:"version" json:"version,omitempty"`
    Author           *string          `yaml:"author" json:"author,omitempty"`
    License          *string          `yaml:"license" json:"license,omitempty"`
    CustomProperties map[string]any   `yaml:"customProperties" json:"customProperties,omitempty"`
    SourceId         string           `yaml:"-" json:"sourceId,omitempty"`
}
```

### Data Providers

| Provider | Source Type | Properties |
|----------|-----------|------------|
| YAML | `yaml` | `yamlCatalogPath` or `path` -- path to YAML with `prompttemplates:` array |

### Routes

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/prompttemplates` | List all prompt templates (supports `filterQuery`) |
| `GET` | `/prompttemplates/{name}` | Get a single prompt template by name |

### Capabilities V2

```
Display Name:  Prompt Templates
Icon:          edit
Entity Kind:   PromptTemplate (plural: prompttemplates)

Filter Fields: name (text), format (select), taskTags (text), version (text),
               author (text)
Source Types:  yaml
Actions:       tag, annotate, deprecate (asset) + refresh (source)
```

### Asset Mapper

Maps `PromptTemplateEntry` to `AssetResource` with kind `PromptTemplate`. The spec map includes: `format`, `template`, `parametersSchema`, `outputSchema`, `modelConstraints`, `examples`, `taskTags`, `version`, `author`, `license`.

---

## Agents Catalog Plugin (Phase 6)

**Package:** `catalog/plugins/agents/`
**Plugin Name:** `agents`
**API Version:** `v1alpha1`
**Base Path:** `/api/agents_catalog/v1alpha1`
**Entity Kind:** `Agent`
**Description:** Agent catalog for AI agents and multi-agent orchestrations

The Agents plugin is the first plugin to use both YAML and Git providers, making it suitable for team-managed agent catalogs stored in version-controlled repositories.

### Entity Schema: AgentEntry

```go
// catalog/plugins/agents/plugin.go
type AgentEntry struct {
    Name             string           `yaml:"name" json:"name"`
    ExternalId       string           `yaml:"externalId" json:"externalId,omitempty"`
    Description      *string          `yaml:"description" json:"description,omitempty"`
    AgentType        *string          `yaml:"agentType" json:"agentType,omitempty"`
    Instructions     *string          `yaml:"instructions" json:"instructions,omitempty"`
    Version          *string          `yaml:"version" json:"version,omitempty"`
    ModelConfig      map[string]any   `yaml:"modelConfig" json:"modelConfig,omitempty"`
    Tools            []map[string]any `yaml:"tools" json:"tools,omitempty"`
    Knowledge        []map[string]any `yaml:"knowledge" json:"knowledge,omitempty"`
    Guardrails       []map[string]any `yaml:"guardrails" json:"guardrails,omitempty"`
    Policies         []map[string]any `yaml:"policies" json:"policies,omitempty"`
    PromptRefs       []map[string]any `yaml:"promptRefs" json:"promptRefs,omitempty"`
    Dependencies     []map[string]any `yaml:"dependencies" json:"dependencies,omitempty"`
    InputSchema      map[string]any   `yaml:"inputSchema" json:"inputSchema,omitempty"`
    OutputSchema     map[string]any   `yaml:"outputSchema" json:"outputSchema,omitempty"`
    Examples         []map[string]any `yaml:"examples" json:"examples,omitempty"`
    Author           *string          `yaml:"author" json:"author,omitempty"`
    License          *string          `yaml:"license" json:"license,omitempty"`
    Category         *string          `yaml:"category" json:"category,omitempty"`
    CustomProperties map[string]any   `yaml:"customProperties" json:"customProperties,omitempty"`
    SourceId         string           `yaml:"-" json:"sourceId,omitempty"`
}
```

### Data Providers

| Provider | Source Type | Properties |
|----------|-----------|------------|
| YAML | `yaml` | `yamlCatalogPath` or `path` -- path to YAML with `agents:` array |
| Git | `git` | `repoUrl`, `branch`, `path` (glob), `authToken`, `syncInterval` |

The Agents plugin is the first consumer of the generic Git provider (`pkg/catalog/providers/git/`). Git sources are loaded at startup via a shallow clone, and background goroutines periodically pull to detect new commits. Each Git source gets its own cancellable context; calling `Stop()` on the plugin cancels all Git sync goroutines.

### Routes

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/agents` | List all agents (supports `filterQuery`) |
| `GET` | `/agents/{name}` | Get a single agent by name |

### Capabilities V2

```
Display Name:  Agents
Icon:          robot
Entity Kind:   Agent (plural: agents)

List Columns:  name, agentType, category, version, author
Filter Fields: name (text), agentType (select: conversational, task_oriented,
               router, planner, executor, evaluator), category (text)
Detail Sections: Overview, Instructions, Model Configuration, Tools & Knowledge,
                 Guardrails & Policies, Dependencies, Input/Output

Source Types:  yaml, git
Actions:       tag, annotate, deprecate (asset) + refresh (source)
```

### Cross-Asset Linking

Agents are the hub entity in the catalog-of-catalogs graph. An agent can reference assets from every other plugin. The asset mapper extracts these references into `AssetLinks.Related`:

```
Agent cross-reference fields and their target kinds:

  tools[].skillRef           --> Skill
  tools[].mcpToolRef         --> McpServer
  knowledge[].knowledgeSourceRef --> KnowledgeSource
  guardrails[].guardrailRef  --> Guardrail
  policies[].policyRef       --> Policy
  promptRefs[].promptTemplateRef --> PromptTemplate
  dependencies[].agentRef    --> Agent (self-referencing for multi-agent)
```

```
                        +-------------------+
                        |      Agent        |
                        +--------+----------+
                                 |
         +-----------+-----------+-----------+-----------+-----------+
         |           |           |           |           |           |
    +----+----+ +----+----+ +---+-----+ +---+-----+ +--+------+ +--+------+
    |  Skill  | |McpServer| |Knowledge| |Guardrail| | Policy  | | Prompt  |
    |         | |         | | Source  | |         | |         | |Template |
    +---------+ +---------+ +---------+ +---------+ +---------+ +---------+
```

### Asset Mapper

Maps `AgentEntry` to `AssetResource` with kind `Agent`. The spec map includes: `agentType`, `instructions`, `modelConfig`, `tools`, `knowledge`, `guardrails`, `policies`, `promptRefs`, `dependencies`, `inputSchema`, `outputSchema`, `examples`, `category`. Cross-asset links are populated in `status.links.related[]`.

---

## Guardrails Plugin (Phase 6)

**Package:** `catalog/plugins/guardrails/`
**Plugin Name:** `guardrails`
**API Version:** `v1alpha1`
**Base Path:** `/api/guardrails_catalog/v1alpha1`
**Entity Kind:** `Guardrail`
**Description:** Guardrail catalog for AI safety and content moderation rules

### Entity Schema: GuardrailEntry

```go
// catalog/plugins/guardrails/plugin.go
type GuardrailEntry struct {
    Name             string         `yaml:"name" json:"name"`
    ExternalId       string         `yaml:"externalId" json:"externalId,omitempty"`
    Description      *string        `yaml:"description" json:"description,omitempty"`
    GuardrailType    *string        `yaml:"guardrailType" json:"guardrailType,omitempty"`
    EnforcementStage *string        `yaml:"enforcementStage" json:"enforcementStage,omitempty"`
    RiskCategories   []string       `yaml:"riskCategories" json:"riskCategories,omitempty"`
    EnforcementMode  *string        `yaml:"enforcementMode" json:"enforcementMode,omitempty"`
    Modalities       []string       `yaml:"modalities" json:"modalities,omitempty"`
    ConfigRef        map[string]any `yaml:"configRef" json:"configRef,omitempty"`
    Version          *string        `yaml:"version" json:"version,omitempty"`
    Author           *string        `yaml:"author" json:"author,omitempty"`
    License          *string        `yaml:"license" json:"license,omitempty"`
    CustomProperties map[string]any `yaml:"customProperties" json:"customProperties,omitempty"`
    SourceId         string         `yaml:"-" json:"sourceId,omitempty"`
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `guardrailType` | Type of guardrail (e.g., `content_filter`, `pii_detector`, `topic_restriction`) |
| `enforcementStage` | When the guardrail applies: `pre_inference`, `post_inference`, `both` |
| `riskCategories` | Risk categories addressed (e.g., `hate_speech`, `violence`, `pii_exposure`) |
| `enforcementMode` | How strictly enforced: `block`, `warn`, `log` |
| `modalities` | Applicable modalities: `text`, `image`, `audio`, `video` |
| `configRef` | Reference to external framework configuration (e.g., NeMo Guardrails YAML) |

### Data Providers

| Provider | Source Type | Properties |
|----------|-----------|------------|
| YAML | `yaml` | `yamlCatalogPath` or `path` -- path to YAML with `guardrails:` array |

### Routes

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/guardrails` | List all guardrails (supports `filterQuery`) |
| `GET` | `/guardrails/{name}` | Get a single guardrail by name |

### Capabilities V2

```
Display Name:  Guardrails
Icon:          shield
Entity Kind:   Guardrail (plural: guardrails)

Filter Fields: name (text), guardrailType (select), enforcementStage (select),
               enforcementMode (select), riskCategories (text), modalities (text)
Source Types:  yaml
Actions:       tag, annotate, deprecate (asset) + refresh (source)
```

### Asset Mapper

Maps `GuardrailEntry` to `AssetResource` with kind `Guardrail`. The spec map includes: `guardrailType`, `enforcementStage`, `riskCategories`, `enforcementMode`, `modalities`, `configRef`.

---

## Policies Plugin (Phase 6)

**Package:** `catalog/plugins/policies/`
**Plugin Name:** `policies`
**API Version:** `v1alpha1`
**Base Path:** `/api/policies_catalog/v1alpha1`
**Entity Kind:** `Policy`
**Description:** Policy catalog for AI governance and access control

### Entity Schema: PolicyEntry

```go
// catalog/plugins/policies/plugin.go
type PolicyEntry struct {
    Name             string         `yaml:"name" json:"name"`
    ExternalId       string         `yaml:"externalId" json:"externalId,omitempty"`
    Description      *string        `yaml:"description" json:"description,omitempty"`
    PolicyType       *string        `yaml:"policyType" json:"policyType,omitempty"`
    Language         *string        `yaml:"language" json:"language,omitempty"`
    BundleRef        *string        `yaml:"bundleRef" json:"bundleRef,omitempty"`
    Entrypoint       *string        `yaml:"entrypoint" json:"entrypoint,omitempty"`
    EnforcementScope *string        `yaml:"enforcementScope" json:"enforcementScope,omitempty"`
    EnforcementMode  *string        `yaml:"enforcementMode" json:"enforcementMode,omitempty"`
    InputSchema      map[string]any `yaml:"inputSchema" json:"inputSchema,omitempty"`
    Version          *string        `yaml:"version" json:"version,omitempty"`
    Author           *string        `yaml:"author" json:"author,omitempty"`
    License          *string        `yaml:"license" json:"license,omitempty"`
    CustomProperties map[string]any `yaml:"customProperties" json:"customProperties,omitempty"`
    SourceId         string         `yaml:"-" json:"sourceId,omitempty"`
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `policyType` | Category of policy (e.g., `access_control`, `usage_quota`, `data_governance`) |
| `language` | Policy language: `rego` (OPA), `cel`, `python`, `json_schema` |
| `bundleRef` | Reference to an OPA bundle or external policy package |
| `entrypoint` | Entry point within the policy bundle (e.g., `data.authz.allow`) |
| `enforcementScope` | Scope of enforcement: `model`, `agent`, `pipeline`, `global` |
| `enforcementMode` | Enforcement behavior: `enforce`, `audit`, `dry_run` |
| `inputSchema` | JSON Schema describing the input decision request |

### Data Providers

| Provider | Source Type | Properties |
|----------|-----------|------------|
| YAML | `yaml` | `yamlCatalogPath` or `path` -- path to YAML with `policies:` array |

### Routes

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/policies` | List all policies (supports `filterQuery`) |
| `GET` | `/policies/{name}` | Get a single policy by name |

### Capabilities V2

```
Display Name:  Policies
Icon:          gavel
Entity Kind:   Policy (plural: policies)

Filter Fields: name (text), policyType (select), language (select),
               enforcementScope (select), enforcementMode (select)
Source Types:  yaml
Actions:       tag, annotate, deprecate (asset) + refresh (source)
```

### Asset Mapper

Maps `PolicyEntry` to `AssetResource` with kind `Policy`. The spec map includes: `policyType`, `language`, `bundleRef`, `entrypoint`, `enforcementScope`, `enforcementMode`, `inputSchema`.

---

## Skills Plugin (Phase 6)

**Package:** `catalog/plugins/skills/`
**Plugin Name:** `skills`
**API Version:** `v1alpha1`
**Base Path:** `/api/skills_catalog/v1alpha1`
**Entity Kind:** `Skill`
**Description:** Skill catalog for tools, operations, and executable actions

### Entity Schema: SkillEntry

```go
// catalog/plugins/skills/plugin.go
type SkillEntry struct {
    Name             string         `yaml:"name" json:"name"`
    ExternalId       string         `yaml:"externalId" json:"externalId,omitempty"`
    Description      *string        `yaml:"description" json:"description,omitempty"`
    SkillType        *string        `yaml:"skillType" json:"skillType,omitempty"`
    InputSchema      map[string]any `yaml:"inputSchema" json:"inputSchema,omitempty"`
    OutputSchema     map[string]any `yaml:"outputSchema" json:"outputSchema,omitempty"`
    Execution        map[string]any `yaml:"execution" json:"execution,omitempty"`
    Safety           map[string]any `yaml:"safety" json:"safety,omitempty"`
    RateLimit        map[string]any `yaml:"rateLimit" json:"rateLimit,omitempty"`
    TimeoutSeconds   *int32         `yaml:"timeoutSeconds" json:"timeoutSeconds,omitempty"`
    RetryPolicy      map[string]any `yaml:"retryPolicy" json:"retryPolicy,omitempty"`
    Compatibility    map[string]any `yaml:"compatibility" json:"compatibility,omitempty"`
    Version          *string        `yaml:"version" json:"version,omitempty"`
    Author           *string        `yaml:"author" json:"author,omitempty"`
    License          *string        `yaml:"license" json:"license,omitempty"`
    CustomProperties map[string]any `yaml:"customProperties" json:"customProperties,omitempty"`
    SourceId         string         `yaml:"-" json:"sourceId,omitempty"`
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `skillType` | Execution type: `python`, `openapi_operation`, `mcp_tool`, `shell`, `http` |
| `execution` | Executor configuration. For `mcp_tool` type: `{ executorType, mcpServerRef, toolName }` |
| `safety` | Safety metadata: `{ requiresApproval, riskLevel, networkAccessRequired }` |
| `rateLimit` | Rate limiting config: `{ maxCallsPerMinute, maxCallsPerHour }` |
| `timeoutSeconds` | Maximum execution time in seconds |
| `retryPolicy` | Retry behavior: `{ maxRetries, backoffSeconds }` |
| `compatibility` | Agent framework compatibility: `{ frameworks: [...] }` |

Skills of type `mcp_tool` link to MCP servers via the `execution.mcpServerRef` field, bridging the Skills plugin to the MCP plugin's entity namespace.

### Data Providers

| Provider | Source Type | Properties |
|----------|-----------|------------|
| YAML | `yaml` | `yamlCatalogPath` or `path` -- path to YAML with `skills:` array |

### Routes

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/skills` | List all skills (supports `filterQuery`) |
| `GET` | `/skills/{name}` | Get a single skill by name |

### Capabilities V2

```
Display Name:  Skills
Icon:          wrench
Entity Kind:   Skill (plural: skills)

Filter Fields: name (text), skillType (select: python, openapi_operation,
               mcp_tool, shell, http), version (text), author (text),
               riskLevel (select: low, medium, high)
Source Types:  yaml
Actions:       tag, annotate, deprecate (asset) + refresh (source)
```

### Asset Mapper

Maps `SkillEntry` to `AssetResource` with kind `Skill`. The spec map includes: `skillType`, `inputSchema`, `outputSchema`, `execution`, `safety`, `rateLimit`, `timeoutSeconds`, `retryPolicy`, `compatibility`.

---

## Git Provider

**Package:** `pkg/catalog/providers/git/`
**Source:** `pkg/catalog/providers/git/provider.go`

The Git provider is a pure Go implementation (using `go-git/go-git/v5`) that clones a Git repository, discovers files matching a glob pattern, and parses them into typed entity records. It has no dependency on an external `git` binary.

### Design

The provider is generic over entity type `E` and artifact type `A` using Go generics:

```go
// pkg/catalog/providers/git/provider.go
type Config[E any, A any] struct {
    RepoURLKey        string
    BranchKey         string
    PathKey           string
    AuthTokenKey      string
    SyncIntervalKey   string
    Parse             func(data []byte) ([]catalog.Record[E, A], error)
    Filter            func(record catalog.Record[E, A]) bool
    Logger            Logger
    DefaultBranch     string               // default: "main"
    DefaultSyncInterval time.Duration      // default: 1 hour
    ShallowClone      *bool                // default: true
}
```

### Features

| Feature | Detail |
|---------|--------|
| Shallow clones | `Depth: 1` by default to minimize bandwidth and disk usage |
| Auth token support | HTTP basic auth via `gogithttp.BasicAuth` (username `"git"`, password = token) |
| Branch selection | Configurable per-source via `branch` property (default: `main`) |
| Glob pattern file discovery | Walks the clone directory, matches files with `**/*.yaml` or custom pattern |
| Commit SHA tracking | `LastCommit()` returns HEAD SHA for provenance and change detection |
| Periodic sync | Background goroutine pulls on a configurable interval (property: `syncInterval`, default: `1h`) |
| Change detection | After each pull, compares HEAD SHA; only re-emits records if commit changed |
| Batch markers | Sends a zero-value record to signal end of each batch |
| Automatic cleanup | Removes the temp clone directory when the context is canceled |

### Lifecycle

```
NewProvider(config, source, reldir)     Parse config properties, set defaults
       |
       v
Records(ctx)                            Clone repo (shallow), read + parse files
       |
       +---> emit initial batch ---> channel
       |
       v
watchAndReload(ctx)                     Ticker loop at syncInterval:
       |                                  1. git pull
       |                                  2. Compare HEAD SHA
       |                                  3. If changed: re-read files, emit batch
       v
ctx.Done()                              Cleanup temp directory
```

### Source Properties

| Property | Default Key | Default Value | Description |
|----------|------------|---------------|-------------|
| Repository URL | `repoUrl` | (required) | Git HTTPS URL to clone |
| Branch | `branch` | `main` | Branch to track |
| File pattern | `path` | `**/*.yaml` | Glob pattern for entity files |
| Auth token | `authToken` | (none) | Bearer token for private repos |
| Sync interval | `syncInterval` | `1h` | Go duration string (e.g., `5m`, `1h`) |
| Shallow clone | `shallowClone` | `true` | Whether to use `depth=1` |

### Usage in the Agents Plugin

The Agents plugin creates a Git provider per `git`-type source:

```go
gitCfg := gitprovider.Config[AgentEntry, any]{
    Parse:  parseAgentRecords,
    Logger: &slogGitLogger{logger: logger},
}
provider, err := gitprovider.NewProvider(gitCfg, catalogSource, "")
ch, err := provider.Records(providerCtx)
```

A `context.CancelFunc` for each Git source is stored in `AgentPlugin.gitCancels`. Calling `Stop()` on the plugin cancels all provider goroutines, which triggers cleanup of cloned directories.

---

## Common Plugin File Structure

Every asset-type plugin follows the same directory layout:

```
catalog/plugins/{name}/
+-- register.go        # init() function calling plugin.Register()
+-- plugin.go          # CatalogPlugin implementation, entity struct, YAML loader
+-- asset_mapper.go    # AssetMapperProvider: maps native entity to AssetResource
+-- actions.go         # ActionProvider: tag, annotate, deprecate, refresh
+-- management.go      # SourceManager, RefreshProvider, DiagnosticsProvider,
|                      # CapabilitiesProvider, CapabilitiesV2Provider,
|                      # UIHintsProvider, CLIHintsProvider, BasePathProvider
+-- data/              # Sample YAML data files
    +-- sample-{entities}.yaml
```

### Registration Pattern

All plugins use the same self-registration via Go `init()`:

```go
// catalog/plugins/{name}/register.go
package {name}

import "github.com/kubeflow/model-registry/pkg/catalog/plugin"

func init() {
    plugin.Register(&{PluginType}{})
}
```

### Common Actions

All six plugins support the same four actions via `ActionProvider`:

| Action | Scope | Mechanism |
|--------|-------|-----------|
| `tag` | asset | `BuiltinActionHandler` with `OverlayStore` (DB-backed) |
| `annotate` | asset | `BuiltinActionHandler` with `OverlayStore` (DB-backed) |
| `deprecate` | asset | `BuiltinActionHandler` with `OverlayStore` (DB-backed) |
| `refresh` | source | Plugin-specific: reloads YAML (or re-clones Git) |

### Common Filter Support

All plugins support the same `filterQuery` syntax on list endpoints:

- Equality: `field='value'`
- Inequality: `field!='value'`
- Pattern: `field LIKE '%value%'`
- Conjunction: `field1='a' AND field2='b'`

---

## Key Files

| File | Purpose |
|------|---------|
| `catalog/plugins/knowledge/plugin.go` | Knowledge Sources plugin (Phase 5), entity struct, YAML loader |
| `catalog/plugins/knowledge/management.go` | V2 capabilities, source management, refresh, diagnostics, UI/CLI hints |
| `catalog/plugins/knowledge/asset_mapper.go` | KnowledgeSource to AssetResource mapper |
| `catalog/plugins/knowledge/actions.go` | Action handler (tag, annotate, deprecate, refresh) |
| `catalog/plugins/prompts/plugin.go` | Prompt Templates plugin, PromptTemplateEntry struct |
| `catalog/plugins/agents/plugin.go` | Agents plugin with YAML + Git provider support |
| `catalog/plugins/agents/asset_mapper.go` | Agent to AssetResource mapper with cross-asset link extraction |
| `catalog/plugins/guardrails/plugin.go` | Guardrails plugin, GuardrailEntry struct |
| `catalog/plugins/policies/plugin.go` | Policies plugin, PolicyEntry struct |
| `catalog/plugins/skills/plugin.go` | Skills plugin, SkillEntry struct with execution config |
| `pkg/catalog/providers/git/provider.go` | Generic Git provider using go-git (pure Go, no binary) |

---

[Back to Plugins](./README.md) | [Prev: Model and MCP Plugins](./model-and-mcp-plugins.md)
