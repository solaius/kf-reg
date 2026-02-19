# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Project Instructions for Claude Code

## Context
You are working in the Kubeflow Model Registry repository, focused on the Model Catalog and the in-flight plugin-based catalog architecture described in the upstream proposal PR.

Your job is to implement a "catalog of catalogs" experience by extending the upstream plugin architecture so the catalog-server can host multiple AI asset catalogs (models plus additional asset types) in one process, with consistent API patterns, a shared database, and a unified OpenAPI spec.

## How to use this spec pack
1. Read 00_project_overview.md through 12_risks_assumptions_dependencies.md in order
2. Produce a plan that answers the open questions and proposes an implementation sequence
3. Implement in small, verifiable slices
4. Keep compatibility guarantees intact for existing Model Catalog consumers

## Guardrails
- Do not introduce breaking changes to existing Model Catalog API paths, schemas, or behaviors
- Follow repository conventions in PROGRAMMING_GUIDELINES.md (contract-first OpenAPI, codegen, DB patterns, tests)
- Prefer deterministic generation for boilerplate, minimize handwritten duplication
- Write tests for any new behavior, and keep CI green
- If you must choose between flexibility and clarity, choose clarity in the public contract and flexibility behind it


## Repository programming guidelines
Before making implementation choices, read PROGRAMMING_GUIDELINES.md and obey it.

Non-negotiables from the repo guidelines:
- Contract-first: OpenAPI spec is source of truth for REST APIs; regenerate stubs and clients after changes
- Do not edit generated files; change the source inputs and re-run generators
- Keep layer boundaries: OpenAPI stubs -> core logic -> converters/mappers -> DB layer (GORM) where applicable
- Use repo-standard error handling and sentinel errors (ErrBadRequest, ErrNotFound, ErrConflict) and wrap with context
- Use goimports import grouping, repo naming conventions, and glog for logging
- Tests: table-driven unit tests where possible and integration tests for DB and API where needed

Practical command loop (adjust to the component you touch):
- make gen
- make lint
- make test
- make openapi/validate (or the catalog OpenAPI validation target if different)

## Working style
- Start by restating the goal in your own words, then list assumptions and open questions
- Propose a short milestone plan (3 to 6 milestones) with concrete outputs per milestone
- After each milestone, run relevant unit and integration tests and summarize results
- Keep changes easy to review: small PR-friendly commits, clean messages

## Definition of Done
- At least one non-model asset-type plugin is functional end-to-end (source config -> ingest -> list/get endpoints)
- Unified catalog-server can list loaded plugins and their health
- Unified OpenAPI spec merges plugin specs deterministically in CI
- Backward compatibility for the existing model catalog is preserved
- Docs are updated to explain how to add a new asset-type plugin and wire UI and CLI integration


## Phase Status
- **Phase 1 (Plugin Architecture)**: Complete
- **Phase 2/2.5 (Catalog Management UX)**: Complete
- **Phase 3 (Catalog Operationalization)**: Complete - real data pipeline, BFF wiring, MCP extended fields, UI operational hardening
- **Phase 4 (Catalog Management Productionization)**: Complete - persistence hardening, validation/rollback, refresh feedback, health probes, 3 rounds of review fixes
- **Phase 5 (Universal Asset Framework)**: Complete - capabilities-driven discovery, universal asset contract, action framework, generic UI/CLI, knowledge sources plugin, conformance suite (all 3 plugins pass)
- **Phase 6 (Core Asset Coverage)**: Complete - 5 new plugins (Prompts, Agents, Guardrails, Policies, Skills), Git provider, management route fix
- **Phase 7 (Lifecycle Governance)**: Complete - state machine, approvals, versioning, promotion, audit
- **Phase 8 (Scale, Multi-tenancy, Enterprise Ops)**: Complete - namespace tenancy, SAR authz, audit v2, async jobs, HA migration locking, leader election, caching
- **Phase 9 (Extensibility and Ecosystem)**: Complete - plugin.yaml schema, catalog-gen vNext (validate, bump-version, build-server), UI hints formalization, importable conformance harness, governance checks, supported plugin index
- **Phase 10 (Original Functionality Confirmation)**: Complete - upstream files byte-identical to merge-base, legacy MCP browsing code removed, plugin-only additions preserved, all 8 plugins verified, conformance suite passes

