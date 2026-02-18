# Configuration Persistence (ConfigStore)

## Overview

The catalog server uses a **pluggable configuration persistence system** to store and version catalog source configuration. The `ConfigStore` interface abstracts the storage backend so the same management API works whether the server runs as a local process with file-backed storage or inside Kubernetes with ConfigMap-backed storage.

**Location:** `pkg/catalog/plugin/`

All implementations provide optimistic concurrency control via content hashing, revision history for rollback, and are safe for concurrent use by multiple goroutines.

## ConfigStore Interface

```go
// pkg/catalog/plugin/configstore.go
var ErrVersionConflict = errors.New("config version conflict: file was modified since last load")

type ConfigRevision struct {
    Version   string    `json:"version"`   // Content hash at snapshot time
    Timestamp time.Time `json:"timestamp"` // When the snapshot was created
    Size      int64     `json:"size"`      // Byte size of the config at that revision
}

type ConfigChangeEvent struct {
    Version string // New content hash after the change
    Error   error  // Set if the watcher encountered an error
}

type ConfigStore interface {
    // Load reads the current config and returns it with a version hash.
    Load(ctx context.Context) (*CatalogSourcesConfig, string, error)

    // Save writes config to storage. version must match the stored version
    // or ErrVersionConflict is returned. Returns the new version hash.
    Save(ctx context.Context, cfg *CatalogSourcesConfig, version string) (string, error)

    // Watch returns a channel of external change events. The channel is
    // closed when the context is cancelled. Implementations that do not
    // support watching may return (nil, nil).
    Watch(ctx context.Context) (<-chan ConfigChangeEvent, error)

    // ListRevisions returns revision history, sorted newest first.
    ListRevisions(ctx context.Context) ([]ConfigRevision, error)

    // Rollback restores config to a previous revision identified by its
    // version hash. Internally re-saves via Save() for concurrency safety.
    Rollback(ctx context.Context, version string) (*CatalogSourcesConfig, string, error)
}
```

| Method | Returns | Description |
|--------|---------|-------------|
| `Load` | config, version, error | Read current config; version is SHA-256 hex digest |
| `Save` | newVersion, error | Write config atomically; rejects stale version with `ErrVersionConflict` |
| `Watch` | channel, error | Stream external changes (nil channel if unsupported) |
| `ListRevisions` | revisions, error | Historical snapshots, newest first |
| `Rollback` | config, newVersion, error | Restore a previous revision by re-saving it |

## Optimistic Concurrency

Both implementations use **content-hash-based optimistic concurrency** to prevent lost updates when multiple callers (management API, reconcile loop, external editor) modify the config concurrently.

```
Caller A                     Storage                     Caller B
   │                            │                            │
   │──── Load() ───────────────>│                            │
   │<─── cfg, version="abc..." ─│                            │
   │                            │<──── Load() ───────────────│
   │                            │───── cfg, version="abc..." │
   │                            │                            │
   │   (modify cfg)             │              (modify cfg)  │
   │                            │                            │
   │──── Save(cfg, "abc...") ──>│                            │
   │<─── newVersion="def..."  ──│                            │
   │                            │                            │
   │                            │<── Save(cfg, "abc...") ────│
   │                            │─── ErrVersionConflict ────>│
   │                            │                            │
   │                            │   (Caller B must re-Load   │
   │                            │    and retry)              │
```

The flow:

1. `Load()` returns the config and a version string (SHA-256 hex digest of raw content)
2. The caller modifies the config in memory
3. `Save()` re-reads the current stored content, computes its hash, and compares with the provided version
4. If the hashes match, the write proceeds and a new version hash is returned
5. If the hashes differ, `ErrVersionConflict` is returned (maps to HTTP 409)

## FileConfigStore

`FileConfigStore` persists configuration as a YAML file on the local filesystem.

**Location:** `pkg/catalog/plugin/file_config_store.go`

### Construction

```go
store, err := plugin.NewFileConfigStore("/config/sources.yaml")
```

