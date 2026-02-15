# Step-by-Step Creation Guide

This document describes how the MCP Catalog feature was implemented, serving as a guide for implementing similar features.

## Overview

The MCP Catalog was implemented in three main phases:

1. **Phase 1:** UI Gallery View (frontend-first approach)
2. **Phase 2:** Database-Backed YAML Source (backend persistence)
3. **Phase 3:** Source Filtering & Merge (advanced configuration)

## Phase 1: UI Gallery View

### Step 1.1: Define OpenAPI Schema

Start by defining the data models in the OpenAPI specification:

```yaml
# api/openapi/src/catalog.yaml

# 1. Add MCP Server schema
McpServer:
  type: object
  required:
    - name
  properties:
    id:
      type: string
    name:
      type: string
    description:
      type: string
    provider:
      type: string
    # ... other fields

# 2. Add MCP Tool schema
McpTool:
  type: object
  required:
    - name
    - accessType
  properties:
    name:
      type: string
    description:
      type: string
    accessType:
      $ref: '#/components/schemas/McpToolAccessType'

# 3. Add enums
McpTransportType:
  type: string
  enum: [stdio, http, sse]

McpToolAccessType:
  type: string
  enum: [read_only, read_write, execute]
```

### Step 1.2: Generate OpenAPI Code

```bash
# Run code generation
make generate-openapi

# This creates:
# - catalog/pkg/openapi/model_mcp_server.go
# - catalog/pkg/openapi/model_mcp_tool.go
# - etc.
```

### Step 1.3: Create React Components

Create the frontend components in order of dependency:

```
1. Types & API Client
   clients/ui/frontend/src/app/api/types/mcpCatalog.ts
   clients/ui/frontend/src/app/api/mcpCatalogService.ts

2. Context Provider
   clients/ui/frontend/src/app/context/McpCatalogContext.tsx

3. Basic Components
   components/McpCatalogCard.tsx
   components/McpCatalogCardBody.tsx
   components/McpSecurityIndicators.tsx
   components/McpToolsList.tsx

4. Filter Components
   components/McpCatalogFilters.tsx
   components/McpCatalogStringFilter.tsx

5. Page Screens
   screens/McpCatalogGalleryView.tsx
   screens/McpServerDetailsView.tsx
   screens/McpCatalog.tsx

6. Routes
   McpCatalogRoutes.tsx
```

### Step 1.4: Add BFF Handlers

Create the BFF layer to proxy requests to the catalog service:

```go
// clients/ui/bff/internal/api/mcp_server_handler.go

// 1. Define handler struct and envelope types
type McpServerListEnvelope Envelope[*models.McpServerList, None]

// 2. Implement handlers
func (app *App) GetAllMcpServersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
        return
    }

    mcpServers, err := app.repositories.ModelCatalogClient.GetAllMcpServers(client, r.URL.Query())
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    app.WriteJSON(w, http.StatusOK, McpServerListEnvelope{Data: mcpServers}, nil)
}

// 3. Register routes
router.GET("/api/v1/mcp_catalog/mcp_servers", app.GetAllMcpServersHandler)
```

## Phase 2: Database-Backed YAML Source

### Step 2.1: Create Database Entity

Define the MCP server entity following existing patterns:

```go
// catalog/internal/db/models/mcp_server.go

type McpServerListOptions struct {
    models.Pagination
    Name        *string
    SourceIDs   *[]string
    TextSearch  *string
    FilterQuery *string
}

type McpServerAttributes struct {
    Name                     *string
    ExternalID               *string
    CreateTimeSinceEpoch     *int64
    LastUpdateTimeSinceEpoch *int64
}

type McpServer interface {
    models.Entity[McpServerAttributes]
}

type McpServerImpl = models.BaseEntity[McpServerAttributes]

type McpServerRepository interface {
    GetByID(id int32) (McpServer, error)
    GetByName(name string) (McpServer, error)
    List(opts McpServerListOptions) (*models.ListWrapper[McpServer], error)
    Save(server McpServer) (McpServer, error)
    DeleteBySource(sourceID string) error
}
```

### Step 2.2: Implement Repository

```go
// catalog/internal/db/service/mcp_server_repo.go

type mcpServerRepository struct {
    db *gorm.DB
}

func (r *mcpServerRepository) List(opts McpServerListOptions) (*models.ListWrapper[McpServer], error) {
    query := r.db.Model(&schema.McpServer{})

    // Apply text search
    if opts.TextSearch != nil && *opts.TextSearch != "" {
        term := "%" + *opts.TextSearch + "%"
        query = query.Where("name LIKE ? OR description LIKE ?", term, term)
    }

    // Apply filter query
    if opts.FilterQuery != nil && *opts.FilterQuery != "" {
        query = filter.ApplyFilterQuery(query, *opts.FilterQuery, filter.RestEntityMcpServer)
    }

    // Execute with pagination
    var results []schema.McpServer
    query.Find(&results)

    return &models.ListWrapper[McpServer]{Items: convertResults(results)}, nil
}
```