## Phase 5 Goals
Make the system a true platform where new asset-type plugins appear in UI and CLI with zero code changes. Proven by adding a Knowledge Sources plugin that works end-to-end without touching frontend or CLI code.

### Phase 5 Exit Criteria
- Knowledge Sources plugin appears in UI nav, renders list/detail, supports actions -- all without changes to `clients/ui/frontend/` or `cmd/catalogctl/`
- All plugins pass conformance suite
- No plugin-specific branching in generic UI components or CLI

### Phase 5 Milestones
| Milestone | Description | Key Files |
|-----------|-------------|-----------|
| M5.1 | Capabilities Schema + Endpoints | `pkg/catalog/plugin/capabilities_types.go`, `capabilities_builder.go` |
| M5.2 | Universal Asset Contract | `pkg/catalog/plugin/asset_types.go`, `asset_mapper.go` |
| M5.3 | Action Model + Framework | `pkg/catalog/plugin/action_types.go`, `action_handler.go`, `overlay_store.go` |
| M5.4 | Model + MCP Plugin Updates | `catalog/plugins/mcp/actions.go`, `asset_mapper.go` |
| M5.5 | Generic UI Components + BFF | `clients/ui/frontend/src/app/pages/genericCatalog/` |
| M5.6 | CLI v2 (catalogctl) | `cmd/catalogctl/` |
| M5.7 | Knowledge Sources Plugin | `catalog/plugins/knowledge/` |
| M5.8 | Conformance Suite | `tests/conformance/` |

### Phase 5 Key Interfaces
```go
// pkg/catalog/plugin/plugin.go - New optional interfaces
type CapabilitiesV2Provider interface { GetCapabilitiesV2() PluginCapabilitiesV2 }
type AssetMapperProvider interface { GetAssetMapper() AssetMapper }
type ActionProvider interface {
    HandleAction(ctx context.Context, scope ActionScope, targetID string, req ActionRequest) (*ActionResult, error)
    ListActions(scope ActionScope) []ActionDefinition
}
```

### Phase 5 Endpoint Patterns
```
GET  /api/plugins/{plugin}/capabilities     # V2 capabilities discovery
POST /api/{plugin}_catalog/{ver}/sources/{id}:action    # Source actions (refresh)
POST /api/{plugin}_catalog/{ver}/entities/{name}:action     # Asset actions (tag, deprecate)
```

## Phase 4 Key Files
- `pkg/catalog/plugin/configstore.go` - ConfigStore interface (Load, Save, Watch, ListRevisions, Rollback)
- `pkg/catalog/plugin/file_config_store.go` - File-backed store with atomic writes and revision history
- `pkg/catalog/plugin/k8s_config_store.go` - Kubernetes ConfigMap-backed ConfigStore with RetryOnConflict
- `pkg/catalog/plugin/validator.go` - Multi-layer YAML validation engine
- `pkg/catalog/plugin/management_handlers.go` - Management API endpoints (sources, validate, revisions, rollback, refresh)
- `pkg/catalog/plugin/server.go` - Plugin server with /livez, /readyz health endpoints
- `catalog/plugins/mcp/management.go` - MCP plugin management (ListSources, ApplySource, Refresh)
- `clients/ui/bff/internal/api/catalog_management_handler.go` - BFF management handlers
- `clients/ui/frontend/src/app/pages/catalogManagement/` - UI management pages

## Running Services (Dev)
- **PostgreSQL**: localhost:5432 (docker container catalog-postgres)
- **Catalog Server**: localhost:8080 (docker container catalog-server)
- **BFF**: localhost:4000 (go run ./cmd/ with CATALOG_SERVER_BASE_URL)
- **Frontend**: localhost:9000 (npm run start:dev)

## Repository conventions to follow (summary)

### Contract-first OpenAPI
- OpenAPI is the source of truth
- Source specs live under api/openapi/src
- Merge scripts generate the final spec artifacts

