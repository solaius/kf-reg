# UI Product Spec

## UI goals

- Provide a unified catalog experience across all plugins
- Avoid hard coding asset types in the UI
- Provide role aware controls (viewer versus operator)
- Make discovery and troubleshooting fast

## Information architecture

Global navigation

- Catalog entry point
- Plugin switcher populated from plugin discovery endpoint
- Optional cross plugin search if supported

Per plugin views

- List view
  - search box
  - filter builder or filterQuery input
  - table or card view
  - pagination

- Detail view
  - overview metadata
  - artifacts section (conditional)
  - source provenance
  - copy reference action

Ops views

- Sources
  - list sources
  - add and edit source
  - enable and disable
  - validate and apply
  - refresh controls

- Status and diagnostics
  - plugin health
  - source refresh history
  - error details with next actions

## UI extensibility model

Generic rendering

- UI uses schema and capability hints to render basic list and detail views for any plugin
- UI uses a shared component library for
  - tables
  - key value metadata display
  - artifacts lists
  - empty states
  - errors

Plugin specific extensions

- Plugins can provide optional UI hints so the generic renderer can
  - choose which fields are primary
  - choose default columns
  - label artifact types
  - define a preferred identity field (name versus id)

- UI supports plugin specific custom pages only when needed
  - the default path should be generic rendering

## Accessibility and usability

- Support keyboard navigation
- Provide clear empty states with guidance
- Provide copyable CLI equivalents in key workflows

## Acceptance Criteria

- A new plugin appears in the UI automatically with generic list and detail rendering
- Ops can manage sources in UI end to end
- Viewer cannot see edit actions and cannot call management endpoints
