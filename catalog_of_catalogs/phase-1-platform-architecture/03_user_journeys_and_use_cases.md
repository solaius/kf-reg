# User journeys and use cases

## Personas
- Platform admin
  - Configures sources.yaml, sets up providers, ensures catalog health
- AI engineer
  - Browses assets to assemble an AI workflow, agent, or application
- Data scientist or ML engineer
  - Discovers models and related assets, compares metadata
- Governance and operations
  - Wants visibility and traceability, even if not controlling lifecycle here
- Tooling developer
  - Adds a new asset-type plugin or provider

## Journey A: Browse and select assets
1. User opens catalog UI or uses CLI
2. Chooses an asset type (models, MCP servers, prompts, etc.)
3. Searches or filters using consistent query semantics
4. Opens an asset detail view and copies a stable reference for downstream usage

Success signals
- The user can find relevant assets quickly across sources
- Filter behavior feels consistent across asset types

## Journey B: Add a new asset type
1. Developer runs catalog-gen (or uses templates) to scaffold a plugin
2. Defines schema and properties
3. Implements at least one provider
4. Registers plugin with catalog-server
5. Adds a plugin section to sources.yaml
6. Runs migrations and starts catalog-server
7. The UI and CLI can discover the new plugin and interact with its endpoints

Success signals
- Most work is in schema and provider logic
- Little or no work is needed to reimplement filtering, pagination, OpenAPI wiring, or DB plumbing

## Journey C: Integrate with registries
1. A registry exists as system of record for an asset type
2. A plugin provider can ingest from registry APIs and keep catalog in sync
3. The catalog remains read-only but provides discovery and cross-asset linking

Success signals
- A registry-backed provider can be implemented without changing the server core
- Source configuration supports registry endpoints and auth configuration patterns

