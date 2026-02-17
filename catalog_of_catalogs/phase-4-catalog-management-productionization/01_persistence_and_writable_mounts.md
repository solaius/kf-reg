# 01_persistence_and_writable_mounts

**Date**: 2026-02-16  
**Owner**: catalog-server plus BFF plus UI  
**Goal**: Make management edits persist correctly in both Docker dev and cluster deployments, without unsafe filesystem assumptions

## Problem statement

Today, the UI edit flow writes YAML back to a file path referenced by a source (example: yamlCatalogPath). In Docker compose, the data directory was mounted read-only (:ro) as defense-in-depth. For cluster deployments, ConfigMaps are always mounted read-only and cannot be written in-place.

This step adds a first-class persistence strategy that:
- Works for local development (filesystem-backed)
- Works in cluster (Kubernetes API-backed)
- Keeps the management-plane API stable for callers (UI and CLI)

## Requirements

### R1: Introduce a SourceConfigStore abstraction

Add an interface in catalog-server (or shared pkg) that represents "where management edits are stored":

- GetSourceConfig(plugin, sourceId) -> {rawYaml, origin, metadata}
- ValidateSourceConfig(plugin, sourceId, rawYaml) -> {valid, errors, warnings, normalized}
- ApplySourceConfig(plugin, sourceId, rawYaml, options) -> {appliedRevision, diffSummary}
- ListRevisions(plugin, sourceId) -> revisions[]
- Rollback(plugin, sourceId, revisionId) -> {appliedRevision}

Notes:
- rawYaml should be treated as the user-edited payload, not a regenerated canonical YAML unless explicitly requested
- origin must be preserved because relative path resolution depends on it

### R2: Provide two store implementations

#### FileSourceConfigStore (Docker dev and bare-metal)
- Reads and writes YAML to the configured file path (current behavior), but safely and atomically
- Supports a local revision history directory next to the file (example: .history/)
- Supports file locking to avoid concurrent write corruption

#### K8sSourceConfigStore (cluster mode)
- Stores YAML content in Kubernetes resources, updated via the Kubernetes API
- Must not attempt to write to ConfigMap volume mounts (ConfigMaps are mounted read-only)

**Design decision (confirmed in review):** Use a single ConfigMap per plugin (not per source) to store the authoritative `sources.yaml` (`CatalogSourcesConfig`). Edit that ConfigMap, and rehydrate plugin configs from it. Implementation: `pkg/catalog/plugin/k8s_config_store.go`.

Requirements:
- Catalog-server must reconcile the ConfigMap into an in-memory `CatalogSourcesConfig` view deterministically on startup and after every Save
- Revision history and snapshots are stored in ConfigMap annotations (key prefix `catalog.kubeflow.org/rev-`), up to 10 revisions
- Optimistic concurrency via content hash (SHA-256) comparison on Save
- ConfigMap data size is capped at 900 KiB to stay within the etcd 1 MiB value limit

### R3: Choose the store via configuration

Add configuration that selects the store mode:
- CATALOG_CONFIG_STORE_MODE=file|k8s
- For k8s mode:
  - CATALOG_CONFIG_NAMESPACE
  - CATALOG_CONFIG_CONFIGMAP_PREFIX (or naming convention)
  - Kubernetes in-cluster auth

### R4: Docker compose becomes writable where needed

Update docker-compose.catalog.yaml:
- Keep config mounts read-only when possible
- Make the specific data directory that holds the editable YAML writable

Example pattern:
- mount catalog/plugins/mcp/data read-write
- keep catalog/config read-only

### R5: Cluster RBAC for config writes

If using ConfigMaps:
- serviceaccount for catalog-server must have permissions:
  - get/list/watch/create/update/patch configmaps in the chosen namespace
- management endpoints must enforce existing RBAC middleware so only authorized roles can mutate config

## Implementation notes

### Atomic file write (file store)
- Write to file.tmp
- fsync
- Rename to target path (atomic on POSIX filesystems)
- Create a revision snapshot before overwrite

### Concurrency control (k8s store)
- Use resourceVersion to prevent lost updates
- The Apply API should include expectedResourceVersion (or ETag-like) returned by Get

### Security checks
- Disallow path traversal and restrict file store writes to an allow-listed root directory
- Enforce max file size (example: 1 MiB) to avoid accidental huge writes

### Sensitive value handling (confirmed in review)
- Sensitive values are referenced via `SecretRef` (Name, Namespace, Key), never inlined
- Sensitive values are redacted on Get via `RedactSensitiveProperties()` before returning to callers
- Sensitive key patterns: password, token, secret, apikey, api_key, credential (case-insensitive substring match)
- SecretRef type definition (in `pkg/catalog/plugin/management_types.go`):
  ```go
  type SecretRef struct {
      Name      string `json:"name" yaml:"name"`
      Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
      Key       string `json:"key" yaml:"key"`
  }
  ```
- The `SecurityWarningsLayer` produces warnings (not errors) when inline credential values are detected, guiding operators toward SecretRef without blocking saves
- Values stored as map objects (SecretRef-like) are not redacted; only plain string values matching sensitive patterns are redacted

## Acceptance criteria

- In Docker compose:
  - Edit MCP YAML in UI, Save, and the underlying YAML file on the host is updated
  - Restart containers and the updated YAML is still in effect
- In cluster mode:
  - Edit MCP YAML in UI, Save, and the corresponding Kubernetes resource is updated via API
  - Pod restarts do not lose the change
  - No writes are attempted against ConfigMap volume mounts
- Get and Apply responses include stable revision identifiers
- A minimal revision history exists and can be listed
- In cluster mode, a single ConfigMap stores all sources for a plugin (not per-source ConfigMaps)
- After server restart, the ConfigMap is reconciled into an in-memory config view
- Sensitive property values are redacted in Get/List responses
- Inline credential values produce validation warnings (not errors) suggesting SecretRef usage

## Definition of Done

- SourceConfigStore interface merged with unit tests
- File store and k8s store implementations merged with tests
- Docker compose updated and verified
- RBAC manifests (or docs) added for cluster mode
- All code follows PROGRAMMING_GUIDELINES.md patterns (errors, logging, testing, structure)

## References

- Kubernetes volumes docs: ConfigMaps are always mounted as readOnly  
  https://kubernetes.io/docs/concepts/storage/volumes/  
- Model Catalog docs (context for catalog service patterns)  
  https://www.kubeflow.org/docs/components/model-registry/reference/model-catalog-rest-api/
