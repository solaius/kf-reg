# Universal Asset Contract

## Overview

The universal asset contract defines a single, type-agnostic envelope --
`AssetResource` -- that every catalog plugin projects its native entities into.
Generic UI components, CLI commands, and cross-plugin search all consume
`AssetResource` instead of plugin-specific types, which means new plugins appear
in the interface without any frontend or CLI code changes.

The contract is **additive**: existing plugin-specific API responses remain
unchanged.  The asset projection is a parallel view that coexists with the
native endpoints.

```
                     Plugin-Specific API
                     (unchanged, full fidelity)
                          |
  +-----------+     +-----+------+     +-----------+
  |  Model    |     |  MCP       |     | Knowledge |
  |  Plugin   |     |  Plugin    |     |  Plugin   |
  +-----+-----+     +-----+-----+     +-----+-----+
        |                 |                 |
        +--------+--------+--------+--------+
                 |  AssetMapper    |
                 v                 v
          +------+------+  +------+------+
          |AssetResource|  |AssetResource|   ... per entity
          +------+------+  +------+------+
                 |
                 v
  +-------------------------------+
  | Generic UI / CLI / Search     |
  | (consumes AssetResource only) |
  +-------------------------------+
```

**Location:** `pkg/catalog/plugin/`

---

## AssetResource Envelope

### AssetResource

The top-level envelope wraps any catalog entity in a uniform shape inspired by
Kubernetes resource conventions:

```go
// pkg/catalog/plugin/asset_types.go
type AssetResource struct {
    APIVersion string         `json:"apiVersion"` // e.g. "catalog/v1alpha1"
    Kind       string         `json:"kind"`       // e.g. "McpServer", "CatalogModel"
    Metadata   AssetMetadata  `json:"metadata"`
    Spec       map[string]any `json:"spec"`
    Status     AssetStatus    `json:"status"`
}
```

| Field | Description |
|-------|-------------|
| `APIVersion` | Schema version, always `"catalog/v1alpha1"` today |
| `Kind` | Entity kind matching the plugin's `SupportedKinds()` |
| `Metadata` | Identity, ownership, labels, tags, source lineage |
| `Spec` | Plugin-specific fields flattened into a free-form map |
| `Status` | Lifecycle phase, health, conditions, cross-references |

### AssetMetadata

```go
type AssetMetadata struct {
    UID         string            `json:"uid"`
    Name        string            `json:"name"`
    DisplayName string            `json:"displayName,omitempty"`
    Description string            `json:"description,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    CreatedAt   string            `json:"createdAt,omitempty"`
    UpdatedAt   string            `json:"updatedAt,omitempty"`
    Owner       *AssetOwner       `json:"owner,omitempty"`
    SourceRef   *SourceRef        `json:"sourceRef,omitempty"`
}
```

### AssetOwner

```go
type AssetOwner struct {
    Name  string `json:"name,omitempty"`
    Email string `json:"email,omitempty"`
    Team  string `json:"team,omitempty"`
}
```

### SourceRef

Links an asset back to the catalog source it was ingested from, enabling
provenance tracking across refreshes:

```go
type SourceRef struct {
    SourceID   string `json:"sourceId"`
    SourceName string `json:"sourceName,omitempty"`
    SourceType string `json:"sourceType,omitempty"`
}
```

### AssetStatus

```go
type AssetStatus struct {
    Lifecycle  LifecycleStatus   `json:"lifecycle"`
    Health     HealthStatus      `json:"health"`
    Conditions []StatusCondition `json:"conditions,omitempty"`
    Links      *AssetLinks       `json:"links,omitempty"`
}
```

### LifecycleStatus Constants

| Constant | Value | Meaning |
|----------|-------|---------|
| `LifecycleActive` | `"active"` | Asset is available for consumption |
| `LifecycleDeprecated` | `"deprecated"` | Asset is still available but superseded |
| `LifecycleRetired` | `"retired"` | Asset is no longer available |
| `LifecycleDraft` | `"draft"` | Asset is not yet published |

```go
type LifecycleStatus string

