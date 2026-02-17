# Prompt Templates plugin specification

## Purpose
Represent reusable prompt templates as first-class assets that can be discovered, versioned, validated, and linked into agents and other workflows.

This plugin should support both:
- Human-authored templates (catalog-as-code)
- Machine-validated templates (schema-validated variables and output structure)

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) MLflow LLM prompt engineering and prompt registry concepts
- Capture what metadata MLflow expects around prompts (template text, parameters, inference configuration, model compatibility)
- Identify fields that support evaluation and iteration

2) MCP prompts (server-provided prompt templates)
- Capture how MCP represents prompts with arguments and structured messages

3) Common prompt template practices from major agent frameworks
- Capture a vendor-neutral minimum set: template format, roles, variables, examples, safety notes, evaluation tags

Research sources (starting points):
- MLflow prompt engineering docs: https://mlflow.org/docs/latest/llms/prompt-engineering/index.html
- MCP prompts spec: https://modelcontextprotocol.io/specification/2025-06-18/server/prompts

## Schema draft
### Entity: PromptTemplate
Required fields
- name
- version (string, semver allowed but not required)
- format (enum)
  - chat_messages
  - completion
  - jinja2
  - handlebars
  - raw
- template
  - For chat_messages: array of { role, content, contentType? }
  - For completion: string
- parametersSchema (JSON Schema 2020-12)
  - Defines template variables and their types
- outputSchema (optional JSON Schema 2020-12)
  - If present, indicates the expected structured output

Strongly recommended fields
- description
- taskTags (array of strings)
- modalities (array, examples: text, image, audio)
- modelConstraints (object)
  - compatibleModelFamilies
  - minContextTokens
  - maxOutputTokens
  - requiredTools (references to Skills assets)
- examples (array)
  - { inputs, expectedOutput?, notes? }
- evaluationHints (object)
  - recommendedMetrics
  - testCasesRef (artifact reference)
- safetyNotes (string)
- license (SPDX expression)
- sourceProvenance (object)
  - upstreamUrl
  - upstreamRevision
  - author
  - createdAt

Artifacts
- templateFile (optional) for large templates or multi-file templates
- testCases (optional) for evaluation harness inputs
- attachments (optional)

Linking
- Can be referenced by Agents
- Can reference Guardrails and Policies (optional) for recommended usage constraints

Filtering fields (minimum)
- name
- version
- format
- taskTags
- modalities
- lifecycleState
- labels

Notes
- Keep template content out of query filters by default
- Provide a text search field if search indexing exists, otherwise rely on name, tags, description

## Providers
Baseline
- YAML provider loads PromptTemplate definitions from local YAML files

Recommended additions in Phase 6
- Git provider for catalog-as-code prompt libraries (teams will likely manage prompts in Git)
- HTTP provider for pulling prompt catalogs from remote endpoints (optional early)

Provider-specific validation
- Validate JSON Schema fields (parametersSchema, outputSchema) against draft 2020-12
- Validate template placeholders match parametersSchema keys
- Validate chat role values against allowed set (system, user, assistant, tool)

## Actions and lifecycle
Supported actions (opt-in via capabilities)
- validate: check schema, placeholders, and optional output schema
- apply: write asset and artifacts into the DB and make it available
- refresh: re-sync from sources

Universal asset actions
- tag, annotate, deprecate, promote
- link: allow linking to Agents, Guardrails, Policies, Knowledge Sources, Skills

Lifecycle guidance
- draft: default for newly loaded assets
- active: promoted templates used by agents
- deprecated: available but discouraged

Safety requirements
- Validation must detect missing parameters, unused parameters, and schema invalidity
- Apply should be blocked if validate fails, unless forced by an explicit flag and audit reason

## API surface
Must conform to the common plugin API patterns:
- /api/prompt_template_catalog/v1alpha1/prompttemplates
  - list with filterQuery and pagination
- /api/prompt_template_catalog/v1alpha1/prompttemplates/{id}
  - get
- /api/prompt_template_catalog/v1alpha1/sources
  - list sources, diagnostics
- /api/prompt_template_catalog/v1alpha1/actions/{action}
  - execute universal actions when enabled

## Definition of done
- Plugin generated with catalog-gen and wired into catalog-server
- CRUD is not required if management actions cover required flows
- YAML provider works and is validated
- UI and CLI show Prompt Templates with:
  - list view, filters, detail view, artifacts
  - execute validate, apply, refresh
- Linking to Agents works via universal link action
- Conformance suite passes for the plugin

## Verification and test plan
Unit
- Schema validation for parametersSchema and outputSchema
- Placeholder extraction and matching against parametersSchema
- Provider parsing tests with negative cases

Integration
- Loader writes correct DB records and artifacts references
- Actions endpoint performs validate/apply/refresh and returns diagnostics

E2E
- Load a sample prompt catalog from YAML and from Git
- Use UI to list and filter by format and tags
- Use CLI to validate and apply a template
- Confirm details view renders template content safely (no XSS)

Regression
- Ensure no changes to Model Catalog and MCP Catalog routes and behaviors