### Filtering and pagination patterns
- Use filterQuery consistently on list endpoints
- Use token-based pagination with pageSize and nextPageToken
- Support orderBy and sortOrder

### Testing expectations
- Add unit tests for new framework utilities
- Add integration tests for ingest and API behavior
- Keep generated artifacts in sync and validated in CI


## Build Commands

### Go Backend
```bash
make build              # Full build with code generation, vetting, linting, and compilation
make build/compile      # Compile only (skip code generation)
make test               # Run tests (excludes controller tests)
make test-cover         # Run tests with coverage report
make lint               # Run golangci-lint
make gen/gorm           # Regenerate GORM structs after schema changes (requires Docker)
make run/proxy          # Start the OpenAPI proxy server from source
```

### Docker Compose (Full Stack)
```bash
make compose/up              # Start with MySQL (pre-built images)
make compose/up/postgres     # Start with PostgreSQL
make compose/local/up        # Start with MySQL (builds from source)
make compose/down            # Stop services
```

### UI Frontend (clients/ui/frontend)
```bash
npm install             # Install dependencies
npm run start:dev       # Development server with hot reload
npm run test            # Run linting, type-check, unit tests, and Cypress
npm run build           # Production build
```

### UI Development (clients/ui)
```bash
make dev-start          # Start BFF and frontend together (standalone mode)
make dev-bff            # Start BFF server only (mocked)
make dev-frontend       # Start frontend dev server only
```

### Python Client (clients/python)
```bash
make install            # Generate OpenAPI client and install with Poetry
make test               # Run pytest
make lint               # Run ruff and mypy
```

## Architecture Overview

**Contract-First REST API**: The server implements the OpenAPI specification at `api/openapi/model-registry.yaml`. All API changes start with the OpenAPI contract.

**Layered Go Architecture**:
- `cmd/` - CLI entry points (Cobra-based), main command is `proxy`
- `pkg/api/api.go` - Core `ModelRegistryApi` interface defining domain operations
- `internal/server/openapi/` - Auto-generated OpenAPI server (router/controller pattern)
- `internal/core/` - `ModelRegistryService` implementing business logic
- `internal/converter/` - Bidirectional converters between OpenAPI and internal models
- `internal/db/models/` - Repository interfaces and GORM models
- `internal/datastore/` - Pluggable datastore connectors (MySQL, PostgreSQL via GORM)

**Code Generation Flow**:
1. OpenAPI spec → `make gen/openapi-server` → server code in `internal/server/openapi/`
2. Database schema → `make gen/gorm` → GORM structs in `internal/db/`
3. Type converters → `make gen/converter` → converters in `internal/converter/`

## Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| Core API Server | `cmd/proxy.go`, `internal/` | REST API serving model metadata |
| Model Catalog | `catalog/` | Federated model discovery (Hugging Face, YAML sources) |
| UI BFF | `clients/ui/bff/` | Backend for Frontend for React UI |
| UI Frontend | `clients/ui/frontend/` | React/TypeScript/PatternFly application |
| Python Client | `clients/python/` | Auto-generated OpenAPI Python client |
| K8s Controller | `cmd/controller/` | Kubernetes controller for CRDs |
| CSI Driver | `cmd/csi/` | Container Storage Interface for model artifacts |

## Database

Supports MySQL 8.3+ and PostgreSQL. When modifying database schema:
1. Update migrations in `internal/db/`
2. Run `make gen/gorm` to regenerate GORM structs (requires Docker)

## Testing

- **Go tests**: `make test` or `make test-cover`
- **Controller tests**: `make controller/test`
- **Frontend tests**: `cd clients/ui/frontend && npm run test`
- **Python tests**: `cd clients/python && make test`
- **Integration tests**: Use Testcontainers (requires Docker)

## DCO Requirement

All commits require Developer Certificate of Origin sign-off: `git commit -s`