### Step 2.3: Create YAML Provider

```go
// catalog/internal/mcp/yaml_mcp_catalog.go

// 1. Define YAML structures
type yamlMcpServer struct {
    Name        string        `yaml:"name"`
    Description string        `yaml:"description"`
    Tools       []yamlMcpTool `yaml:"tools"`
    // ... other fields
}

type yamlMcpCatalog struct {
    Source     string          `yaml:"source"`
    McpServers []yamlMcpServer `yaml:"mcp_servers"`
}

// 2. Implement provider function
func NewYamlMcpProvider(ctx context.Context, source *McpSource, reldir string) (<-chan McpServerProviderRecord, error) {
    // Read YAML file
    path := source.Properties["yamlCatalogPath"].(string)
    data, _ := os.ReadFile(path)

    var catalog yamlMcpCatalog
    yaml.Unmarshal(data, &catalog)

    // Emit records
    ch := make(chan McpServerProviderRecord)
    go func() {
        defer close(ch)
        for _, server := range catalog.McpServers {
            ch <- server.ToMcpServerProviderRecord()
        }
    }()

    return ch, nil
}

// 3. Register provider
var RegisteredMcpProviders = map[string]McpServerProviderFunc{
    "yaml": NewYamlMcpProvider,
}
```

### Step 2.4: Create MCP Loader

```go
// catalog/internal/catalog/mcp_loader.go

type McpLoader struct {
    paths    []string
    services service.Services
}

func (l *McpLoader) Start(ctx context.Context) error {
    // 1. Read all source configs
    allSources, _ := l.readAndMergeSources()

    // 2. Load servers from each source
    for _, source := range allSources {
        provider := RegisteredMcpProviders[source.Type]
        records, _ := provider(ctx, &source, "")

        for record := range records {
            l.services.McpServerRepository.Save(record.Server)
        }
    }

    // 3. Watch for file changes
    for _, path := range l.paths {
        go l.watchPath(ctx, path)
    }

    return nil
}
```

### Step 2.5: Implement DB Catalog Provider

```go
// catalog/internal/mcp/db_mcp_catalog.go

type DbMcpCatalogProvider struct {
    repository McpServerRepository
}

func (p *DbMcpCatalogProvider) ListMcpServers(ctx context.Context, name, q, filterQuery, namedQuery string) ([]model.McpServer, error) {
    // Build list options
    opts := dbmodels.McpServerListOptions{}
    if q != "" {
        opts.TextSearch = &q
    }
    if filterQuery != "" {
        opts.FilterQuery = &filterQuery
    }

    // Query database
    result, _ := p.repository.List(opts)

    // Convert to API model
    servers := make([]model.McpServer, len(result.Items))
    for i, item := range result.Items {
        servers[i] = convertDbToApiMcpServer(item)
    }

    return servers, nil
}
```

## Phase 3: Source Filtering & Merge

### Step 3.1: Implement Source Merge

```go
// catalog/internal/catalog/mcp_source_merge.go

func MergeMcpSourcesFromPaths(paths []string, readFunc func(string) ([]McpSource, error)) (map[string]McpSource, error) {
    merged := make(map[string]McpSource)

    for _, path := range paths {
        sources, _ := readFunc(path)
        for _, source := range sources {
            source.Origin = path
            if existing, ok := merged[source.Id]; ok {
                merged[source.Id] = MergeMcpSource(existing, source)
            } else {
                merged[source.Id] = source
            }
        }
    }

    return merged, nil
}

func MergeMcpSource(base, override McpSource) McpSource {
    result := base

    if override.Name != "" {
        result.Name = override.Name
    }
    if override.Enabled != nil {
        result.Enabled = override.Enabled
    }
    if len(override.IncludedServers) > 0 {
        result.IncludedServers = override.IncludedServers
    }
    // ... merge other fields

    return result
}
```

### Step 3.2: Implement Server Filtering

