# Catalog of Catalogs Documentation

Comprehensive documentation for the catalog-of-catalogs platform -- a plugin-based architecture that hosts multiple AI asset catalogs (models, MCP servers, knowledge sources, agents, prompts, guardrails, policies, skills) in a single unified server process.

## Overview

The catalog-of-catalogs extends the Kubeflow Model Registry with a **universal asset framework** where new asset-type plugins appear in the UI and CLI with zero code changes. The system provides:

- **Plugin Architecture** - Self-registering plugins with failure isolation and unified HTTP serving
- **8 Asset Catalogs** - Models, MCP Servers, Knowledge Sources, Agents, Prompts, Guardrails, Policies, Skills
- **Source Management** - Runtime CRUD with persistent configuration, validation, and rollback
- **Universal Asset Framework** - Capabilities-driven discovery, asset mapping, and action execution
- **Generic Clients** - React UI and CLI that render any plugin from its capabilities document
- **Conformance Suite** - Ensures all plugins meet the universal framework contract
- **Multi-Tenancy** - Namespace-based tenant isolation with server-side enforcement
- **Enterprise Authorization** - Kubernetes SAR-based RBAC with identity extraction and caching
- **Audit Logging** - Structured audit events for all management actions with retention
- **Async Refresh Jobs** - Database-backed job queue with worker pool and retry logic
- **HA Readiness** - Migration locking, leader election, safe multi-replica operation

**Status:** Feature Branch (`plugin-catalog-gen`)

## Architecture

```
┌───────────────────────────────────────────────────────────────┐
│  Clients                                                       │
│  ┌─────────────────────┐  ┌──────────────────────────────┐   │
│  │  React UI            │  │  catalogctl CLI               │   │
│  │  (GenericCatalog)    │  │  (dynamic subcommands)        │   │
│  └─────────┬───────────┘  └──────────────┬───────────────┘   │
│            │ REST                         │ REST (direct)      │
│  ┌─────────▼───────────┐                 │                    │
│  │  BFF Layer (:4000)  │                 │                    │
│  │  Proxy + transform  │                 │                    │
│  └─────────┬───────────┘                 │                    │
└────────────┼─────────────────────────────┼────────────────────┘
             │                             │
┌────────────▼─────────────────────────────▼────────────────────┐
│  Catalog Server (:8080)                                        │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  /api/plugins          Plugin discovery + V2 capabilities │ │
│  │  /healthz /livez /readyz  Health endpoints                │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Plugin Framework                                         │ │
│  │  ┌───────┐ ┌─────┐ ┌──────┐ ┌──────┐ ┌───────┐         │ │
│  │  │ Model │ │ MCP │ │ Know │ │Agent │ │Prompt │ ...     │ │
│  │  └───┬───┘ └──┬──┘ └──┬───┘ └──┬───┘ └───┬───┘         │ │
│  │      │   Capabilities + Actions + AssetMapper             │ │
│  │      │   OverlayStore (tags, annotations, lifecycle)      │ │
│  └──────┼────────────────────────────────────────────────────┘ │
│         │                                                      │
│  ┌──────▼────────────────────────────────────────────────────┐ │
│  │  Middleware Stack (Phase 8)                                │ │
│  │  CORS -> Tenancy -> Identity -> Authz -> Audit -> Cache   │ │
│  └──────┬────────────────────────────────────────────────────┘ │
│         │                                                      │
│  ┌──────▼────────────────────────────────────────────────────┐ │
│  │  ConfigStore (file / k8s)    │  PostgreSQL (GORM)         │ │
│  │  Async Jobs (refresh_jobs)   │  Audit Events              │ │
│  └───────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

## Documentation Structure

```
prod-doc/catalog_of_catalogs/
├── README.md                            # This file
│
├── plugin-framework/                    # Plugin system architecture
│   ├── README.md
│   ├── architecture.md                  # Core interfaces, registry, server lifecycle
│   ├── creating-plugins.md              # Step-by-step plugin creation guide
│   └── configuration.md                 # sources.yaml, env vars, CLI flags
│
├── source-management/                   # Source CRUD, persistence, validation
│   ├── README.md
│   ├── config-stores.md                 # File and K8s configuration stores
│   ├── validation-pipeline.md           # Multi-layer validation engine
│   └── refresh-and-diagnostics.md       # Refresh, rate limiting, diagnostics
│
├── universal-assets/                    # Capabilities, asset contract, actions
│   ├── README.md
│   ├── capabilities-discovery.md        # V2 capabilities schema and builder
│   ├── asset-contract.md                # AssetResource, AssetMapper, overlays
│   └── action-framework.md              # ActionProvider, builtins, :action endpoints
│
├── plugins/                             # Concrete plugin implementations
│   ├── README.md
│   ├── model-and-mcp-plugins.md         # Model and MCP plugins
│   └── asset-type-plugins.md            # Knowledge, Prompts, Agents, Guardrails, Policies, Skills
│
├── operations/                          # Deployment, security, and enterprise ops
│   ├── README.md
│   ├── deployment.md                    # Docker, Kubernetes, health probes, HA
│   ├── security.md                      # RBAC, JWT, SecretRef, authz, audit
│   ├── enterprise-ops-runbook.md        # Day-0/1 ops, troubleshooting, backup
│   └── upgrade-guide.md                 # Phase 8 upgrade and migration guide
│
└── clients/                             # Client surfaces
    ├── README.md
    ├── bff-integration.md               # BFF proxy handlers
    ├── generic-ui.md                    # Capabilities-driven React components
    └── catalogctl-and-conformance.md    # Dynamic CLI and conformance suite