## Phase Status

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 (Plugin Architecture) | Complete | Catalog plugin framework, plugin registry, server lifecycle |
| Phase 2/2.5 (Catalog Management UX) | Complete | MCP catalog realization, management plane, BFF integration |
| Phase 3 (Catalog Operationalization) | Complete | Real data pipeline, BFF wiring, MCP extended fields, UI hardening |
| Phase 4 (Management Productionization) | Complete | Persistence hardening, validation/rollback, refresh feedback, health probes (3 review fix rounds) |
| Phase 5 (Universal Asset Framework) | Complete | Capabilities-driven discovery, universal asset contract, action framework, generic UI/CLI, knowledge sources plugin, conformance suite |
| Phase 6 (Core Asset Coverage) | Complete | 5 new plugins (Prompts, Agents, Guardrails, Policies, Skills), Git provider, management route fix |
| Phase 7 (Lifecycle Governance) | Complete | State machine, approvals, versioning, promotion bindings, audit |
| Phase 8 (Scale, Multi-tenancy, Enterprise Ops) | Complete | Namespace tenancy, SAR authz, audit v2, async jobs, HA migration locking, leader election, caching |
| Phase 9 (Extensibility and Ecosystem) | Complete | plugin.yaml, catalog-gen vNext, UI hints, conformance harness, server builder, governance, plugin index |
| Phase 10 (Original Functionality Confirmation) | Complete | Upstream files reverted to merge-base, legacy MCP removed, plugin additions preserved, all 8 plugins verified |

## Phase 9 Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/plugin_metadata.go` | PluginMetadataSpec types, LoadPluginMetadata, ValidatePluginMetadata |
| `pkg/catalog/plugin/ui_hints_types.go` | FieldDisplayType enum, ListViewHints, DetailViewHints, SearchHints, ActionDisplayHints |
| `pkg/catalog/plugin/ui_hints_validator.go` | ValidateUIHints validation |
| `pkg/catalog/plugin/governance_checks.go` | RunGovernanceChecks for plugin directories |
| `pkg/catalog/conformance/harness.go` | RunConformance importable conformance suite |
| `pkg/catalog/conformance/config.go` | HarnessConfig for conformance test configuration |
| `cmd/catalog-gen/validate.go` | catalog-gen validate command with --governance flag |
| `cmd/catalog-gen/bump_version.go` | catalog-gen bump-version command |
| `cmd/catalog-gen/server_builder.go` | catalog-gen build-server command |
| `deploy/plugin-index/` | Supported plugin index with 8 built-in plugin entries |

## Available Skills

| Skill | Purpose |
|-------|---------|
| `implement_catalog_plugin` | Add or extend a catalog plugin (includes Phase 5 interfaces) |
| `scaffold_plugin` | Create a new plugin from scratch following Phase 5 patterns |
| `enable_generic_ui_and_cli` | Wire capabilities-driven UI and CLI |
| `validate_and_test_backend` | Backend validation and testing |
| `validate_openapi_and_contracts` | OpenAPI spec validation |
| `docker_stack_verify` | Docker Compose stack verification (includes Phase 5 capabilities and actions) |
| `run_playwright_tests` | Playwright MCP browser-based UI testing |
| `run_conformance_tests` | Plugin conformance suite verification |
| `create_implementation_report` | Standardized milestone implementation reports |
| `update_prod_doc` | Update product documentation in `prod-doc/catalog_of_catalogs/` after milestones |
| `plan_and_align` | Planning and alignment |
| `rollback_config` | Config rollback operations |

## Phase 4 Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/configstore.go` | ConfigStore interface (Load, Save, Watch, ListRevisions, Rollback) |
| `pkg/catalog/plugin/file_config_store.go` | File-backed store with atomic writes, SHA-256 versioning, revision history |
| `pkg/catalog/plugin/k8s_config_store.go` | Kubernetes ConfigMap-backed ConfigStore with RetryOnConflict |
| `pkg/catalog/plugin/validator.go` | Multi-layer YAML validation engine (parse, strict, semantic, provider) |
| `pkg/catalog/plugin/management_handlers.go` | Management API (sources, validate, revisions, rollback, refresh, apply) |
| `pkg/catalog/plugin/management_types.go` | Types: SourceConfigInput, ApplyResult, RefreshResult, ValidationResult |
| `pkg/catalog/plugin/server.go` | Plugin server with /livez, /readyz health endpoints, DB+plugin checks |
| `cmd/catalog-server/main.go` | Entry point with CATALOG_CONFIG_STORE_MODE (file, k8s, none) |
| `cmd/healthcheck/main.go` | Minimal HTTP healthcheck binary for distroless images |
| `catalog/plugins/mcp/management.go` | MCP plugin management (ListSources, ApplySource, Refresh) |
| `clients/ui/bff/internal/api/catalog_management_handler.go` | BFF management handlers |
| `deploy/catalog-server/rbac.yaml` | K8s RBAC for ConfigMap access |
| `deploy/catalog-server/deployment.yaml` | K8s deployment with startup/liveness/readiness probes |