```go
// catalog/internal/catalog/mcp_server_filter.go

type McpServerFilter struct {
    includePatterns []*regexp.Regexp
    excludePatterns []*regexp.Regexp
}

func NewMcpServerFilterFromSource(source *McpSource) (*McpServerFilter, error) {
    filter := &McpServerFilter{}

    for _, pattern := range source.IncludedServers {
        re, _ := patternToRegexp(pattern)
        filter.includePatterns = append(filter.includePatterns, re)
    }

    for _, pattern := range source.ExcludedServers {
        re, _ := patternToRegexp(pattern)
        filter.excludePatterns = append(filter.excludePatterns, re)
    }

    return filter, nil
}

func (f *McpServerFilter) Allows(serverName string) bool {
    // Check exclusions first
    for _, pattern := range f.excludePatterns {
        if pattern.MatchString(serverName) {
            return false
        }
    }

    // If no include patterns, allow all
    if len(f.includePatterns) == 0 {
        return true
    }

    // Must match at least one include pattern
    for _, pattern := range f.includePatterns {
        if pattern.MatchString(serverName) {
            return true
        }
    }

    return false
}
```

### Step 3.3: Add Named Queries

```go
// In McpLoader
func (l *McpLoader) readAndMergeNamedQueries() {
    l.namedQueries = make(map[string]map[string]FieldFilter)

    for _, path := range l.paths {
        config, _ := l.readConfig(path)
        for name, filters := range config.NamedQueries {
            if l.namedQueries[name] == nil {
                l.namedQueries[name] = make(map[string]FieldFilter)
            }
            for field, filter := range filters {
                l.namedQueries[name][field] = filter
            }
        }
    }
}

// In DbMcpCatalogProvider
func (p *DbMcpCatalogProvider) ListMcpServers(..., namedQuery string) {
    if namedQuery != "" {
        namedQueries := p.namedQueryResolver()
        if filters, ok := namedQueries[namedQuery]; ok {
            filterQuery = convertNamedQueryToFilterQuery(filters)
        }
    }
    // ... rest of query
}
```

## Testing Approach

### Unit Tests

```go
// Test YAML provider
func TestYamlMcpProvider(t *testing.T) {
    source := &McpSource{
        Properties: map[string]any{
            "yamlCatalogPath": "testdata/servers.yaml",
        },
    }

    records, err := NewYamlMcpProvider(context.Background(), source, "")
    require.NoError(t, err)

    var servers []McpServerProviderRecord
    for r := range records {
        if r.Server != nil {
            servers = append(servers, r)
        }
    }

    assert.Len(t, servers, 3)
}

// Test server filter
func TestMcpServerFilter(t *testing.T) {
    filter, _ := NewMcpServerFilterFromSource(&McpSource{
        IncludedServers: []string{"github-*"},
        ExcludedServers: []string{"*-deprecated"},
    })

    assert.True(t, filter.Allows("github-copilot"))
    assert.False(t, filter.Allows("github-deprecated"))
    assert.False(t, filter.Allows("slack-bot"))
}
```

### Integration Tests

```go
func TestMcpCatalogAPI(t *testing.T) {
    // Setup test server with database
    db := setupTestDB(t)
    service := catalog.NewDbMcpCatalogProvider(db)

    // Load test data
    loader := catalog.NewMcpLoader(service, []string{"testdata/sources.yaml"})
    loader.Start(context.Background())

    // Test list endpoint
    servers, err := service.ListMcpServers(context.Background(), "", "kubernetes", "", "")
    require.NoError(t, err)
    assert.Greater(t, len(servers), 0)
}
```

### Frontend Tests

```typescript
// McpCatalogCard.test.tsx
describe('McpCatalogCard', () => {
  it('renders server information', () => {
    const server: McpServer = {
      name: 'test-server',
      provider: 'Test Provider',
      description: 'A test server',
    };

    render(<McpCatalogCard server={server} />);

    expect(screen.getByText('test-server')).toBeInTheDocument();
    expect(screen.getByText('Test Provider')).toBeInTheDocument();
  });
});
```

## Code Generation

### Generate OpenAPI Models

```bash
# From repository root
cd api/openapi
./generate.sh

# Or using make
make generate-openapi
```

### Generate Goverter Converters

```bash
# If using goverter for type conversion
go generate ./...
```

## Deployment

### Add ConfigMaps

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-catalog-config
data:
  mcp-sources.yaml: |
    catalogs:
      - id: default
        name: Default MCP Servers
        type: yaml
        properties:
          yamlCatalogPath: servers.yaml

  servers.yaml: |
    source: Default
    mcp_servers:
      - name: example-mcp
        provider: Example
        # ...
```

### Update Kustomization

```yaml
# kustomization.yaml
configMapGenerator:
  - name: mcp-catalog-config
    files:
      - mcp-sources.yaml
      - servers.yaml
```

---

[Back to MCP Catalog Index](./README.md) | [Previous: Remaining Work](./remaining-work.md)
