# 04 UI hints spec (plugin-driven UI without bespoke code)

## Objective

Make UI and CLI able to render and operate on assets using plugin-provided metadata:
- field display types
- grouping and layout
- searchable fields and facets
- default filters and sorting
- action presentation

## Where UI hints live

Preferred: extend the existing **capabilities endpoint** to include UI hints.
Secondary: embed vendor extensions in OpenAPI via `x-...` fields, then generate the same hints automatically.

OpenAPI explicitly supports vendor extensions (x-*).

## UI hints schema: top level

```yaml
ui:
  listView:
    titleField: "displayName"
    columns:
      - field: "name"
        label: "Name"
        display: "link"
      - field: "owner"
        label: "Owner"
        display: "user"
      - field: "status.state"
        label: "State"
        display: "badge"
    defaultSort:
      field: "metadata.updatedAt"
      direction: "desc"
    defaultFilters:
      - filterQuery: "lifecycle.state != 'archived'"
  search:
    searchableFields: ["name", "description", "owner", "labels.*"]
    facets:
      - field: "lifecycle.state"
        display: "pill"
      - field: "riskLevel"
        display: "dropdown"
  detailView:
    sections:
      - title: "Summary"
        fields: ["name", "displayName", "description", "owner", "team"]
      - title: "Governance"
        fields: ["riskLevel", "intendedUse", "complianceTags"]
      - title: "Diagnostics"
        panels: ["auditTrail", "refreshStatus"]
  actions:
    primary: ["apply", "refresh"]
    secondary: ["enable", "disable", "deprecate", "promote"]
    confirmations:
      - action: "archive"
        prompt: "Archive this asset? This hides it from default views."
```

## Field display types

Standard display types:
- text, markdown
- badge (enum/known values)
- tags (array)
- link (URL)
- repoRef (org/repo)
- imageRef (OCI image)
- dateTime
- code (monospace)
- json (pretty)
- secretRef (redacted display; never show values)

## Search semantics

Search must be backed by server-side filtering:
- UI hints declare which fields are searchable/facetable
- server must expose mappings or support those fields in filterQuery

## Definition of Done

- A versioned UI hints schema exists
- Capabilities endpoint includes UI hints
- UI and CLI can render a new plugin using only capabilities + OpenAPI metadata

## Acceptance Criteria

- A brand-new plugin can render list + detail pages and actions without UI/CLI code changes
- UI hints do not allow unsafe behavior (e.g., secret leakage)

## Verification plan

- Conformance suite validates UI hints schema shape
- Add a toy plugin with nested/array fields and ensure generic UI renders correctly

## Notes on schema-driven forms

If we choose schema-driven editing screens:
- JSON Schema + uiSchema patterns are widely used to separate “what” from “how” for forms.
