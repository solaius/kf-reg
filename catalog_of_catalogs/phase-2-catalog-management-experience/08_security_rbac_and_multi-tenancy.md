# Security, RBAC, and Multi Tenancy

## Threat model focus

- Prevent unauthorized source modifications
- Prevent information leakage across tenants where multi user mode applies
- Ensure auditability of changes

## Requirements

Authentication

- UI and CLI requests must be authenticated in real deployments
- Local dev can support mock auth flows for development

Authorization

- Viewer role can read
  - plugins
  - assets
  - sources (read only)

- Operator role can
  - create, update, delete sources
  - enable, disable sources
  - trigger refresh
  - view diagnostics

Server side enforcement

- Management endpoints must enforce operator role
- Errors should be explicit and consistent

Namespace and profile boundaries

- If the platform uses namespace scoping, catalog management operations must be scoped appropriately
- Cross namespace operations should be disabled by default unless explicitly configured

Auditability

- Source changes should be traceable
  - who
  - when
  - what changed

## Acceptance Criteria

- A viewer cannot perform any source mutation via UI or CLI
- An operator can perform full source management in UI and CLI
- Unauthorized attempts produce consistent error codes and messages
