# Catalog Service Architecture

## Overview

The Catalog Service is a **federated metadata aggregation layer** that provides read-only discovery across multiple catalog sources.

**Location:** `catalog/`

## Core Components

### APIProvider Interface

```go
// catalog/internal/catalog/catalog.go
type APIProvider interface {
    // GetModel retrieves a single model by name
    GetModel(ctx context.Context, name string) (*openapi.CatalogModel, error)

    // ListModels searches models with filtering and pagination
    ListModels(ctx context.Context, params ListModelsParams) (*openapi.CatalogModelList, error)

    // GetArtifacts retrieves artifacts for a model
    GetArtifacts(ctx context.Context, name string, params ListArtifactsParams) (*openapi.CatalogArtifactList, error)

    // GetPerformanceArtifacts retrieves performance metrics
    GetPerformanceArtifacts(ctx context.Context, name string) (*openapi.CatalogMetricsArtifactList, error)

    // GetFilterOptions returns available filter fields
    GetFilterOptions(ctx context.Context) (*openapi.FilterOptions, error)
}
```

### ListModelsParams

```go
type ListModelsParams struct {
    PageSize        *int32
    NextPageToken   *string
    OrderBy         *string
    SortOrder       *string
    Query           *string      // Free-form search
    FilterQuery     *string      // Advanced filter DSL
    SourceID        []string     // Filter by source
    SourceLabel     []string     // Filter by label
}
```

## Source Provider Pattern

### ModelProviderFunc

```go
type ModelProviderFunc func(source *Source) ([]CatalogModel, error)
```

### Provider Registration

```go
// catalog/internal/catalog/loader.go
func (l *Loader) RegisterModelProvider(sourceType string, provider ModelProviderFunc) {
    l.providers[sourceType] = provider
}

// Registration in init
loader.RegisterModelProvider("yaml", yamlModelProvider)
loader.RegisterModelProvider("hf", hfModelProvider)
```

## Loader System

### Loader Structure

```go
type Loader struct {
    paths           []string
    providers       map[string]ModelProviderFunc
    eventHandlers   []LoaderEventHandler
    sourceCollection *SourceCollection
    mu              sync.RWMutex
}
```

### Configuration Loading

```go
func (l *Loader) Load() error {
    l.mu.Lock()
    defer l.mu.Unlock()

    // Load from all config paths
    for _, path := range l.paths {
        sources, err := l.loadSourcesFromPath(path)
        if err != nil {
            return err
        }

        for _, source := range sources {
            // Get provider for source type
            provider, ok := l.providers[source.Type]
            if !ok {
                continue
            }

            // Load models
            models, err := provider(&source)
            if err != nil {
                // Log error but continue
                continue
            }

            // Add to collection
            l.sourceCollection.AddSource(source, models)
        }
    }

    // Notify event handlers
    for _, handler := range l.eventHandlers {
        handler.OnLoad(l.sourceCollection)
    }

    return nil
}
```

### Hot-Reload Support

```go
func (l *Loader) StartWatcher(ctx context.Context) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    for _, path := range l.paths {
        watcher.Add(path)
    }

    go func() {
        for {
            select {
            case event := <-watcher.Events:
                if event.Op&fsnotify.Write == fsnotify.Write {
                    l.Load()
                }
            case <-ctx.Done():
                watcher.Close()
                return
            }
        }
    }()

    return nil
}
```

### Event Handlers

```go
type LoaderEventHandler interface {
    OnLoad(collection *SourceCollection)
}

// Performance metrics loader
type PerformanceMetricsLoader struct {
    metricsPath string
    db          *gorm.DB
}

func (p *PerformanceMetricsLoader) OnLoad(collection *SourceCollection) {
    // Load performance metrics after models are loaded
    p.loadMetrics()
}
```

## Source Collection

### Structure

```go
type SourceCollection struct {
    sources map[string]*Source
    models  map[string][]CatalogModel
    labels  map[string][]string
    mu      sync.RWMutex
}
```

### Priority-Based Merging

Sources loaded from multiple paths are merged with later sources taking priority:

