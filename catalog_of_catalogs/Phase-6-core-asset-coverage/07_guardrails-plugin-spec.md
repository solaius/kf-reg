# Guardrails plugin specification

## Purpose
Represent LLM guardrails and safety configurations as first-class assets that can be discovered, validated, versioned, and linked into agents.

Guardrails are expected to be packaged as configuration bundles (often multi-file) and may be delivered as OCI artifacts or Git-managed code.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) NeMo Guardrails
- What config files exist (config.yml, flows, Colang)
- How policies and rails are represented and enforced
- What metadata is needed to apply guardrails to an agent

2) Guardrails.ai and RAIL patterns
- How validators are defined and composed
- What metadata is needed to evaluate and enforce them

3) Enterprise safety requirements
- Risk categories, supported modalities
- Audit and provenance requirements

Research sources (starting points):
- NeMo Guardrails docs: https://docs.nvidia.com/nemo/guardrails/
- Guardrails.ai docs: https://www.guardrailsai.com/docs/

## Schema draft
### Entity: Guardrail
Required fields
- name
- version
- guardrailType (enum)
  - nemo_guardrails
  - guardrails_ai
  - regex_rules
  - policy_rules
  - moderation_profile
- enforcementStage (enum)
  - pre_prompt
  - post_generation
  - tool_use
  - retrieval
  - output_format
- description

Strongly recommended fields
- modalities (array: text, image, audio)
- riskCategories (array)
  - examples: pii, secrets, hate, violence, self_harm, harassment, malware, policy_violation
- enforcementMode (enum: advisory, required)
- configFormat (enum)
  - yaml
  - json
  - bundle (multi-file)
- configRef (artifact reference)
  - For nemo_guardrails: bundle including config.yml and flows
  - For guardrails_ai: rail spec and validators
- validationProfile (object)
  - requiredFiles
  - schemaChecks
  - lintChecks
- compatibleAgentTypes (array)
- compatibleModels (optional)
- license (SPDX expression)
- provenance (object)
  - upstreamUrl
  - upstreamRevision
  - author
  - createdAt

Artifacts
- guardrailBundle (recommended)
- testCases (recommended)
- documentation (optional)

Filtering fields (minimum)
- name
- version
- guardrailType
- enforcementStage
- riskCategories
- enforcementMode
- lifecycleState

## Providers
Baseline
- YAML provider loads Guardrail metadata, and optionally references a local bundle path

Recommended additions in Phase 6
- OCI provider for guardrail bundles published as artifacts
- Git provider for guardrail-as-code repositories

Validation requirements
- Validate that referenced bundle exists and required files are present
- Validate config syntax:
  - YAML well-formed
  - Optional: framework-specific linting if feasible
- Validate test cases exist when enforcementMode is required, or document why not

## Actions and lifecycle
Supported actions (opt-in)
- validate: bundle integrity, syntax validation, required files, optional lint checks
- apply: persist metadata and make bundle available
- enable, disable: if the platform supports enabling guardrails for a given scope
- refresh: re-sync from sources
- promote, deprecate: lifecycle transitions

Safety requirements
- No secret values inside guardrail configs
- If secrets are required, store secret references only

## API surface
Must conform to the common plugin API patterns:
- /api/guardrail_catalog/v1alpha1/guardrails
- /api/guardrail_catalog/v1alpha1/guardrails/{id}
- /api/guardrail_catalog/v1alpha1/sources
- /api/guardrail_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works and validates references
- OCI provider or Git provider is implemented and used end-to-end for guardrails
- UI and CLI can list, filter, view details, and execute validate/apply
- Agents can link to guardrails and display those links
- Conformance suite passes

## Verification and test plan
Unit
- Bundle reference validation and required file checks
- YAML parsing and schema validation

Integration
- Provider sync persists metadata and artifact pointers
- validate and apply actions produce diagnostics

E2E
- Load at least one guardrail bundle from Git or OCI
- Use CLI to validate and apply
- Link a guardrail to an Agent and verify UI shows it

Security
- Tests that secret-like values are rejected or flagged