const (
    LifecycleActive     LifecycleStatus = "active"
    LifecycleDeprecated LifecycleStatus = "deprecated"
    LifecycleRetired    LifecycleStatus = "retired"
    LifecycleDraft      LifecycleStatus = "draft"
)
```

### HealthStatus Constants

| Constant | Value | Meaning |
|----------|-------|---------|
| `HealthUnknown` | `"unknown"` | Health has not been assessed |
| `HealthHealthy` | `"healthy"` | Asset is operating normally |
| `HealthDegraded` | `"degraded"` | Asset is partially impaired |
| `HealthUnhealthy` | `"unhealthy"` | Asset is non-functional |

```go
type HealthStatus string

const (
    HealthUnknown   HealthStatus = "unknown"
    HealthHealthy   HealthStatus = "healthy"
    HealthDegraded  HealthStatus = "degraded"
    HealthUnhealthy HealthStatus = "unhealthy"
)
```

### StatusCondition

Follows the Kubernetes-style `type/status/reason` pattern:

```go
type StatusCondition struct {
    Type    string `json:"type"`
    Status  string `json:"status"` // "True", "False", "Unknown"
    Reason  string `json:"reason,omitempty"`
    Message string `json:"message,omitempty"`
}
```

### AssetLinks and LinkRef

```go
type AssetLinks struct {
    Related []LinkRef `json:"related,omitempty"`
}

type LinkRef struct {
    Kind string `json:"kind"`
    Name string `json:"name"`
    UID  string `json:"uid,omitempty"`
}
```

### AssetList

Paginated list following the repository's token-based pagination convention:

```go
type AssetList struct {
    APIVersion    string          `json:"apiVersion"`
    Kind          string          `json:"kind"` // always "AssetList"
    Items         []AssetResource `json:"items"`
    NextPageToken string          `json:"nextPageToken,omitempty"`
    TotalSize     int             `json:"totalSize,omitempty"`
}
```

---

## AssetMapper Interface

Each plugin implements `AssetMapper` to project its native entity types into the
universal envelope.  The mapper is the only code a plugin author writes to
participate in the generic UI and CLI.

```go
// pkg/catalog/plugin/asset_mapper.go
type AssetMapper interface {
    // MapToAsset converts a single plugin-specific entity into an AssetResource.
    MapToAsset(entity any) (AssetResource, error)

    // MapToAssets batch-converts a slice of entities.
    MapToAssets(entities []any) ([]AssetResource, error)

    // SupportedKinds returns the entity kinds this mapper handles.
    SupportedKinds() []string
}
```

A typical implementation follows this pattern:

```
Plugin Entity (e.g. McpServer)
       |
       |  type assertion  -->  extract standard fields into AssetMetadata
       |                   -->  flatten plugin-specific fields into Spec map
       |                   -->  map status fields into AssetStatus
       v
  AssetResource { Kind: "McpServer", Metadata: {...}, Spec: {...}, Status: {...} }
```

The mapper must handle every kind returned by `SupportedKinds()`.  If the entity
passed to `MapToAsset` is not a recognized type, the mapper returns an error.

---

## AssetMapperProvider, AssetLister, AssetGetter

These optional interfaces let plugins opt into increasingly rich integration
with the generic layer:

```go
// AssetMapperProvider -- expose the mapper to the framework
type AssetMapperProvider interface {
    GetAssetMapper() AssetMapper
}

// AssetLister -- support paginated listing via /api/assets
type AssetLister interface {
    ListAssets(ctx context.Context, opts AssetListOptions) (*AssetList, error)
}

