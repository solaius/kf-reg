# 02_yaml_validation_safety_and_rollback

**Date**: 2026-02-16  
**Owner**: catalog-server plus BFF plus UI  
**Goal**: Prevent invalid edits from breaking the catalog and provide a safe rollback path

## Problem statement

Edits are currently treated as plain text with no validation prior to save. A typo can break parsing or produce silent partial loads, leaving the UI and CLI in confusing states.

We need strict validation, meaningful error feedback, and a rollback workflow.

## Requirements

### R1: Add server-side validation as the source of truth

Add a management-plane endpoint that validates a candidate YAML payload without applying it:
- POST /api/catalog-management/v1alpha1/plugins/{plugin}/sources/{sourceId}:validate
- Request: { rawYaml, options }
- Response: { valid, errors[], warnings[], normalizedYaml? }

Validation must include:
- YAML syntax correctness
- Strict field checking where structs are used (unknown fields should error)
- Plugin-specific semantic validation (required keys, allowed enums, constraints)

Implementation note for Go YAML strictness:
- Use yaml.Decoder with KnownFields(true) when decoding into structs

**Design decision (confirmed in review):** Unknown fields must fail validation for both the top-level `sources.yaml` structure and plugin-specific config blocks. This is enforced at two levels:

1. **Framework level** - The `StrictFieldsLayer` in the multi-layer validator (`pkg/catalog/plugin/validator.go`) checks the top-level source config structure using `yaml.Decoder` with `KnownFields(true)`
2. **Plugin level** - Each plugin's `ValidateSource()` method validates plugin-specific content. For MCP, `catalog/plugins/mcp/management.go` defines `mcpServerStrictEntry` (19 known fields) and uses `yaml.NewDecoder` with `KnownFields(true)` to reject unknown fields in `mcpservers` entries. Plugin-specific validation is invoked via the `ProviderLayer` in the validation pipeline.

The full validation pipeline order: YAML parse -> Strict fields -> Semantic -> Security warnings -> Provider (plugin-specific)

### R2: Add apply-time safety checks

On Apply:
- Re-run validation server-side (never trust client)
- If invalid, reject and return the full error set
- If valid, apply with an atomic write or atomic resource update

### R3: Add rollback support

Provide:
- GET /api/catalog-management/v1alpha1/plugins/{plugin}/sources/{sourceId}/revisions
- POST /api/catalog-management/v1alpha1/plugins/{plugin}/sources/{sourceId}:rollback with a revisionId

Rollback must:
- restore the exact previous payload
- trigger a refresh (or mark refresh required) so the catalog reflects the rollback immediately

### R4: UI validation and error UX

UI changes:
- Add a Validate action next to Save
- Save should be disabled when current content is known-invalid
- Show errors inline with a collapsible details view
- Show warnings non-blocking

Toast behavior should follow PatternFly guidance:
- concise titles
- avoid filler words like "successfully"

### R5: Diff and confirmation for risky edits (optional but recommended)

For apply:
- show a diff summary (added or removed keys, line counts)
- require confirmation when edits change critical fields (example: provider type, yamlCatalogPath, loader config pointers)

## Validation layers (implemented order)

1. **YAMLParseLayer** - YAML syntax correctness
2. **StrictFieldsLayer** - Structural decode with `KnownFields(true)`, rejects unknown fields
3. **SemanticLayer** - Required fields, allowed values, constraints
4. **SecurityWarningsLayer** - Warns (not errors) when sensitive values are inlined instead of using SecretRef. Uses `WarningOnly: true` flag to route issues to Warnings instead of Errors
5. **ProviderLayer** - Plugin-specific validation via `ValidateSource()`. For MCP, this includes strict field checking on the content block (mcpservers entries) using a dedicated `mcpServerStrictEntry` struct

## Acceptance criteria

- Invalid YAML cannot be applied
- Unknown fields in top-level source config produce an error (not silently ignored)
- Unknown fields in plugin-specific content blocks (e.g., MCP server entries) produce an error via ProviderLayer validation
- Inline sensitive values produce validation warnings (not errors) suggesting SecretRef usage
- Validation errors are shown in UI without leaving the page
- Rollback restores a previous revision and results in the catalog reflecting the rollback after refresh
- Revision history retains at least the last N revisions (choose N, example: 20)

## Definition of Done

- validate endpoint implemented with tests
- apply path enforces validation with tests
- revision list and rollback implemented with tests
- UI validation and error display implemented with manual verification steps documented
- No regression to existing model catalog endpoints or plugin behavior

## References

- go-yaml v3 strict checking via Decoder.KnownFields  
  https://github.com/go-yaml/yaml/blob/v3/yaml.go  
- PatternFly alert content guidance for toast notifications  
  https://www.patternfly.org/components/alert/design-guidelines  
