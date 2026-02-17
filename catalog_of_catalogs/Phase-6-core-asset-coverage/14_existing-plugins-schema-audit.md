# Existing plugin schema audits and alignment updates (Model, MCP, Knowledge Sources)

## Objective
Before and during Phase 6, confirm that the existing plugins:
- fully conform to the universal asset contract
- expose accurate capabilities that drive UI and CLI
- have schemas that capture the minimum metadata needed for real workflows
- can participate cleanly in linking, filtering, and action execution

This doc defines what to validate, what to update, and what tests must pass.

## Model plugin schema audit
### Deep research outputs required
Consult and summarize:
- Kubeflow model registry and catalog docs
- Model cards practices (fields for intended use, limitations, evaluation, ethical considerations)
- MLflow model signatures and input/output schema patterns

Research starting points:
- Kubeflow Model Catalog API: https://www.kubeflow.org/docs/components/model-registry/reference/model-catalog-rest-api/
- MLflow model signatures: https://mlflow.org/docs/latest/models/signatures.html
- Hugging Face model cards: https://huggingface.co/docs/hub/model-cards

### Minimum schema requirements
- Universal fields:
  - name, version, description
  - labels, annotations
  - lifecycle state and timestamps
  - provenance and license
- Model-specific fields:
  - modelType (llm, embedding, vision, multimodal, classical)
  - supportedModalities
  - supportedTasks
  - inferenceInterfaces (openai compatible, vllm, tgi, custom)
  - deploymentHints (optional)
  - artifact pointers and digests (if model is an artifact)
  - modelCardRef (artifact) for richer documentation

### Required checks
- Filter and search behavior for:
  - modality, tasks, modelType, lifecycle, license
- Capability reporting matches actual feature support

## MCP plugin schema audit
### Deep research outputs required
Consult and summarize:
- MCP server metadata conventions
- Tool schema patterns (JSON Schema 2020-12 input schemas)
- Local vs remote server representation conventions

Research starting points:
- MCP tools spec: https://modelcontextprotocol.io/specification/2025-06-18/server/tools

### Minimum schema requirements
- Universal fields
- MCP-specific fields:
  - serverType (local, remote)
  - local container image ref (for local)
  - remote baseUrl and authType (for remote)
  - supported transports
  - tool inventory metadata:
    - tool names
    - per-tool inputSchema (JSON Schema)
    - optional output schemas
  - health and connectivity hints

### Required checks
- UI and CLI must clearly show whether a server is local or remote
- Providers and sources must be able to populate real MCP entries, not placeholder data

## Knowledge Sources plugin schema audit
### Deep research outputs required
Consult and summarize:
- Common RAG source patterns
- Connection metadata patterns for databases and web sources
- Governance needs: access, pii, retention, ownership

### Minimum schema requirements
- Universal fields
- Knowledge source specific fields:
  - sourceType (files, web, vector_db, graph_db, sql_db, saas_connector)
  - connectionRef (secret ref only, no secrets)
  - indexingConfig (chunking, embedding model, refresh schedule)
  - contentFilters and access metadata
  - supported query modes (keyword, vector, hybrid)
  - artifacts (ingestion configs, sample queries)

### Required checks
- Agents can link to knowledge sources reliably
- UI and CLI can filter by sourceType, accessType, and refreshability

## Definition of done
- A written schema delta list exists for each plugin (fields to add, fields to rename, constraints to tighten)
- Backward compatibility plan exists for any breaking changes
- Conformance suite passes for Model, MCP, and Knowledge Sources
- At least one e2e scenario includes:
  - an Agent linking to Prompt Template, Skills, Guardrails, Policies, Knowledge Sources, MCP, and a Model reference

## Verification plan
- Schema validation unit tests per plugin
- Provider sync integration tests per plugin
- UI and CLI conformance tests for:
  - list and filters
  - details rendering
  - actions execution
  - link resolution