The constructor validates that the path contains no `..` traversal components (`ErrPathTraversal`).

### Atomic Writes

Saves use a write-to-temp-then-rename strategy to prevent partial writes on crash:

```
1. Create temp file in same directory   (.sources-XXXXX.yaml.tmp)
2. Write marshaled YAML to temp file
3. fsync temp file
4. os.Rename(temp, target)              (atomic on POSIX)
5. Clean up temp file on any error path
```

### SHA-256 Content Hashing

The version string is the full SHA-256 hex digest of the raw file bytes:

```go
func hashBytes(data []byte) string {
    h := sha256.Sum256(data)
    return fmt.Sprintf("%x", h)
}
```

This means any byte-level change to the file (including whitespace or comment edits) produces a new version, which triggers the reconcile loop to re-initialize plugins.

### Revision History

Every `Save()` snapshots the current file content to a `.history/` directory before overwriting:

```
/config/
  sources.yaml                         # Current config
  .history/
    1706140800_a1b2c3d4.yaml          # {unix_timestamp}_{version_short}.yaml
    1706140500_e5f6a7b8.yaml
    1706140200_c9d0e1f2.yaml
    ...
```

| Property | Value |
|----------|-------|
| Filename format | `{unix_timestamp}_{version_short}.yaml` |
| Version short | First 8 characters of SHA-256 hex digest |
| Max revisions kept | 20 (`maxRevisionHistory`) |
| Pruning | Oldest entries removed after each save |
| Max config file size | 1 MiB (`maxConfigFileSize`) |

`ListRevisions()` scans the `.history/` directory, parses timestamps and version prefixes from filenames, and returns them sorted newest first.

### Rollback

`Rollback(version)` performs:

1. Scan `.history/` for a file whose version prefix matches the requested version
2. Read and parse the snapshot file
3. Read the current file to get its version hash
4. Call `Save()` with the restored config and the current version (normal concurrency check)

The version parameter is matched against the short (8-char) prefix stored in the filename. Path traversal is prevented because the version is matched against directory entries, never interpolated into a file path.

### Watch

`FileConfigStore` does not implement file watching. It returns `(nil, nil)` from `Watch()`. The server's reconcile loop polls `Load()` every 30 seconds to detect external changes.

## K8sSourceConfigStore

`K8sSourceConfigStore` persists configuration in a **Kubernetes ConfigMap**. The YAML config is stored as a data key; revision metadata and snapshots are stored in annotations.

**Location:** `pkg/catalog/plugin/k8s_config_store.go`

### Construction

```go
store := plugin.NewK8sSourceConfigStore(
    clientset,           // kubernetes.Interface
    "model-registry",    // namespace
    "catalog-sources",   // ConfigMap name
    "sources.yaml",      // data key within the ConfigMap
)
```

### ConfigMap Layout

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: catalog-sources
  namespace: model-registry
  annotations:
    catalog.kubeflow.org/revisions: '[{"version":"a1b2c3d4","timestamp":"...","size":1234}]'
    catalog.kubeflow.org/rev-a1b2c3d4: |
      # snapshot of previous config YAML
      apiVersion: catalog/v1alpha1
      ...
data:
  sources.yaml: |
    apiVersion: catalog/v1alpha1
    kind: CatalogSources
    catalogs:
      ...
