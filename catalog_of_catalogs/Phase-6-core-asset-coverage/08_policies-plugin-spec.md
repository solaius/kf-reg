# Policies plugin specification

## Purpose
Represent policy-as-code assets that govern AI usage, compliance, access, and operational constraints.

Policies should be usable by agents and ops workflows, and should align with policy engines such as Open Policy Agent (Rego).

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) Open Policy Agent (OPA)
- Rego language basics and common metadata needs
- Bundle packaging and distribution
- How policies are versioned and promoted

2) AI governance and compliance needs (vendor-neutral)
- Typical policy categories: access control, data residency, PII handling, model allowlists, tool allowlists
- Audit requirements

Research sources (starting points):
- OPA docs: https://www.openpolicyagent.org/docs/latest/
- OPA bundle management: https://www.openpolicyagent.org/docs/latest/management-bundles/

## Schema draft
### Entity: Policy
Required fields
- name
- version
- policyType (enum)
  - access_control
  - data_governance
  - safety
  - tool_allowlist
  - model_allowlist
  - compliance
- language (enum)
  - rego
  - yaml_rules
  - json_rules
- description

Strongly recommended fields
- bundleRef (artifact reference)
  - For rego: an OPA bundle tar.gz or an OCI artifact containing policy files and data
- entrypoint (string)
  - Policy package and rule name to evaluate
- inputSchema (optional JSON Schema 2020-12)
  - Describes expected input document
- outputSchema (optional JSON Schema 2020-12)
  - Describes decision result shape
- defaultDecision (optional)
- enforcementScope (enum)
  - agent
  - organization
  - namespace
  - project
- enforcementMode (enum: advisory, required)
- testCasesRef (artifact reference)
- license (SPDX expression)
- provenance (object)
  - upstreamUrl
  - upstreamRevision
  - author
  - createdAt

Artifacts
- policyBundle (required for rego)
- policyDocs (optional)
- testCases (recommended)

Filtering fields (minimum)
- name
- version
- policyType
- language
- enforcementScope
- enforcementMode
- lifecycleState

## Providers
Baseline
- YAML provider loads Policy metadata and references policy bundles stored locally

Recommended additions in Phase 6
- Git provider for policy-as-code repositories (highly recommended)
- OCI provider for distributing bundles as artifacts (recommended)
- HTTP provider optional for pulling from policy hubs

Validation requirements
- Validate bundle structure:
  - required files exist
  - tar integrity if applicable
- Validate Rego syntax when language is rego (at least basic linting if possible)
- Validate test cases exist for required enforcementMode, or document exception

## Actions and lifecycle
Supported actions (opt-in)
- validate: bundle integrity + syntax checks + optional test execution
- apply: persist metadata and make bundles available
- enable, disable: if platform supports enabling policies by scope
- refresh: re-sync from sources
- promote, deprecate: lifecycle transitions

Safety requirements
- Policies must not embed secrets
- Policy bundles must record provenance and digest

## API surface
Must conform to the common plugin API patterns:
- /api/policy_catalog/v1alpha1/policies
- /api/policy_catalog/v1alpha1/policies/{id}
- /api/policy_catalog/v1alpha1/sources
- /api/policy_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works
- Git provider is implemented and used end-to-end for policies
- OCI provider is implemented for bundles or planned with clear path
- UI and CLI show policies with validate and apply workflows
- Agents can link to policies and display enforcementMode
- Conformance suite passes

## Verification and test plan
Unit
- Bundle and metadata validation
- Rego syntax validation (basic)

Integration
- Provider sync persists policies and artifact pointers with digests
- validate and apply actions return clear diagnostics

E2E
- Load a policy bundle from Git
- Validate and apply via CLI
- View details and lifecycle state in UI
- Link policy to agent and verify it appears in dependencies panel