## Running Services (Dev)

| Service | Address | Container |
|---------|---------|-----------|
| PostgreSQL | localhost:5432 | catalog-postgres |
| Catalog Server | localhost:8080 | catalog-server |
| BFF | localhost:4000 | `go run ./cmd/` with CATALOG_SERVER_BASE_URL |
| Frontend | localhost:9000 | `npm run start:dev` |

### Catalog Server Health
```bash
curl -s http://localhost:8080/livez    # Liveness (always 200)
curl -s http://localhost:8080/readyz   # Readiness (DB + plugins + initial load)
```

### Docker Compose (Catalog Stack)
```bash
docker compose -f docker-compose.catalog.yaml up --build -d    # Start
docker compose -f docker-compose.catalog.yaml down -v           # Stop
```

### Phase 5 Verification
```bash
# Capabilities discovery
curl -s http://localhost:8080/api/plugins/mcp/capabilities | python3 -m json.tool

# Action execution
curl -s -X POST http://localhost:8080/api/mcp_catalog/v1alpha1/mcpservers/filesystem:action \
  -H 'Content-Type: application/json' \
  -d '{"action":"tag","dryRun":true,"params":{"tags":["test"]}}' | python3 -m json.tool

# Conformance suite
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1
```

## Product Documentation (`prod-doc/catalog_of_catalogs/`)

The `prod-doc/catalog_of_catalogs/` folder contains the **canonical product documentation** for the catalog-of-catalogs platform. It covers the plugin framework, source management, universal asset framework, all 8 plugins, deployment/security operations, and client integrations (BFF, generic UI, CLI). It is organized by topic (not by phase) and contains 23 files across 7 directories. See `prod-doc/catalog_of_catalogs/README.md` for the full structure and navigation.

This documentation must be kept current as the authoritative reference for how the catalog platform works. It is distinct from the phase-level implementation reports (which capture what was done in each milestone) — prod-doc describes the system as it is today.

## Milestone Completion Workflow

After completing each milestone, two documentation deliverables are required:

### 1. Implementation Report (required)

Create an implementation report in the current phase's `implementation-reports/` folder using the `create_implementation_report` skill. The report captures what was done, why, and how.

**Location**: `catalog_of_catalogs/<current-phase-folder>/implementation-reports/`
**Naming**: `M<number>_<short-slug>.md` (e.g., `M6.5_skills-plugin.md`)

Example for the current phase structure:
```
catalog_of_catalogs/Phase-6-core-asset-coverage/implementation-reports/
├── M6.1_git-provider-and-stack-preparation.md
├── M6.2_prompt-templates-plugin.md
├── M6.3_agents-catalog-plugin.md
└── ...
```

### 2. Product Documentation Update (required)

Update the relevant files in `prod-doc/catalog_of_catalogs/` to reflect the new functionality using the `update_prod_doc` skill. This ensures the product documentation stays current.

**What to update depends on the change** — see the mapping table in the `update_prod_doc` skill for guidance. At minimum:
- New plugins: update `plugins/asset-type-plugins.md` and root `README.md` plugin inventory
- New interfaces: update `plugin-framework/architecture.md`
- New endpoints: update root `README.md` API surface table
- New config options: update `plugin-framework/configuration.md`

### Workflow summary
```
Complete milestone
    │
    ├──▶ Run `create_implementation_report` skill
    │    → writes to catalog_of_catalogs/<phase>/implementation-reports/
    │
    └──▶ Run `update_prod_doc` skill
         → updates relevant files in prod-doc/catalog_of_catalogs/
```