```

| Annotation | Purpose |
|------------|---------|
| `catalog.kubeflow.org/revisions` | JSON array of `ConfigRevision` objects (metadata index) |
| `catalog.kubeflow.org/rev-{hash}` | Full YAML snapshot for each revision (data key = short version hash) |

### Revision Tracking

Revisions are stored entirely within ConfigMap annotations (no external storage needed):

- Before each `Save()`, the current data value is snapshotted into a `catalog.kubeflow.org/rev-{hash}` annotation
- The revision metadata index (`catalog.kubeflow.org/revisions`) is updated with the new entry
- Old revisions are pruned to keep at most 10 (`maxK8sRevisionHistory`)
- Max data size per ConfigMap is capped at 900 KiB (leaving headroom within the 1 MiB etcd value limit)

### Conflict Handling

Two layers of conflict detection protect against concurrent writes:

1. **Content hash comparison** -- same as `FileConfigStore`, the SHA-256 of the data value must match the caller's version
2. **Kubernetes resource version** -- if the ConfigMap was modified between GET and UPDATE, the API server returns HTTP 409, which is mapped to `ErrVersionConflict`

### RetryOnConflict

For callers that want automatic retry with exponential backoff:

```go
newVersion, err := store.RetryOnConflict(ctx, func(cfg *CatalogSourcesConfig) error {
    // Mutate cfg in place
    cfg.Catalogs["mcp"].Sources = append(cfg.Catalogs["mcp"].Sources, newSource)
    return nil
}, 3) // max 3 retries
```

The retry loop:

1. `Load()` the current config and version
2. Call the mutate function to apply changes in-place
3. `Save()` with the loaded version
4. On `ErrVersionConflict`, back off (`50ms * 2^attempt`) and retry from step 1
5. After `maxRetries` exhausted, return the conflict error

### RBAC Requirements

The catalog server's ServiceAccount needs the following permissions on the target ConfigMap:

```yaml
# deploy/catalog-server/rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: catalog-server
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "update"]
    resourceNames: ["catalog-sources"]
```

Without these permissions, `Load()` and `Save()` will return authorization errors at runtime.

### Watch

`K8sSourceConfigStore` does not currently implement watching. It returns `(nil, nil)` from `Watch()`. A future implementation could use Kubernetes informers to watch for ConfigMap changes. The server's reconcile loop polls `Load()` periodically instead.

## Store Mode Selection

The config store backend is selected at server startup via the `CATALOG_CONFIG_STORE_MODE` environment variable (or the `-config-store` flag):

```
cmd/catalog-server/main.go

    ┌───────────────────────────────────────────────┐
    │  CATALOG_CONFIG_STORE_MODE                    │
    │  (or -config-store flag)                      │
    └───────────────────┬───────────────────────────┘
                        │
          ┌─────────────┼──────────────┐
          ▼             ▼              ▼
      "file"         "k8s"          "none"
     (default)                    (no persistence)
          │             │              │
          ▼             ▼              ▼
    FileConfigStore  K8sSource     No store;
    backed by        ConfigStore   mutations are
    sources.yaml     backed by     not persisted
    on disk          ConfigMap
```

| Mode | Default | Backend | Additional Env Vars |
|------|---------|---------|---------------------|
| `file` | Yes | `FileConfigStore` using the `-sources` path | None |
| `k8s` | No | `K8sSourceConfigStore` with in-cluster client | `CATALOG_CONFIG_NAMESPACE` (default: `default`), `CATALOG_CONFIG_CONFIGMAP_NAME` (default: `catalog-sources`), `CATALOG_CONFIG_CONFIGMAP_KEY` (default: `sources.yaml`) |
| `none` | No | No store wired; management API mutations are lost on restart | None |

The `k8s` mode also wires a `K8sSecretResolver` so that source properties containing `SecretRef` objects are resolved from Kubernetes Secrets at runtime.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/configstore.go` | `ConfigStore` interface, `ConfigRevision`, `ConfigChangeEvent`, `ErrVersionConflict` |
| `pkg/catalog/plugin/file_config_store.go` | File-backed store with atomic writes, SHA-256 versioning, `.history/` snapshots |
| `pkg/catalog/plugin/k8s_config_store.go` | Kubernetes ConfigMap-backed store with annotation revisions, `RetryOnConflict` |
| `cmd/catalog-server/main.go` | Store mode selection (`file` / `k8s` / `none`) at server startup |
| `deploy/catalog-server/rbac.yaml` | Kubernetes RBAC granting ConfigMap access to the catalog-server ServiceAccount |

---

[Back to Source Management](./README.md) | [Next: Validation Pipeline](./validation-pipeline.md)
