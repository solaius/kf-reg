# Product requirements

This file focuses on product-level requirements and constraints. It intentionally avoids prescribing exact code structure, but it is specific about the user-visible contract and the invariants the implementation must preserve.

## Functional requirements

### FR1: Multi-plugin hosting (catalog-server as host runtime)
- catalog-server must host multiple plugins in one process
- Plugins must be enabled or disabled by configuration
- A plugin that is disabled must not register routes, run migrations, or start background watchers
- A plugin that fails to initialize should not prevent other plugins from serving when feasible
  - The server should surface failed plugin status via /api/plugins

#### Required capabilities
- Start with N plugins configured, 0 to N can be enabled
- List enabled and disabled plugins, including failure state
- Support a safe default where only the model plugin is enabled to preserve current deployments

### FR2: Plugin discovery and metadata
The server must expose a plugins discovery endpoint usable by generic UI and CLI clients.

#### Minimum plugin metadata
- pluginName
- apiBasePath
- apiVersion
- entityKinds (primary entity kind, optional artifact kinds)
- capabilities
  - listEntities
  - getEntity
  - listSources
  - hotReload (if supported)
  - artifacts (if supported)
- status
  - enabled
  - initialized
  - serving
  - lastError (string, optional)
  - lastHealthyTime (timestamp, optional)

A richer contract is fine if it remains additive.

### FR3: Plugin lifecycle
Each plugin must support a lifecycle that makes startup predictable and safe.

Required lifecycle phases
- Load configuration
- Init (per-plugin config, validate inputs)
- Migrations (schema setup)
- RegisterRoutes (mount HTTP API)
- Start (background watchers, scheduled refresh)
- Healthy (health and readiness integration)
- Stop (graceful shutdown)

Lifecycle requirements
- Migrations must be idempotent and run once per plugin per server start
- Plugins must not access other plugins' config sections directly
- Plugins must share the DB connection but own their tables

### FR4: Source configuration (sources.yaml)
A single sources.yaml config defines sources per plugin.

Required source fields
- id (unique within plugin)
- type (provider type, example yaml, http, registry)
- enabled (default true)
- properties (provider-specific key/value bag)

Optional source fields
- include and exclude patterns for file-based catalogs
- refreshInterval or watch settings where supported
- tags for grouping sources in UI

Requirements
- Plugins read only their own section
- Validation errors must point to the exact plugin and source id that failed
- The system should support multiple sources per plugin, concurrently ingested

### FR5: Source status and provenance
The system must expose enough status to debug ingestion and provide user confidence.

Required per-source status
- lastAttemptTime
- lastSuccessTime
- lastError
- entityCount and artifactCount (where supported)
- source fingerprint or revision (best effort)
  - file mtime and hash for yaml
  - etag or revision for http or registry

### FR6: Ingestion and persistence
- Providers ingest entities and artifacts from sources
- Ingestion persists to the shared DB
- API reads from DB, not directly from files
- Ingestion should support both:
  - on startup refresh
  - periodic refresh or file watch for file-backed sources

Ingestion requirements
- A failed source should not poison other sources
- Partial failures should be visible via per-source status
- A stable upsert model should exist so refresh does not duplicate entities

### FR7: Query experience consistency
All list endpoints across plugins must support consistent query semantics.

Required
- filterQuery
- pageSize and nextPageToken
- orderBy and sortOrder
- stable, deterministic pagination behavior

If a plugin cannot support a feature, it must declare that via capabilities and return a clear error.

### FR8: Common resource envelopes and metadata
All entities must provide a common baseline:
- name
- description
- timestamps
- custom properties metadata map
- provenance information

Use shared schemas to avoid per-plugin drift.

### FR9: Unified OpenAPI ownership and merge
- Each plugin owns its OpenAPI spec
- Shared schemas live in a single place
- A deterministic merge process creates a unified spec used for docs and CI validation

Requirements
- Schema collisions are avoided by namespacing or prefixing
- Operation ids are unique
- The merge process is deterministic and has a check mode in CI

### FR10: Backward compatibility invariants
- Model Catalog paths, schemas, and behaviors remain unchanged
- Existing configuration continues to work for the model plugin
- Any optional naming mapping for legacy config keys is explicitly tested

### FR11: Developer ergonomics for new asset types
A developer must be able to add a new catalog type with minimal boilerplate.

Minimum expectations
- A documented scaffold path
- Clear separation of generated and editable code
- Regeneration must not overwrite handwritten logic
- Generated code must remain in sync and enforced by CI

### FR12: Cross-asset linking, minimal viable
We need a way to express relationships between assets without hardcoding.

Minimum requirements
- Stable reference string emitted by every entity
- A shared schema field for references where appropriate
- Optional resolve endpoint or deterministic mapping to resolve references to URLs

This does not require a full graph model in v1.

## Non-functional requirements summary
See 09_observability_security_and_nonfunctional.md for details.

