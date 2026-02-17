# Agents catalog plugin specification

## Purpose
Represent AI agents as first-class, linkable assets that can be assembled from other assets (prompt templates, skills/tools, knowledge sources, guardrails, policies, evaluators) and managed consistently by Ops and AI Engineers.

This plugin is the centerpiece of the catalog-of-catalogs because agents tie everything together.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) Agent definition patterns in modern agent frameworks
- Minimum fields: instructions, model config, tool definitions, memory and context sources, guardrails/policies, input/output schemas, examples
- How agent capabilities are declared (supported tools, supported modalities, invocation style)

2) Interoperability protocols
- Agent-to-agent and tool schemas where applicable
- How to represent "handoffs" or delegation

3) Versioning and lifecycle practices for agents
- How teams promote and deprecate agent versions
- How dependencies are pinned (by version or digest)

Research sources (starting points):
- OpenAI Agents SDK docs (agent concepts, tools, handoffs): https://platform.openai.com/docs/agents
- Agent2Agent protocol discussions (A2A): https://a2a.readthedocs.io/
- MCP tools schema patterns for tool input and output schemas: https://modelcontextprotocol.io/specification/2025-06-18/server/tools

## Schema draft
### Entity: Agent
Required fields
- name
- version
- agentType (enum)
  - conversational
  - task_oriented
  - router
  - planner
  - executor
  - evaluator
- description
- inputSchema (optional JSON Schema 2020-12)
- outputSchema (optional JSON Schema 2020-12)
- instructions
  - Either inline string, or reference to a PromptTemplate asset

Strongly recommended fields
- modelConfig (object)
  - preferredModelRef (reference to Model asset, optional)
  - fallbackModelRefs (optional)
  - maxTokens
  - temperature
  - topP
  - presencePenalty
  - frequencyPenalty
- tools (array)
  - toolType (enum)
    - skillRef (reference to Skills asset)
    - mcpToolRef (reference to MCP server + tool name)
    - openapiToolRef (reference to OpenAPI spec artifact)
  - required (bool)
  - timeoutSeconds
  - retryPolicy
- knowledge (array)
  - knowledgeSourceRefs (references to Knowledge Sources assets)
  - retrievalHints (optional)
    - topK
    - filters
    - chunkingProfile
- guardrails (array)
  - guardrailRefs (references to Guardrails assets)
  - enforcementMode (enum: advisory, required)
- policies (array)
  - policyRefs (references to Policies assets)
  - enforcementMode (enum: advisory, required)
- evaluation (object)
  - evaluatorRefs (references to Evaluators assets)
  - benchmarkRefs (references to Benchmarking assets)
  - qualityGates (object)
    - mustPass (list of checks)
    - thresholds
- dependencies (object)
  - promptTemplates (list of refs)
  - skills (list of refs)
  - mcpServers (list of refs)
  - knowledgeSources (list of refs)
  - guardrails (list of refs)
  - policies (list of refs)
- runtimeHints (object)
  - executionEnvironment (enum: local, cluster, remote)
  - requiredSecrets (list of secret names, no values)
  - requiredPermissions (optional)
- examples (array)
  - { input, expectedOutput?, notes? }
- documentation (object)
  - usageNotes
  - troubleshooting
  - owner
  - supportContact
- license (SPDX expression)
- provenance (object)
  - upstreamUrl
  - upstreamRevision
  - author
  - createdAt

Artifacts (recommended)
- agentSpecFile (canonical JSON or YAML definition)
- testCases (evaluation harness inputs)
- diagrams or docs (optional)

Linking rules
- Agents can link to any other asset type
- References should support pinning to version or digest
- Agent must remain resolvable even if dependencies are deprecated (warn, do not break)

Filtering fields (minimum)
- name
- version
- agentType
- lifecycleState
- owner
- labels
- requiredTools (derived from tools)
- requiredKnowledgeSources (derived from knowledge)

## Providers
Baseline
- YAML provider loads Agent definitions from local YAML

Recommended additions in Phase 6
- Git provider for agent catalogs managed as code
- HTTP provider for remote agent hubs (optional)

Validation requirements
- Validate inputSchema and outputSchema if present
- Validate references resolve:
  - prompt templates exist if referenced
  - skills exist if referenced
  - MCP servers exist if referenced
  - knowledge sources exist if referenced
- Validate no secret values are present in the asset definition
- Validate tool timeouts and retry policies are within safe limits

## Actions and lifecycle
Supported actions (opt-in)
- validate: schema validation + dependency resolution + safety checks
- apply: persist into DB and make discoverable
- refresh: re-sync from sources
- promote, deprecate: lifecycle transitions with audit metadata
- link: add links to related assets (prompt templates, skills, guardrails, policies)

Optional future actions (not required in Phase 6)
- deploy or run: out of scope unless already supported by platform components

Safety requirements
- validate must be required before apply by default
- apply must record the dependency graph snapshot (versions or digests)
- promote must require a validation pass and optional evaluation gate checks when configured

## API surface
Must conform to the common plugin API patterns:
- /api/agent_catalog/v1alpha1/agents
  - list with filterQuery and pagination
- /api/agent_catalog/v1alpha1/agents/{id}
  - get
- /api/agent_catalog/v1alpha1/sources
  - list sources, diagnostics
- /api/agent_catalog/v1alpha1/actions/{action}
  - execute validate/apply/refresh and universal actions

Optional helper endpoints (if already supported by the universal framework)
- /api/agent_catalog/v1alpha1/agents/{id}/dependencies
  - return resolved dependency graph (computed server-side)

## Definition of done
- Plugin generated with catalog-gen and wired into catalog-server
- YAML provider works with strong validation
- Git provider works for agent catalogs and captures commit provenance
- UI and CLI show Agents with:
  - list view, filters, detail view
  - dependencies panel rendering resolved links
  - action bar for validate/apply/promote/deprecate/refresh
- Agent linking works with at least:
  - prompt templates
  - skills
  - knowledge sources
  - MCP servers
- Conformance suite passes for the plugin

## Verification and test plan
Unit
- Schema validation and reference parsing
- Safety checks (no secret values, safe timeouts)
- Dependency resolver logic tests (happy and negative paths)

Integration
- Loader persists agent entities and link edges correctly
- Actions endpoint returns structured diagnostics with clear errors

E2E
- Load a Git-based agent catalog
- Use CLI to validate and apply agents
- Use UI to browse agents, view dependencies, filter by agentType, and execute actions
- Verify that broken references are reported clearly and block promotion

Regression
- Ensure no changes to Model Catalog and MCP Catalog routes and behaviors
