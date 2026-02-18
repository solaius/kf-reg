# Catalog Plugins

This section documents the concrete plugin implementations that provide specific asset-type catalogs.

## Contents

| Document | Description |
|----------|-------------|
| [Model and MCP Plugins](./model-and-mcp-plugins.md) | The two original plugins (Phase 1-3) |
| [Asset Type Plugins](./asset-type-plugins.md) | Knowledge, Prompts, Agents, Guardrails, Policies, Skills, and the Git provider |

## Plugin Inventory

| Plugin | Entity Kind | Source Types | Phase | Status |
|--------|------------|-------------|-------|--------|
| **model** | CatalogModel | yaml, hf | Phase 1 | Production |
| **mcp** | McpServer | yaml | Phase 1 | Production |
| **knowledge** | KnowledgeSource | yaml | Phase 5 | Production |
| **prompts** | PromptTemplate | yaml | Phase 6 | Production |
| **agents** | Agent | yaml, git | Phase 6 | Production |
| **guardrails** | Guardrail | yaml | Phase 6 | Production |
| **policies** | Policy | yaml | Phase 6 | Production |
| **skills** | Skill | yaml | Phase 6 | Production |

## Provider Types

| Provider | Description | Used By |
|----------|-------------|---------|
| **yaml** | Load entities from local YAML files with hot-reload | All plugins |
| **hf** | Query HuggingFace Hub API | Model plugin |
| **git** | Clone Git repositories, discover files via glob patterns | Agents plugin |
| **http** | Fetch entity data from HTTP endpoints | Available but not yet wired |

## Cross-Asset Linking

Plugins can reference entities from other plugins via `AssetLinks` in the `AssetResource` envelope. The Agents plugin is the primary example:

```
Agent "customer-support-agent"
  ├── references → Prompt "support-template"
  ├── references → Guardrail "content-safety"
  ├── references → Policy "data-access-policy"
  ├── references → KnowledgeSource "product-docs"
  └── references → Skill "ticket-lookup"
```

These cross-references are stored as `LinkRef` objects in `AssetStatus.Links.Related` and rendered in the generic detail view.

---

[Back to Catalog of Catalogs](../README.md)
