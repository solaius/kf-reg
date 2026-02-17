# Datasets plugin specification

## Purpose
Represent datasets as discoverable assets with rich metadata for governance, compliance, and ML or LLM workflows.

This catalog is metadata-first. It may optionally link to dataset locations, but does not host datasets.

## Deep research outputs (required)
Authoritative inputs to consult and the conclusions to capture:

1) MLCommons Croissant
- Key fields for dataset description, distributions, licensing, citations, and provenance

2) Hugging Face dataset cards
- Common metadata fields teams already use: languages, tags, size categories, licensing, task categories

3) schema.org Dataset
- Minimal structured representation for dataset metadata

4) Datasheets for Datasets
- Human-facing documentation fields that capture intended use, biases, collection process

Research sources (starting points):
- Croissant: https://github.com/mlcommons/croissant
- HF dataset cards: https://huggingface.co/docs/hub/datasets-cards
- schema.org Dataset: https://schema.org/Dataset
- Datasheets for Datasets: https://arxiv.org/abs/1803.09010

## Schema draft
### Entity: Dataset
Required fields
- name
- version
- description
- datasetType (enum)
  - tabular
  - text
  - image
  - audio
  - video
  - multimodal
- license (SPDX expression when possible)

Strongly recommended fields
- modalities (array)
- languages (array)
- tasks (array)
- size (object)
  - numRecords
  - numBytes
  - sizeCategory (enum similar to HF, optional)
- splits (array)
  - { name, numRecords, numBytes }
- schema (object)
  - schemaFormat (enum: jsonschema, parquet, avro, unknown)
  - schemaRef (artifact reference)
- distributions (array)
  - { uri, format, checksum?, accessType? }
- access (object)
  - accessType (enum: public, internal, restricted)
  - dataResidency (optional)
  - pii (enum: none, suspected, confirmed)
  - sensitiveCategories (array)
- provenance (object)
  - source
  - collectionMethod (optional)
  - createdAt
  - updatedAt
  - citation (optional)
- governance (object)
  - allowedUses (array)
  - disallowedUses (array)
  - retentionPolicy (optional)
  - deletionPolicy (optional)
- documentation (object)
  - datasheetRef (artifact reference, optional)
  - knownBiases
  - limitations
  - recommendedPreprocessing
- owner and supportContact
- labels and annotations (universal)

Artifacts
- croissantMetadata (optional artifact)
- datasetCard (optional artifact)
- schemaFile (optional artifact)
- datasheet (optional artifact)

Filtering fields (minimum)
- name
- version
- datasetType
- modalities
- languages
- license
- access.accessType
- access.pii
- lifecycleState

## Providers
Baseline
- YAML provider for Dataset metadata

Recommended additions in Phase 6
- HTTP provider for remote dataset catalogs (optional)
- Git provider for dataset metadata repositories (recommended)

Validation requirements
- Validate license expression where possible
- Validate distributions URIs are well-formed
- Validate schemaRef if schemaFormat is jsonschema
- Validate access fields are present for restricted datasets

## Actions and lifecycle
Supported actions (opt-in)
- validate: schema validation and metadata completeness checks
- apply: persist metadata into DB
- refresh: re-sync from sources
- promote, deprecate: lifecycle transitions
- tag, annotate, link

Optional future actions (not required in Phase 6)
- materialize or copy dataset: out of scope

## API surface
Must conform to the common plugin API patterns:
- /api/dataset_catalog/v1alpha1/datasets
- /api/dataset_catalog/v1alpha1/datasets/{id}
- /api/dataset_catalog/v1alpha1/sources
- /api/dataset_catalog/v1alpha1/actions/{action}

## Definition of done
- Plugin generated and wired into catalog-server
- YAML provider works with validation
- Git provider is implemented or planned, and used for at least one dataset metadata repo in tests
- UI and CLI can browse and filter datasets by type, license, access, and pii
- Conformance suite passes

## Verification and test plan
Unit
- License validation and access metadata validation
- Parsing of splits and distributions

Integration
- Provider sync persists dataset entries and artifact pointers

E2E
- Load dataset metadata from YAML or Git
- Filter by datasetType and accessType in UI
- Validate and apply via CLI
- Verify details include provenance and documentation
