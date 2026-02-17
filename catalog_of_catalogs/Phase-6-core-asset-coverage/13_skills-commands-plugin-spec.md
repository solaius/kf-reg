# Skills (commands) plugin specification

## Purpose
Represent reusable "skills" as cataloged tool definitions that agents can reference.

A Skill is a tool contract plus an execution hint. It is not the executor itself. The executor may be:
- an MCP server tool
- an HTTP API described by OpenAPI
- a platform-native action (Kubernetes or OpenShift)
- a safe command wrapper (highly constrained)

This plugin is critical for making agents portable across environments.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) MCP tools schema
- How tools represent name, description, inputSchema, and result schema patterns

2) OpenAPI 3.1
- How to represent API-backed tools with JSON Schema-based request and response schemas

3) Platform-native actions
- How to represent Kubernetes resource operations safely and declaratively

Research sources (starting points):
- MCP tools: https://modelcontextprotocol.io/specification/2025-06-18/server/tools
- OpenAPI 3.1: https://spec.openapis.org/oas/v3.1.0
- Kubernetes CRD OpenAPI v3 schema: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/

## Schema draft
### Entity: Skill
Required fields
- name
- version
- description
- skillType (enum)
  - mcp_tool
  - openapi_operation
  - k8s_action
  - shell_command
- inputSchema (JSON Schema 2020-12)
- outputSchema (optional JSON Schema 2020-12)

Strongly recommended fields
- execution (object)
  - executorType (enum)
    - mcp_server
    - http
    - kubernetes
    - local_command
  - mcp (optional)
    - mcpServerRef
    - toolName
  - http (optional)
    - openapiRef (artifact ref)
    - operationId
    - baseUrlOverride (optional)
  - kubernetes (optional)
    - apiGroup
    - version
    - kind
    - verbsAllowed (array: get, list, create, update, patch, delete)
    - namespaceScope (enum: namespaced, cluster)
  - command (optional)
    - allowedBinaries (array)
    - argsTemplate (string)
    - workingDirPolicy
- safety (object)
  - requiresApproval (bool)
  - riskLevel (enum: low, medium, high)
  - networkAccessRequired (bool)
  - secretRefs (list of secret names only)
  - allowedNamespaces (optional)
- rateLimit (optional)
- timeoutSeconds (required)
- retryPolicy (optional)
- compatibility (object)
  - supportedEnvironments (local, cluster, remote)
- provenance, license, owner fields

Artifacts
- openapiSpec (optional)
- docs
- examples
- tests

Filtering fields (minimum)
- name
- version
- skillType
- riskLevel
- lifecycleState

## Providers
Baseline
- YAML provider loads skills

Recommended additions in Phase 6
- Git provider for skills-as-code repositories (recommended)
- HTTP provider optional for remote skills hubs
- OCI provider optional for distributing OpenAPI specs or skill bundles

Validation requirements
- Validate JSON Schema correctness
- Validate executor information is complete for skillType
- Validate shell_command skills:
  - only allowed binaries
  - no unbounded argument injection (strict template rules)
- Validate timeoutSeconds and riskLevel consistency

## Actions and lifecycle
Supported actions (opt-in)
- validate: schema validation + executor validation + safety checks
- apply: persist metadata and artifacts pointers
- refresh: re-sync
- promote, deprecate, tag, annotate, link

Safety requirements
- shell_command skills must default to requiresApproval true
- Secrets must never be embedded, only referenced

## API surface
Must conform to the common plugin API patterns:
- /api/skill_catalog/v1alpha1/skills
- /api/skill_catalog/v1alpha1/skills/{id}
- /api/skill_catalog/v1alpha1/sources
- /api/skill_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works with strict validation
- At least one skill is referenced by an Agent successfully
- UI renders skill details safely and highlights riskLevel
- CLI can validate and apply skills with consistent output
- Conformance suite passes

## Verification and test plan
Unit
- Validation of schema and executor completeness
- Safety rules for shell_command skills

Integration
- Provider sync persists skills and artifact pointers

E2E
- Load skills from YAML or Git
- Validate and apply via CLI
- Link skills to an Agent and verify UI dependencies view shows them
