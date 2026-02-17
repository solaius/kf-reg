# Universal Asset Contract

## Purpose
Define a common, cross-plugin representation that UI and CLI can rely on for:
- listing assets
- rendering details
- displaying lifecycle/status
- applying consistent tagging/annotation
- linking assets together

This contract must be:
- additive (no breaking changes)
- implementable for existing plugins (model, mcp) via mapping or minor schema extension
- expressive enough for future asset types (datasets, prompts, agents, evaluations, guardrails, knowledge sources)

## Terminology
- Plugin: a catalog type (model, mcp, dataset, etc.)
- Entity: the primary asset type the plugin manages (e.g., Model, McpServer)
- Artifact: secondary resources attached to an entity (e.g., model files, metrics, configs)
- Asset: a generic term for an entity instance in any plugin

## Universal Asset Shape (conceptual)
All assets are represented to UI/CLI as an AssetResource with a stable envelope.

### AssetResource (logical)
- apiVersion: string (plugin group/version, e.g., catalog.kubeflow.org/v1alpha1)
- kind: string (entity kind, e.g., Model, McpServer, KnowledgeSource)
- metadata:
  - uid: string
  - name: string
  - displayName: string (optional)
  - description: string (optional)
  - labels: map[string]string
  - annotations: map[string]string
  - tags: [string] (user-facing tags, distinct from labels)
  - createdAt, updatedAt: RFC3339 timestamps
  - owner: { id, displayName, email } (optional)
  - sourceRef: { plugin, sourceId, sourceType } (where it came from)
- spec:
  - plugin-defined content (opaque to generic UI except for “hinted” fields)
- status:
  - lifecycle:
    - phase: enum [draft, active, deprecated, archived]
    - reason: string
    - message: string
    - lastTransitionTime: timestamp
  - conditions: array of { type, status, reason, message, lastTransitionTime }
  - health:
    - state: enum [unknown, healthy, degraded, unhealthy]
    - lastCheckedAt: timestamp
    - details: map[string]any (optional)
  - links:
    - inbound: list of LinkRef
    - outbound: list of LinkRef

### LinkRef (logical)
- type: string (e.g., usesPrompt, usesGuardrail, usesMcpServer)
- target: { plugin, kind, uid, name }
- attributes: map[string]string (optional)

## Compatibility with existing BaseResource
If the server already exposes BaseResource/BaseResourceList, the Universal Asset Contract is satisfied by:
- keeping BaseResource stable
- adding fields additively via:
  - metadata.labels/annotations/tags
  - status block
  - sourceRef

Existing APIs can remain unchanged; the universal contract can be provided via:
- a new generic endpoint (see Capabilities spec), or
- plugin-specific responses including the same fields additively

## Required vs Optional fields
Required (must be present for all assets exposed to the generic UI/CLI):
- metadata.uid, metadata.name, metadata.labels, metadata.annotations
- metadata.createdAt, metadata.updatedAt
- metadata.sourceRef.plugin, metadata.sourceRef.sourceId (if applicable)
- status.lifecycle.phase (may be defaulted to active)
- status.health.state (may be unknown)

Optional:
- displayName, description, owner, tags, links, conditions

## Mapping rules
Plugins can map their existing entity models to AssetResource by:
- mapping primary identifiers into metadata.uid/name
- mapping existing custom properties into spec or metadata.annotations
- providing UI hints (see Plugin Capabilities) to expose meaningful spec fields in generic views

## Acceptance Criteria
- AC1: Model and MCP entity responses include (or can be projected to) the required universal fields with no data loss for their current UI use cases
- AC2: Generic UI list view can render any asset using only the universal fields and capability hints
- AC3: Generic CLI can print table view using universal fields (name, kind, lifecycle, health, source)
- AC4: Lifecycle and health fields are additive and do not break existing clients

## Definition of Done
- Universal asset schema is represented in OpenAPI (common.yaml) and is referenced (allOf) by each plugin’s entity schemas or available via a generic projection endpoint
- Migration notes exist for model/mcp to ensure consistent behavior
- Unit tests validate the universal fields are present in responses (contract tests)

## Verification plan
- Unit: schema compilation and OpenAPI validation
- Integration: spin up catalog-server with model+mcp and validate list responses include required fields
- UI: snapshot tests for generic list/detail rendering for both model and mcp
- CLI: golden tests for table/json/yaml output for model and mcp
