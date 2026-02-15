# Catalog Database Models

This document covers the data models used by the Catalog Service.

## Overview

The Catalog Service uses a separate set of database models from the core Model Registry, optimized for read-heavy discovery operations.

## Core Models

### CatalogModel

```go
// catalog/internal/db/models/catalog_model.go
type CatalogModel struct {
    ID                       int32   `gorm:"primaryKey;autoIncrement"`
    SourceID                 string  `gorm:"column:source_id;not null"`
    Name                     string  `gorm:"column:name;not null"`
    Description              *string `gorm:"column:description"`
    Readme                   *string `gorm:"column:readme;type:text"`
    Maturity                 *string `gorm:"column:maturity"`
    Language                 *string `gorm:"column:language;type:json"`       // JSON array
    Tasks                    *string `gorm:"column:tasks;type:json"`          // JSON array
    LibraryName              *string `gorm:"column:library_name"`
    License                  *string `gorm:"column:license"`
    LicenseLink              *string `gorm:"column:license_link"`
    Provider                 *string `gorm:"column:provider"`
    CreateTimeSinceEpoch     int64   `gorm:"column:create_time_since_epoch"`
    LastUpdateTimeSinceEpoch int64   `gorm:"column:last_update_time_since_epoch"`
}
```

### CatalogArtifact

```go
type CatalogArtifact struct {
    ID             int32   `gorm:"primaryKey;autoIncrement"`
    CatalogModelID int32   `gorm:"column:catalog_model_id;not null"`
    URI            string  `gorm:"column:uri;not null"`
}
```

### CatalogModelArtifact

Join table for model-artifact relationships:

```go
type CatalogModelArtifact struct {
    CatalogModelID    int32 `gorm:"primaryKey"`
    CatalogArtifactID int32 `gorm:"primaryKey"`
}
```

### CatalogSource

```go
type CatalogSource struct {
    ID        string  `gorm:"primaryKey"`
    Name      string  `gorm:"column:name;not null"`
    Enabled   bool    `gorm:"column:enabled;default:true"`
    AssetType string  `gorm:"column:asset_type"`  // "model", "mcp"
}
```

### CatalogMetricsArtifact

Performance metrics storage:

```go
type CatalogMetricsArtifact struct {
    ID             int32   `gorm:"primaryKey;autoIncrement"`
    CatalogModelID int32   `gorm:"column:catalog_model_id;not null"`
    MetricName     string  `gorm:"column:metric_name"`
    MetricValue    float64 `gorm:"column:metric_value"`
    HardwareType   *string `gorm:"column:hardware_type"`
    HardwareCount  *int32  `gorm:"column:hardware_count"`
}
```

## Property Options

Materialized view for filter options:

```go
type PropertyOption struct {
    ID       int32  `gorm:"primaryKey;autoIncrement"`
    SourceID string `gorm:"column:source_id;not null"`
    Field    string `gorm:"column:field;not null"`
    Value    string `gorm:"column:value;not null"`
}
```

## Repository Interfaces

### CatalogModelRepository

```go
type CatalogModelRepository interface {
    Create(model *CatalogModel) (*CatalogModel, error)
    Update(model *CatalogModel) (*CatalogModel, error)
    GetByID(id int32) (*CatalogModel, error)
    GetByName(sourceID, name string) (*CatalogModel, error)
    List(opts CatalogModelListOptions) ([]CatalogModel, *Pagination, error)
    DeleteBySource(sourceID string) error
}
```

### CatalogSourceRepository

```go
type CatalogSourceRepository interface {
    Upsert(source *CatalogSource) (*CatalogSource, error)
    GetByID(id string) (*CatalogSource, error)
    List(opts CatalogSourceListOptions) ([]CatalogSource, error)
    Delete(id string) error
}
```

### CatalogMetricsArtifactRepository

```go
type CatalogMetricsArtifactRepository interface {
    Create(metric *CatalogMetricsArtifact) (*CatalogMetricsArtifact, error)
    GetByModelID(modelID int32) ([]CatalogMetricsArtifact, error)
    DeleteBySource(sourceID string) error
}
```

## Database Schema

### Tables

