# Concepts and terminology

## Key terms
- Catalog: A read-only discovery service that aggregates metadata about assets from one or more sources
- Registry: A system of record for assets where lifecycle and governance metadata is authored and managed
- Asset type: A category of AI asset (model, dataset, prompt template, agent, evaluation benchmark, MCP server, etc.)
- Plugin: A self-contained catalog implementation for a single asset type, loaded by catalog-server
- Source: A configured upstream location where assets are discovered and ingested (YAML file, HTTP endpoint, registry, git repo, etc.)
- Provider: Code that knows how to read a specific source type and produce entities and artifacts for ingestion
- Entity: The primary object listed by a plugin (example: Model, McpServer, Dataset, PromptTemplate)
- Artifact: A secondary object associated to an entity (example: model files, metrics, attachments)
- Reference: A typed link from one asset to another, potentially across plugins

## Mental model
The catalog-server is the host. Each plugin is a catalog for one asset type. Plugins are configured via a shared sources.yaml. Plugins ingest assets from sources, persist them to a shared DB, and expose a REST API for list and get operations.

A unified OpenAPI spec exists for documentation and client generation.

## Cross-asset references
We want an extensible way to link assets without hardcoding specific relationships into the server.

Minimum viable requirements
- Every asset should have a stable reference string that can be stored in metadata and exchanged between systems
- The reference format should be extensible and carry enough information to resolve back to a concrete API
- References should work across plugins and across sources

Suggested reference shape (example, not a hard requirement)
- scheme: catalog
- components: plugin, source, name, optional version
- example: catalog://mcp/source/internal-servers/mcpservers/my-server@v1

The important point is that references must be stable and resolvable, not that the string format is identical to this example.

