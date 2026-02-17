# UI and CLI requirements for Phase 6

## Goal
Ensure every Phase 6 plugin can be managed through the generic UI components and the plugin-driven CLI v2 without plugin-specific code.

## UI requirements
### Universal list view
- Must render for every plugin using capabilities schema
- Must support:
  - paging
  - sorting (at least by name and updatedAt)
  - plugin-defined filter fields
  - source selection and visibility of source diagnostics
  - lifecycle state badges

### Universal detail view
- Must render:
  - universal metadata
  - plugin-specific fields
  - artifacts panel with digests, sizes, provenance
  - dependencies and links panel
  - diagnostics panel:
    - validation results
    - last sync status
    - last action status

### Actions bar
- Must show only actions declared by plugin capabilities
- Must enforce:
  - validate before apply (default policy)
  - confirmation for high-risk actions
- Must show progress and results:
  - startedAt, finishedAt
  - success/failure
  - structured error messages

### Accessibility and safety
- No unsafe rendering of template content or notebook content
- Guardrails, policies, and skills must display safety indicators clearly
- Avoid leaking secrets or tokens in UI

## CLI requirements
### Discovery
- List available plugins and their capabilities
- List sources per plugin and show sync health

### List and get
- Consistent output across plugins
- Support:
  - pagination
  - filter queries
  - output formats (table, json, yaml if supported)

### Actions execution
- Consistent verbs:
  - validate, apply, refresh
  - enable, disable (if supported)
  - tag, annotate, promote, deprecate, link
- Consistent exit codes and error messages

### Traceability
- Every action should return an actionRun id
- CLI should support "describe action-run" if the universal framework provides it

## Phase 6 acceptance criteria for UI and CLI
- At least 4 new plugins are usable end-to-end with UI and CLI
- No plugin-specific UI routes or CLI commands are introduced for those flows
- All action flows provide clear feedback and do not require log digging