```

## Quick Navigation

| Need | Go To |
|------|-------|
| Understand the plugin architecture | [Plugin Framework Architecture](./plugin-framework/architecture.md) |
| Create a new asset-type plugin | [Creating Plugins](./plugin-framework/creating-plugins.md) |
| Configure catalog sources | [Configuration](./plugin-framework/configuration.md) |
| Manage sources at runtime | [Source Management](./source-management/README.md) |
| Understand capabilities discovery | [Capabilities Discovery](./universal-assets/capabilities-discovery.md) |
| Learn about the action system | [Action Framework](./universal-assets/action-framework.md) |
| Deploy the catalog stack | [Deployment](./operations/deployment.md) |
| Set up authentication | [Security](./operations/security.md) |
| Enable multi-tenancy | [Deployment: Multi-Tenant](./operations/deployment.md#multi-tenant-deployment-phase-8) |
| Configure authorization | [Security: Multi-Tenant Authorization](./operations/security.md#multi-tenant-authorization-phase-8) |
| HA deployment | [Deployment: HA](./operations/deployment.md#ha-deployment-phase-8) |
| Operate in production | [Enterprise Ops Runbook](./operations/enterprise-ops-runbook.md) |
| Upgrade to Phase 8 | [Upgrade Guide](./operations/upgrade-guide.md) |
| Build UI integrations | [Generic UI](./clients/generic-ui.md) |
| Run conformance tests | [catalogctl and Conformance](./clients/catalogctl-and-conformance.md) |

## API Surface

| Method | Endpoint Pattern | Description |
|--------|-----------------|-------------|
| GET | `/api/plugins` | List plugins with V2 capabilities |
| GET | `/api/plugins/{name}/capabilities` | Single plugin V2 capabilities |
| GET | `/{basePath}/{entities}` | List entities |
| GET | `/{basePath}/{entities}/{name}` | Get entity by name |
| GET | `/{basePath}/management/sources` | List data sources |
| POST | `/{basePath}/management/sources` | Apply source configuration |
| POST | `/{basePath}/management/validate-source` | Validate source config |
| POST | `/{basePath}/management/refresh` | Refresh all sources |
| POST | `/{basePath}/management/entities/{name}:action` | Execute entity action |
| POST | `/{basePath}/management/sources/{id}:action` | Execute source action |
| GET | `/{basePath}/management/revisions` | List config revisions |
| POST | `/{basePath}/management/rollback` | Restore previous config |
| GET | `/{basePath}/management/diagnostics` | Plugin diagnostics |
| GET | `/api/tenancy/v1alpha1/namespaces` | List namespaces available to user |
| GET | `/api/audit/v1alpha1/events` | List audit events (paginated, filtered) |
| GET | `/api/audit/v1alpha1/events/{id}` | Get audit event by ID |
| GET | `/api/jobs/v1alpha1/refresh` | List refresh jobs (paginated, filtered) |
| GET | `/api/jobs/v1alpha1/refresh/{id}` | Get refresh job by ID |
| POST | `/api/jobs/v1alpha1/refresh/{id}:cancel` | Cancel a queued refresh job |
| GET | `/healthz`, `/livez` | Liveness probe |
| GET | `/readyz` | Readiness probe |

## Plugin Inventory

| Plugin | Entity Kind | Source Types | Base Path |
|--------|------------|-------------|-----------|
| **model** | CatalogModel | yaml, hf | `/api/model_catalog/v1alpha1` |
| **mcp** | McpServer | yaml | `/api/mcp_catalog/v1alpha1` |
| **knowledge** | KnowledgeSource | yaml | `/api/knowledge_catalog/v1alpha1` |
| **prompts** | PromptTemplate | yaml | `/api/prompts_catalog/v1alpha1` |
| **agents** | Agent | yaml, git | `/api/agents_catalog/v1alpha1` |
| **guardrails** | Guardrail | yaml | `/api/guardrails_catalog/v1alpha1` |
| **policies** | Policy | yaml | `/api/policies_catalog/v1alpha1` |
| **skills** | Skill | yaml | `/api/skills_catalog/v1alpha1` |

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/plugin.go` | Core and optional plugin interfaces |
| `pkg/catalog/plugin/server.go` | Server lifecycle, routing, health |
| `pkg/catalog/plugin/capabilities_types.go` | V2 capabilities schema |
| `pkg/catalog/plugin/asset_types.go` | Universal AssetResource types |
| `pkg/catalog/plugin/action_types.go` | Action framework types |
| `pkg/catalog/plugin/configstore.go` | ConfigStore interface |
| `pkg/catalog/plugin/management_handlers.go` | Management API handlers |
| `pkg/catalog/plugin/validator.go` | Multi-layer validation engine |
| `cmd/catalog-server/main.go` | Server entry point |
| `cmd/catalogctl/main.go` | CLI entry point |
| `catalog/config/sources.yaml` | Default source configuration |
| `tests/conformance/` | Conformance test suite |
| `pkg/tenancy/` | Multi-tenant context, middleware, resolvers (Phase 8) |
| `pkg/authz/` | Authorization: SAR, identity, caching, mapper (Phase 8) |
| `pkg/audit/` | Audit logging: middleware, handlers, retention (Phase 8) |
| `pkg/jobs/` | Async refresh: job store, worker pool, handlers (Phase 8) |
| `pkg/cache/` | Discovery caching: LRU, middleware, invalidation (Phase 8) |
| `pkg/ha/` | HA: migration locking, leader election (Phase 8) |

---

[Back to Documentation Root](../README.md)