// AssetGetter -- support single-entity retrieval by kind and name
type AssetGetter interface {
    GetAsset(ctx context.Context, kind string, name string) (*AssetResource, error)
}
```

### AssetListOptions

```go
type AssetListOptions struct {
    Kind        string `json:"kind,omitempty"`        // Filter to a specific entity kind
    PageSize    int    `json:"pageSize,omitempty"`     // Maximum items to return
    PageToken   string `json:"pageToken,omitempty"`    // Opaque pagination token
    FilterQuery string `json:"filterQuery,omitempty"`  // SQL-like filter expression
    OrderBy     string `json:"orderBy,omitempty"`      // Field to sort by
    SortOrder   string `json:"sortOrder,omitempty"`    // "ASC" or "DESC"
    SourceID    string `json:"sourceId,omitempty"`     // Filter to a specific source
}
```

### Interface Hierarchy

A plugin can implement any combination of these interfaces.  The framework
discovers them at startup and exposes the corresponding generic endpoints:

```
CatalogPlugin (required)
    |
    +-- AssetMapperProvider      -->  /api/assets (via framework adapter)
    |       |
    |       +-- AssetLister      -->  /api/assets (direct, higher fidelity)
    |       |
    |       +-- AssetGetter      -->  /api/assets/{kind}/{name}
    |
    +-- ActionProvider           -->  entity and source actions
    +-- CapabilitiesV2Provider   -->  V2 capabilities discovery
```

---

## MapToAssetsBatch Helper

`MapToAssetsBatch` is a utility function that eliminates the loop-and-collect
boilerplate every plugin mapper would otherwise duplicate:

```go
// pkg/catalog/plugin/asset_mapper.go
func MapToAssetsBatch(
    entities []any,
    mapFn func(any) (AssetResource, error),
) ([]AssetResource, error) {
    result := make([]AssetResource, 0, len(entities))
    for _, entity := range entities {
        asset, err := mapFn(entity)
        if err != nil {
            return nil, err
        }
        result = append(result, asset)
    }
    return result, nil
}
```

Plugin mappers use it to implement `MapToAssets` in one line:

```go
func (m *McpAssetMapper) MapToAssets(entities []any) ([]AssetResource, error) {
    return plugin.MapToAssetsBatch(entities, m.MapToAsset)
}
```

On error the batch aborts early and returns `nil` plus the first error.  This
fail-fast behavior is intentional -- partial results would confuse pagination
logic in the generic layer.

---

## DefaultAssetStatus

Convenience constructor that returns the sensible starting state for a freshly
ingested entity:

```go
func DefaultAssetStatus() AssetStatus {
    return AssetStatus{
        Lifecycle: LifecycleActive,  // active
        Health:    HealthUnknown,    // unknown
    }
}
```

Plugins call this from their mapper and then overlay any additional status
fields (conditions, links, health) specific to the entity being projected.

---

## Overlay Store

### Purpose

Source data is immutable across refreshes -- the plugin re-ingests entities from
upstream catalogs and replaces the in-memory state.  User-applied metadata
(tags, annotations, labels, lifecycle changes) must survive these refreshes.
The **OverlayStore** provides this persistence.

```
 User action                    Plugin refresh
 (tag, annotate, deprecate)     (re-ingest from source)
       |                              |
       v                              v
+-------------------+      +-----------------------+
| OverlayStore      |      | Plugin in-memory data |
| (catalog_overlays)|      | (replaced each cycle) |
+-------------------+      +-----------------------+
       |                              |
       +----------+-------------------+
                  |
                  v
           AssetResource (merged at response time)
