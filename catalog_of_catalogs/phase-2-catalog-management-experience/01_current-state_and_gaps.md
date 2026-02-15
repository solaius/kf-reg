# Current State and Gaps

## What is already in place (Phase 1 outputs)

Backend

- A unified catalog-server process that can host multiple catalog plugins
- Plugin lifecycle with init, migrations, route registration, start, and health checks
- Model catalog wrapped as a plugin with no API breaking changes
- MCP plugin generated via catalog-gen with YAML provider, persistence, and list and get endpoints
- OpenAPI merge pipeline that produces a unified catalog spec

UI backend for frontend (BFF)

- Plugin discovery endpoints were added so the UI can learn what catalog plugins exist
- A stable pattern for feature gating UI based on what is installed

Docs

- Developer documentation exists for the plugin framework and MCP plugin example

## What is missing to hit the end goal

Product experience

- A consistent UI that can browse any plugin, not just a fixed set
- A consistent CLI story that mirrors the UI capabilities

Operations and management

- A management plane for catalog sources that is safe and usable
- A way to validate configuration changes before they cause runtime failures
- Ingestion status, error reporting, and debugging affordances
- Clear RBAC and multi-tenant behavior for viewing versus managing catalogs

Developer experience

- A documented pattern for plugin specific UI extensions where generic UI is insufficient
- A documented pattern for plugin specific CLI extensions where generic CLI is insufficient
- Explicit guidance for when to extend generated code versus customize templates or hooks