```sql
CREATE TABLE catalog_source (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    asset_type VARCHAR(50)
);

CREATE TABLE catalog_model (
    id INT PRIMARY KEY AUTO_INCREMENT,
    source_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    readme MEDIUMTEXT,
    maturity VARCHAR(50),
    language JSON,
    tasks JSON,
    library_name VARCHAR(255),
    license VARCHAR(100),
    license_link TEXT,
    provider VARCHAR(255),
    create_time_since_epoch BIGINT,
    last_update_time_since_epoch BIGINT,
    FOREIGN KEY (source_id) REFERENCES catalog_source(id),
    UNIQUE KEY (source_id, name)
);

CREATE TABLE catalog_artifact (
    id INT PRIMARY KEY AUTO_INCREMENT,
    catalog_model_id INT NOT NULL,
    uri TEXT NOT NULL,
    FOREIGN KEY (catalog_model_id) REFERENCES catalog_model(id)
);

CREATE TABLE catalog_metrics_artifact (
    id INT PRIMARY KEY AUTO_INCREMENT,
    catalog_model_id INT NOT NULL,
    metric_name VARCHAR(100),
    metric_value DOUBLE,
    hardware_type VARCHAR(50),
    hardware_count INT,
    FOREIGN KEY (catalog_model_id) REFERENCES catalog_model(id)
);

CREATE TABLE property_option (
    id INT PRIMARY KEY AUTO_INCREMENT,
    source_id VARCHAR(255) NOT NULL,
    field VARCHAR(255) NOT NULL,
    value VARCHAR(255) NOT NULL,
    UNIQUE KEY (source_id, field, value)
);
```

### Indexes

```sql
CREATE INDEX idx_catalog_model_source ON catalog_model(source_id);
CREATE INDEX idx_catalog_model_name ON catalog_model(name);
CREATE INDEX idx_catalog_model_maturity ON catalog_model(maturity);
CREATE INDEX idx_catalog_model_provider ON catalog_model(provider);
CREATE INDEX idx_catalog_metrics_model ON catalog_metrics_artifact(catalog_model_id);
CREATE INDEX idx_property_option_source ON property_option(source_id);
```

## Custom Properties

Custom properties are stored in a separate property table:

```go
type CatalogModelProperty struct {
    CatalogModelID   int32   `gorm:"primaryKey"`
    Name             string  `gorm:"primaryKey"`
    IsCustomProperty bool    `gorm:"column:is_custom_property"`
    IntValue         *int32
    DoubleValue      *float64
    StringValue      *string
    BoolValue        *bool
}
```

### Property Access

```go
func (r *CatalogModelRepositoryImpl) GetWithProperties(id int32) (*CatalogModel, map[string]any, error) {
    var model CatalogModel
    if err := r.db.First(&model, id).Error; err != nil {
        return nil, nil, err
    }

    var props []CatalogModelProperty
    r.db.Where("catalog_model_id = ?", id).Find(&props)

    propMap := make(map[string]any)
    for _, p := range props {
        propMap[p.Name] = p.getValue()
    }

    return &model, propMap, nil
}
```

## Caching Strategy

### Source-Level Caching

Models are cached per source after loading:

```go
type SourceCache struct {
    models    map[string][]CatalogModel
    loadedAt  time.Time
    mu        sync.RWMutex
}

func (c *SourceCache) Get(sourceID string) ([]CatalogModel, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    models, ok := c.models[sourceID]
    return models, ok
}
```

### Property Options Cache

Property options are refreshed on source reload:

```go
type PropertyOptionsCache struct {
    options   map[string]map[string][]string  // sourceID -> field -> values
    refresher *PropertyOptionsRefresher
}

func (c *PropertyOptionsCache) Refresh(sourceID string, models []CatalogModel) {
    opts := make(map[string][]string)

    for _, model := range models {
        // Collect unique values
        opts["maturity"] = appendUnique(opts["maturity"], model.Maturity)
        opts["provider"] = appendUnique(opts["provider"], model.Provider)
        // ...
    }

    c.options[sourceID] = opts
}
```

---

[Back to Catalog Service Index](./README.md) | [Previous: Filtering System](./filtering-system.md) | [Next: Performance Metrics](./performance-metrics.md)
