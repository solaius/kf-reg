# Functional Requirements

This section defines the requirements for Phase 2. It intentionally focuses on outcomes and behavior, not implementation.

## A. Catalog Discovery (UI and CLI)

A1. Plugin inventory

- UI and CLI can list installed plugins
- Each plugin shows
  - Name
  - Version (if available)
  - API base path
  - Health state
  - Supported entity types and artifacts (if discoverable)

A2. Cross plugin navigation

- UI supports switching between asset types without a hard coded list
- UI shows only plugins that exist at runtime

A3. List and filter

- UI and CLI support list and filter for entities in each plugin
- Filtering supports the existing filterQuery semantics
- Pagination behavior is consistent across plugins

A4. Details and artifacts

- UI and CLI can fetch a single entity by ID or name (plugin specific)
- UI and CLI can list artifacts attached to an entity when the plugin supports artifacts
- UI supports deep links that include plugin and entity identity

## B. Source Management (Ops focused, UI and CLI)

B1. List sources

- UI and CLI can list sources per plugin
- Each source shows
  - ID
  - Provider type (YAML, HTTP, etc)
  - Enabled or disabled
  - Include or exclude globs if present
  - Last refresh status and timestamp

B2. Validate config

- UI and CLI can validate a proposed source change before applying it
- Validation returns actionable error messages

B3. Apply changes

- UI and CLI can create, update, enable, disable, and delete sources if supported by the management plane
- If a deployment chooses a file only config model, UI and CLI must still support
  - Displaying the effective config
  - Producing a patch or suggested YAML snippet that an operator can apply

B4. Trigger refresh

- UI and CLI can trigger refresh per source and per plugin
- UI and CLI can show refresh progress and final outcome

## C. Operational Status and Diagnostics

C1. Plugin health

- Existing /healthz and /readyz remain
- A plugin status endpoint exists with per plugin health plus last init and last refresh info

C2. Ingestion status

- Per plugin and per source status includes
  - Last successful refresh time
  - Last attempted refresh time
  - Error summary and a pointer to detailed logs

C3. Error surfaces

- UI provides a diagnostics view for ops
- CLI provides a diagnostics command that prints structured output

## D. RBAC and Multi Tenancy

D1. Roles

- Viewer role
  - Can list plugins and assets
  - Can view details
  - Cannot change sources

- Operator role
  - All viewer capabilities
  - Can manage sources and trigger refresh
  - Can view diagnostics

D2. Enforcement

- Server side enforcement exists for management endpoints
- UI and CLI hide or disable management actions when unauthorized

## E. Extensibility

E1. New plugin minimal path

- Scaffold plugin with catalog-gen
- Add blank import in server main
- Add sources config
- Build and run
- UI and CLI automatically surface the new plugin without code changes for basic list and detail views

E2. Override path

- A plugin can provide custom list and detail presentation hints to UI and CLI
- A plugin can customize provider behavior via hooks or templates without forking generated files

## Acceptance Criteria

- Demo: add a new source, validate, apply, refresh, and browse resulting assets in UI and CLI
- Demo: disable a source, verify assets removed or marked stale according to policy
- Demo: introduce a malformed config and show validation and safe failure behavior
- Demo: add a new generated plugin type and have it appear in UI and CLI with no manual UI coding for basic views