```

### OverlayRecord

```go
// pkg/catalog/plugin/overlay_store.go
type OverlayRecord struct {
    PluginName string      `gorm:"primaryKey;column:plugin_name"`
    EntityKind string      `gorm:"primaryKey;column:entity_kind"`
    EntityUID  string      `gorm:"primaryKey;column:entity_uid"`
    Tags       StringSlice `gorm:"column:tags;type:text"`
    Annotations JSONMap    `gorm:"column:annotations;type:text"`
    Labels     JSONMap     `gorm:"column:labels;type:text"`
    Lifecycle  string      `gorm:"column:lifecycle_phase"`
    UpdatedAt  time.Time   `gorm:"column:updated_at;autoUpdateTime"`
}
```

The composite primary key `(plugin_name, entity_kind, entity_uid)` uniquely
identifies any entity across all plugins.

**Table:** `catalog_overlays`

### Custom GORM Types

Two custom GORM value types handle JSON serialization to the `text` columns:

| Type | Go Type | DB Representation |
|------|---------|-------------------|
| `StringSlice` | `[]string` | JSON array in TEXT column |
| `JSONMap` | `map[string]string` | JSON object in TEXT column |

Both implement `sql.Scanner` and `driver.Valuer` so GORM can round-trip them
transparently.

### OverlayStore Methods

```go
type OverlayStore struct {
    db *gorm.DB
}

func NewOverlayStore(db *gorm.DB) *OverlayStore
func (s *OverlayStore) AutoMigrate() error

func (s *OverlayStore) Upsert(record *OverlayRecord) error
func (s *OverlayStore) Get(pluginName, entityKind, entityUID string) (*OverlayRecord, error)
func (s *OverlayStore) Delete(pluginName, entityKind, entityUID string) error
func (s *OverlayStore) ListByPlugin(pluginName string) ([]OverlayRecord, error)
```

| Method | Description |
|--------|-------------|
| `AutoMigrate` | Creates or updates the `catalog_overlays` table via GORM auto-migration |
| `Upsert` | Creates or updates an overlay using `ON CONFLICT UPDATE ALL` |
| `Get` | Returns the overlay for a specific entity, or `(nil, nil)` if none exists |
| `Delete` | Removes an overlay by its composite key |
| `ListByPlugin` | Returns all overlays belonging to a plugin |

### Overlay Merge Flow

The action framework writes overlays; plugins read and merge them at response
time.  The merge happens when a plugin builds its `AssetResource` response:

```
1.  User invokes action (e.g. tag, annotate, deprecate)
        |
        v
2.  ActionHandler validates request, calls OverlayStore.Upsert()
        |
        v
3.  OverlayStore persists the change in catalog_overlays
        |
        v
4.  (later) Generic endpoint calls plugin.ListAssets() or plugin.GetAsset()
        |
        v
5.  Plugin mapper projects native entity --> AssetResource
        |
        v
6.  Plugin calls OverlayStore.Get() for the entity UID
        |
        v
7.  If overlay exists:
        - Merge overlay.Tags into Metadata.Tags
        - Merge overlay.Labels into Metadata.Labels
        - Merge overlay.Annotations into Metadata.Annotations
        - If overlay.Lifecycle is set, override Status.Lifecycle
        |
        v
8.  Return merged AssetResource to caller
```

This design means:

- **Source refreshes never lose user changes** -- overlays live in the database,
  not in the plugin's in-memory entity store.
- **Actions are plugin-agnostic** -- the action framework writes overlays using
  the same schema regardless of which plugin owns the entity.
- **Merge is lazy** -- overlays are applied only when the entity is read, not
  eagerly when the overlay is written.

---

## Key Files

| File | Location | Purpose |
|------|----------|---------|
| `asset_types.go` | `pkg/catalog/plugin/asset_types.go` | AssetResource, AssetMetadata, AssetStatus, lifecycle/health enums, AssetList |
| `asset_mapper.go` | `pkg/catalog/plugin/asset_mapper.go` | AssetMapper interface, AssetMapperProvider, AssetLister, AssetGetter, AssetListOptions, MapToAssetsBatch, DefaultAssetStatus |
| `overlay_store.go` | `pkg/catalog/plugin/overlay_store.go` | OverlayRecord, OverlayStore (Upsert, Get, Delete, ListByPlugin), StringSlice, JSONMap custom GORM types |

---

[Back to Universal Assets](./README.md) | [Prev: Capabilities Discovery](./capabilities-discovery.md) | [Next: Action Framework](./action-framework.md)