```go
func (s *SourceCollection) AddSource(source Source, models []CatalogModel) {
    s.mu.Lock()
    defer s.mu.Unlock()

    existingSource, exists := s.sources[source.ID]
    if exists {
        // Merge with priority (new overwrites old)
        mergedSource := s.mergeSource(existingSource, &source)
        s.sources[source.ID] = mergedSource
    } else {
        s.sources[source.ID] = &source
    }

    s.models[source.ID] = models
}

func (s *SourceCollection) mergeSource(existing, new *Source) *Source {
    merged := *existing

    // Override non-nil fields from new
    if new.Name != "" {
        merged.Name = new.Name
    }
    if new.Enabled != nil {
        merged.Enabled = new.Enabled
    }
    if len(new.Labels) > 0 {
        merged.Labels = new.Labels
    }

    return &merged
}
```

## Database-Backed Catalog

### Implementation

```go
// catalog/internal/catalog/db_catalog.go
type dbCatalogImpl struct {
    db              *gorm.DB
    sourceRepo      *service.CatalogSourceRepository
    modelRepo       *service.CatalogModelRepository
    artifactRepo    *service.CatalogArtifactRepository
    metricsRepo     *service.CatalogMetricsArtifactRepository
}

func (c *dbCatalogImpl) ListModels(ctx context.Context, params ListModelsParams) (*openapi.CatalogModelList, error) {
    // Build query with filters
    query := c.db.Model(&models.CatalogModel{})

    // Apply source filter
    if len(params.SourceID) > 0 {
        query = query.Where("source_id IN ?", params.SourceID)
    }

    // Apply filter query
    if params.FilterQuery != nil && *params.FilterQuery != "" {
        query = c.applyFilterQuery(query, *params.FilterQuery)
    }

    // Apply free-form search
    if params.Query != nil && *params.Query != "" {
        query = c.applySearch(query, *params.Query)
    }

    // Apply pagination
    query = c.applyPagination(query, params)

    // Execute
    var results []models.CatalogModel
    if err := query.Find(&results).Error; err != nil {
        return nil, err
    }

    return c.mapToOpenAPI(results, params)
}
```

## Server Startup

```go
// catalog/cmd/catalog.go
func runCatalog(cmd *cobra.Command, args []string) error {
    // Create database connector
    connector, err := embedmd.NewEmbedMDService(&embedmd.EmbedMDConfig{
        DatabaseType: cfg.DatabaseType,
        DatabaseDSN:  cfg.DatabaseDSN,
    })

    // Build specification
    spec := buildCatalogSpec()

    // Connect
    repoSet, err := connector.Connect(spec)

    // Create loader
    loader := catalog.NewLoader(cfg.CatalogsPaths)
    loader.RegisterModelProvider("yaml", catalog.YAMLModelProvider)
    loader.RegisterModelProvider("hf", catalog.HFModelProvider)

    // Register event handlers
    loader.AddEventHandler(&catalog.PerformanceMetricsLoader{
        MetricsPath: cfg.PerformanceMetricsPath,
        DB:          repoSet.DB(),
    })

    loader.AddEventHandler(&catalog.PropertyOptionsRefresher{
        DB: repoSet.DB(),
    })

    // Load initial configuration
    loader.Load()

    // Start hot-reload watcher
    loader.StartWatcher(ctx)

    // Create API provider
    apiProvider := catalog.NewDBCatalog(repoSet)

    // Create and start server
    server := openapi.NewServer(apiProvider)
    return server.ListenAndServe(cfg.ListenAddress)
}
```

## Request Flow

```
HTTP Request (GET /models?source=hf&q=llama)
        │
        ▼
┌─────────────────────────────────────────────────┐
│           OpenAPI Controller                     │
│    api_model_catalog_service.go                 │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│           APIProvider.ListModels()               │
│    db_catalog.go                                │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│           Query Builder                          │
│    - Source filter                              │
│    - Search query                               │
│    - Advanced filters                           │
│    - Pagination                                 │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│           GORM Query                             │
│    CatalogModel table                           │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│           Response Mapping                       │
│    models.CatalogModel → openapi.CatalogModel  │
└─────────────────────────────────────────────────┘
```

---

[Back to Catalog Service Index](./README.md) | [Next: Source Providers](./source-providers.md)
