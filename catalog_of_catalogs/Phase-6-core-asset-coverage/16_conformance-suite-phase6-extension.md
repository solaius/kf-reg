# Conformance suite extensions for Phase 6

## Objective
Extend the existing conformance suite so every new plugin and provider type can be validated automatically.

Phase 6 adds two dimensions:
- More plugin types (new schemas, new actions)
- More provider types (HTTP, Git, OCI)

## Conformance levels
### Level 0: Schema conformance
- Plugin exposes capabilities schema entry
- Universal asset contract fields exist and validate
- Required plugin fields exist and validate
- Filter fields declared by plugin exist in schema

### Level 1: Provider conformance (per provider type)
YAML provider
- Loads a minimal catalog file
- Rejects invalid schema
- Reports diagnostics

HTTP provider
- Handles pagination or documents limitations
- Implements timeouts and retry policies
- Rejects invalid TLS by default
- Reports diagnostics

Git provider
- Captures commit SHA and file provenance
- Respects include and exclude rules
- Reports diagnostics

OCI provider
- Pulls artifacts by digest
- Verifies digests
- Reports artifactType and annotations in diagnostics

### Level 2: Actions conformance
- validate works and returns structured diagnostics
- apply works and persists state changes
- refresh works for the plugin
- lifecycle actions promote and deprecate enforce rules

### Level 3: UI and CLI conformance
- CLI can list and get assets using capabilities-driven routing
- CLI can execute validate and apply
- UI can render list and detail views using generic components
- UI can execute validate and apply with feedback

## Phase 6 conformance test matrix
For each shipped plugin:
- Schema conformance
- YAML provider conformance
- At least one non-YAML provider conformance if the plugin claims support
- Actions conformance for validate/apply/refresh
- UI and CLI conformance smoke tests

For each provider type:
- Contract tests with fake sources
- Integration tests with a minimal real source scenario

## Required test fixtures
- A minimal catalog set per plugin:
  - valid catalog
  - invalid catalog with expected errors
- A minimal provider source config per provider type:
  - valid config
  - invalid config
- A golden set of expected list outputs for CLI table mode

## Definition of done
- CI runs conformance suite for all shipped Phase 6 plugins
- Conformance suite failures are actionable, with clear error messages
- A new plugin can be added with a small fixture set and pass conformance without manual UI work
