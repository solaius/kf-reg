# Notebooks plugin specification

## Purpose
Represent notebooks as cataloged artifacts for discovery and reuse, with enough metadata to understand runtime requirements and safe handling.

This plugin treats notebooks as artifacts and does not attempt to execute them in Phase 6.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) Jupyter nbformat specification
- Required fields and versioning rules
- How to validate a notebook and detect risky content

2) Enterprise notebook governance
- Capturing dependencies, kernel, language
- Detecting outputs, secrets, and external calls

Research source:
- nbformat: https://nbformat.readthedocs.io/en/latest/format_description.html

## Schema draft
### Entity: Notebook
Required fields
- name
- version
- description
- notebookRef (artifact reference)
  - ipynb file location or external artifact pointer

Strongly recommended fields
- nbformatVersion (string or int)
- kernelSpec (object)
  - name
  - displayName
  - language
- languageInfo (object)
  - name
  - version (optional)
- dependencies (array)
  - { type: pip|conda|system, spec }
- intendedUse (string)
- dataDependencies (array of Dataset refs, optional)
- modelDependencies (array of Model refs, optional)
- safety (object)
  - hasOutputs (bool)
  - outputsStripped (bool)
  - containsSecretsSuspected (bool)
  - networkAccessRequired (bool)
  - externalUris (optional list)
- provenance, license, owner fields

Artifacts
- notebookFile (required)
- requirementsFile (optional)
- environmentFile (optional)
- docs (optional)

Filtering fields (minimum)
- name
- version
- kernelSpec.language
- safety flags
- lifecycleState

## Providers
Baseline
- YAML provider loads notebook metadata and references local notebook files

Recommended additions in Phase 6
- Git provider for notebooks-as-code repositories (highly recommended)
- OCI provider optional for distributing notebooks as artifacts

Validation requirements
- Validate notebook file is valid JSON and matches nbformat schema
- Flag notebooks containing outputs unless explicitly allowed
- Flag suspected secret patterns
- Extract kernel and language metadata for indexing

## Actions and lifecycle
Supported actions (opt-in)
- validate: nbformat validation, safety scanning, dependency extraction
- apply: persist metadata and artifact pointers
- refresh: re-sync from sources
- promote, deprecate, tag, annotate, link

Optional future actions
- execute: out of scope in Phase 6

## API surface
Must conform to the common plugin API patterns:
- /api/notebook_catalog/v1alpha1/notebooks
- /api/notebook_catalog/v1alpha1/notebooks/{id}
- /api/notebook_catalog/v1alpha1/sources
- /api/notebook_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works
- Git provider used end-to-end for at least one notebook catalog in tests
- UI can list notebooks and show key safety metadata
- CLI can validate and apply notebooks
- Conformance suite passes

## Verification and test plan
Unit
- nbformat validation tests (valid and invalid notebooks)
- safety scan tests for outputs and secret patterns

Integration
- Sync persists notebook entries with extracted metadata

E2E
- Load notebooks from Git
- Filter by language and safety flags in UI
- Validate and apply via CLI
