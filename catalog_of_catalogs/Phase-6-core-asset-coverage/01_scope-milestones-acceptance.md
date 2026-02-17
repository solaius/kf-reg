# Scope, milestones, and acceptance matrix

## In scope
- New asset plugins:
  - Guardrails
  - Prompt Templates
  - Agents catalog
  - Policies
  - Datasets
  - Evaluators
  - Benchmarking
  - Notebooks
  - Skills (commands)
- Provider ecosystem:
  - YAML provider remains baseline for all plugins
  - HTTP provider for remote catalogs
  - Git provider for catalog-as-code workflows
  - OCI registry provider for assets-as-artifacts workflows where appropriate
- Schema audits and alignment updates:
  - Model plugin schema alignment with universal contract and capabilities
  - MCP plugin schema alignment with universal contract and capabilities
  - Knowledge Sources plugin schema alignment with universal contract and capabilities
- Conformance and verification:
  - Extend conformance suite to include Phase 6 plugin requirements and provider contracts
  - End-to-end validation in both UI and CLI for a representative set of plugins and providers

## Primary user stories
### AI Engineer
- Discover assets across multiple sources (local and remote)
- Filter and compare assets quickly (fields, tags, lifecycle status, capabilities)
- Take safe management actions with clear feedback (validate, apply, refresh)
- Link assets together (agent uses prompt templates + guardrails + policies + knowledge sources)

### Ops for AI
- Control which sources are enabled and trusted
- Validate and apply catalog changes through safe workflows (dry-run, rollback, audit)
- Operate reliably (health checks, readiness, clear diagnostics)
- Manage lifecycle states (promote, deprecate) consistently across plugins

## Milestones (M6.x)
M6.1 Provider ecosystem foundations
- HTTP provider MVP that can be reused by multiple plugins
- Git provider MVP (read-only) for catalog-as-code repositories
- OCI provider MVP (read-only) for at least one plugin

M6.2 Prompt Templates plugin (real wiring)
- YAML provider (baseline)
- HTTP provider (optional but recommended early)
- UI and CLI fully manage prompt templates using generic components

M6.3 Agents catalog plugin (must ship)
- YAML provider baseline
- Git provider integration recommended for realistic agent catalog workflows
- UI and CLI fully manage agents using generic components
- Links to other assets supported (prompts, guardrails, policies, knowledge sources, skills)

M6.4 Guardrails + Policies plugins
- Guardrails: YAML and optionally OCI provider
- Policies: YAML and Git provider (OPA bundles as artifacts)

M6.5 Datasets + Evaluators + Benchmarks plugins (minimum 2 ship fully)
- Choose at least two to reach the "4 additional catalogs from real sources" exit criteria if earlier milestones land fewer
- Prioritize Datasets + Evaluators for immediate usefulness

M6.6 Notebooks + Skills plugins (if time)
- Notebooks: YAML and Git providers
- Skills: YAML baseline, align with MCP tool schema patterns

M6.7 Schema audits and alignment updates for existing plugins
- Update Model, MCP, Knowledge Sources to meet Phase 5 contract and Phase 6 quality bar

## Exit acceptance matrix
To declare Phase 6 complete:
- At least 4 new plugins are fully functional end-to-end:
  - List, get, filter
  - Source management
  - Action execution with feedback loop
  - UI and CLI support with no plugin-specific frontend/cli code
- At least 2 provider types beyond YAML are exercised end-to-end:
  - HTTP and Git are recommended as the minimum
- Conformance suite passes for all shipped plugins and providers
- Dev docs exist for:
  - How to add a new plugin with catalog-gen
  - How to add a new provider type
  - How to add new schema fields safely and regenerate
