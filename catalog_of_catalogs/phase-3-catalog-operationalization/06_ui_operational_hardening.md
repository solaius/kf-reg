# UI: Operational Hardening for Catalog Management and MCP Catalog

_Last updated: 2026-02-16_

## Goals
- Make UI pages accurate with live server data
- Remove placeholders where they mislead users
- Make actions predictable and safe for Ops users

## Catalog Management (Ops)
Required behaviors:
- Plugins page
  - shows plugins list and health
  - shows entity type badges
  - link into plugin details
- Plugin detail (Sources tab)
  - shows sources with status, entity counts, last refresh, errors
  - enable/disable toggle gated by RBAC
  - add source modal supports validate then apply
  - delete requires confirmation and shows impact statement
- Diagnostics tab
  - top-level plugin diagnostics
  - per-source diagnostics accessible from sources table

## MCP Catalog (AI Engineer)
Required behaviors:
- Gallery
  - search by name + description
  - filters: local/remote, transport, license, labels
  - counts reflect filtered set
- Detail page
  - overview tab includes connection info
  - show local image URI when local
  - show remote base URL when remote
  - show auth hint if present

## Role gating
- viewer sees actions disabled with tooltip explaining “operator role required”
- operator can perform mutations and sees success toast

## Logo strategy
- If logo URL is present, show it
- If missing, show consistent placeholder icon
- Do not require logo availability for correctness

## Definition of Done
- All pages render correctly with real mode integrations
- All action buttons are role-gated correctly
- No placeholder rows, counts, or statuses in real mode

## Acceptance Criteria
- AC1: Plugin sources table shows real status and count fields
- AC2: Add source validates, applies, and source appears with correct status
- AC3: Viewer cannot mutate and sees clear affordances
- AC4: MCP gallery filters operate on live data and remain responsive
