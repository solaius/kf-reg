# Benchmarking plugin specification

## Purpose
Represent benchmark suites and tasks as discoverable assets so teams can standardize how they compare models, prompts, or agents.

Benchmarks link to Datasets, Evaluators, and optionally Models and Agents.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) MLPerf benchmark conventions
- How benchmarks capture rules, datasets, metrics, and reproducibility requirements

2) Task-based LLM evaluation harnesses
- How tasks are represented and invoked (lm-evaluation-harness style)

Research sources (starting points):
- MLPerf: https://mlcommons.org/en/inference-datacenter/
- lm-evaluation-harness: https://github.com/EleutherAI/lm-evaluation-harness

## Schema draft
### Entity: BenchmarkSuite
Required fields
- name
- version
- description
- benchmarkType (enum)
  - performance
  - quality
  - safety
  - reliability
- tasks (array of BenchmarkTask references or inline definitions)

Strongly recommended fields
- datasets (array of Dataset refs)
- evaluators (array of Evaluator refs)
- metrics (array)
  - { name, metricType, description, threshold?, higherIsBetter? }
- runConfig (object)
  - harnessType (enum: lm_eval_harness, custom, mlflow, other)
  - environment (object)
    - containerImageRef (optional)
    - hardwareRequirements (optional)
    - runtimeParameters (optional)
- reproducibility (object)
  - seed
  - deterministic (bool)
  - datasetPinning (version or digest)
  - evaluatorPinning (version or digest)
- governance (object)
  - allowedUses
  - disallowedUses
- provenance, license, owner fields

Artifacts
- harnessConfigFile
- taskDefinitionsFile
- documentation
- rulesFile

Filtering fields (minimum)
- name
- version
- benchmarkType
- metric names
- lifecycleState

## Providers
Baseline
- YAML provider for benchmark suite metadata and references

Recommended additions in Phase 6
- Git provider for benchmark-as-code repositories (highly recommended)
- HTTP provider optional for benchmark hubs

Validation requirements
- Validate referenced datasets and evaluators resolve
- Validate metrics schema and thresholds
- Validate runConfig is complete for the harnessType selected

## Actions and lifecycle
Supported actions (opt-in)
- validate: schema validation + reference resolution
- apply: persist metadata
- refresh: re-sync
- promote, deprecate, tag, annotate, link

Optional future actions (not required in Phase 6)
- run: benchmark execution is out of scope unless integrated with an evaluation system

## API surface
Must conform to the common plugin API patterns:
- /api/benchmark_catalog/v1alpha1/benchmarksuites
- /api/benchmark_catalog/v1alpha1/benchmarksuites/{id}
- /api/benchmark_catalog/v1alpha1/sources
- /api/benchmark_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works
- Git provider used for at least one benchmark catalog in tests
- UI and CLI can browse benchmark suites, see datasets and evaluators links, and validate/apply
- Conformance suite passes

## Verification and test plan
Unit
- Task and metric validation
- Reference resolver tests

Integration
- Sync persists benchmark suite entries and artifact pointers

E2E
- Load benchmark suite from Git
- Filter by benchmarkType in UI
- Validate and apply via CLI
- Link benchmark suite to an Agent and verify UI shows it in dependencies panel
