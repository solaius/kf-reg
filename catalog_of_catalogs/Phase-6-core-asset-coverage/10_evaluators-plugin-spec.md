# Evaluators plugin specification

## Purpose
Represent evaluators as reusable assets that define how to measure the quality, safety, and correctness of model outputs or agent behavior.

Evaluators must be linkable to Agents and Benchmarks.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) MLflow LLM evaluation patterns
- How MLflow structures evaluation inputs, metrics, and results
- Common evaluator types for LLMs (judge models, heuristic checks, embedding similarity, toxicity)

2) Task evaluation harness patterns
- How evaluators are packaged and executed
- How to represent metric definitions and thresholds

Research sources (starting points):
- MLflow LLM evaluation: https://mlflow.org/docs/latest/llms/llm-evaluate/index.html
- lm-evaluation-harness: https://github.com/EleutherAI/lm-evaluation-harness

## Schema draft
### Entity: Evaluator
Required fields
- name
- version
- description
- evaluatorType (enum)
  - llm_judge
  - heuristic
  - embedding_similarity
  - policy_check
  - schema_validation
- inputSchema (optional JSON Schema 2020-12)
- outputSchema (optional JSON Schema 2020-12)

Strongly recommended fields
- metrics (array)
  - { name, metricType, description, range?, higherIsBetter?, threshold? }
- implementation (object)
  - implType (enum)
    - prompt_based
    - code
    - remote_service
  - promptTemplateRef (optional reference to PromptTemplate)
  - codeRef (optional artifact reference)
  - endpoint (optional for remote_service)
- judgeModelConfig (optional)
  - modelRef or model family name
  - promptTemplateRef for judge rubric
- testCasesRef (artifact reference)
- compatibility (object)
  - supportedModalities
  - supportedTasks
- provenance, license, owner fields

Artifacts
- evaluatorCodePackage (optional)
- rubricPrompt (optional)
- testCases (recommended)
- documentation (optional)

Filtering fields (minimum)
- name
- version
- evaluatorType
- metric names
- lifecycleState

## Providers
Baseline
- YAML provider for evaluator metadata and optional references to artifacts

Recommended additions in Phase 6
- Git provider for evaluator-as-code repositories
- HTTP provider for remote evaluator catalogs (optional)
- OCI provider for code packages (optional and only if safe)

Validation requirements
- Validate metric definitions are complete and thresholds are sane
- Validate referenced prompt templates and models resolve
- Validate code packages are referenced by digest if external

## Actions and lifecycle
Supported actions (opt-in)
- validate: schema validation, reference resolution, optional static checks on code artifacts
- apply: persist metadata and make available
- refresh: re-sync
- promote, deprecate, tag, annotate, link

Optional future actions (not required in Phase 6)
- run: executing evaluators is out of scope unless already integrated with an evaluation service

## API surface
Must conform to the common plugin API patterns:
- /api/evaluator_catalog/v1alpha1/evaluators
- /api/evaluator_catalog/v1alpha1/evaluators/{id}
- /api/evaluator_catalog/v1alpha1/sources
- /api/evaluator_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works and validates schemas
- At least one evaluator catalog is loaded from a real source (Git preferred)
- UI and CLI can browse evaluators, see metrics, and link evaluators to agents
- Conformance suite passes

## Verification and test plan
Unit
- Metric schema validation
- Reference resolution checks

Integration
- Sync persists evaluators and artifacts pointers
- validate and apply actions return diagnostics

E2E
- Load evaluator definitions from Git
- Browse and filter evaluators in UI by evaluatorType
- Validate and apply via CLI
- Link evaluator to an Agent and verify UI dependencies panel updates
