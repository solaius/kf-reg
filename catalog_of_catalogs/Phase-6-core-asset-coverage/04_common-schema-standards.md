# Common schema standards and research checklist

## Why this exists
Phase 6 requires "deep research" before finalizing each plugin schema. This file centralizes the baseline standards and patterns each plugin should align to, plus a repeatable research checklist for Claude Code to execute per plugin.

## Baseline standards (recommended defaults)
### JSON Schema dialect
- Use JSON Schema Draft 2020-12 for embedded schemas (inputSchema, outputSchema, parameter schemas)
- When embedding schemas, always declare $schema unless the project enforces a default
- Prefer structural, machine-validated schemas that can power UI form generation and validations

References:
- JSON Schema Draft 2020-12: https://json-schema.org/draft/2020-12
- MCP tools schema defaults to 2020-12 when $schema is absent: https://modelcontextprotocol.io/specification/2025-06-18/server/tools
- OpenAPI 3.1 aligns with JSON Schema and supports jsonSchemaDialect: https://spec.openapis.org/oas/v3.1.0

### Licensing identifiers
- Store license using SPDX license expressions where possible
- Always capture attribution and upstream source

References:
- Fedora SPDX docs: https://docs.fedoraproject.org/en-US/legal/spdx/
- SPDX license list data: https://github.com/spdx/license-list-data

### Dataset metadata standards
Prefer capturing dataset metadata with fields that can map cleanly to:
- MLCommons Croissant (dataset metadata in JSON-LD)
- Hugging Face dataset cards (YAML front matter patterns)
- schema.org Dataset
- Datasheets for Datasets (human-facing documentation fields)

References:
- Croissant spec: https://github.com/mlcommons/croissant
- Hugging Face dataset cards: https://huggingface.co/docs/hub/datasets-cards
- schema.org Dataset: https://schema.org/Dataset
- Datasheets for Datasets paper: https://arxiv.org/abs/1803.09010

### Policies and guardrails standards
- Policy-as-code conventions: Open Policy Agent (Rego) and bundles
- Guardrails frameworks and config patterns:
  - NeMo Guardrails config and Colang flows
  - Guardrails.ai validators and RAIL specs (where applicable)

References:
- OPA docs: https://www.openpolicyagent.org/docs/latest/
- OPA bundles: https://www.openpolicyagent.org/docs/latest/management-bundles/
- NeMo Guardrails docs: https://docs.nvidia.com/nemo/guardrails/
- Guardrails.ai docs: https://www.guardrailsai.com/docs/

### Notebooks
- Jupyter notebook format is nbformat JSON
- Treat notebooks as artifacts, validate nbformat and capture kernel and language metadata

References:
- nbformat docs: https://nbformat.readthedocs.io/en/latest/format_description.html

### Benchmarks and evaluation harnesses
- Evaluation: MLflow evaluate for LLM evaluation patterns and metrics
- Benchmark suites: MLPerf patterns for benchmark definition and reproducibility
- Task-based eval suites: lm-evaluation-harness patterns

References:
- MLflow evaluate docs: https://mlflow.org/docs/latest/llms/llm-evaluate/index.html
- MLPerf: https://mlcommons.org/en/inference-datacenter/
- lm-evaluation-harness: https://github.com/EleutherAI/lm-evaluation-harness

### Kubeflow Model Catalog patterns
- Model Catalog is a federated metadata aggregation layer for read-only discovery across multiple sources
- Schemas and APIs should remain backward compatible where required

Reference:
- Kubeflow Model Catalog REST API: https://www.kubeflow.org/docs/components/model-registry/reference/model-catalog-rest-api/

## Repeatable deep research checklist per plugin
For each asset plugin in Phase 6, execute:
1) Identify 2 to 5 authoritative sources describing the asset type's real-world metadata needs
2) Identify de facto standards and schemas
3) Enumerate the minimum viable fields that enable real workflows
4) Enumerate optional fields that future-proof the schema, but do not overfit to one vendor
5) Define artifact representation (inline vs external) and how digests and provenance are captured
6) Map the schema to the universal asset contract:
   - lifecycle fields
   - labels and annotations
   - capabilities and actions
7) Define provider requirements:
   - YAML baseline shape
   - which additional providers (HTTP, Git, OCI) make sense and why
8) Define validation rules:
   - schema validation
   - integrity checks
   - security checks (secrets, unsafe commands)
9) Define filtering and indexing strategy:
   - which fields are filterable
   - which are searchable
10) Define tests:
   - unit tests for schema and provider parsing
   - integration tests for loader persistence
   - UI and CLI conformance tests

## Definition of done
- Every Phase 6 plugin spec includes a "Research outputs" section with the results of steps 1 to 4
- Every Phase 6 plugin spec includes a "Schema draft" section informed by the research outputs
- Every Phase 6 plugin spec includes a "Validation and tests" section that can be executed in CI
