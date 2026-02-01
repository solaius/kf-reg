# Datastore Abstraction

The Model Registry uses a **pluggable datastore architecture** that allows different backend implementations while maintaining a consistent API.

## Overview

**Location:** `internal/datastore/`

**Key Files:**
- `connector.go` - Connector interface and registration
- `repos.go` - Repository specification and RepoSet interface

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Application Layer                               │
│                   (ModelRegistryService)                             │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Datastore Abstraction                             │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Connector Interface                       │   │
│  │  - Type() string                                             │   │
│  │  - Connect(spec *Spec) (RepoSet, error)                     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                       │
│              ┌───────────────┼───────────────┐                      │
│              ▼               ▼               ▼                      │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐       │
│  │  EmbedMD MySQL  │ │EmbedMD Postgres │ │ Future Backend  │       │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘
```

## Connector Interface

```go
// internal/datastore/connector.go
type Connector interface {
    // Type returns the connector type identifier
    Type() string

    // Connect establishes connection and returns repository set
    Connect(spec *Spec) (RepoSet, error)
}
```

## RepoSet Interface

```go
type RepoSet interface {
    // TypeMap returns mapping of type names to database IDs
    TypeMap() map[string]int32

    // Repository retrieves a repository by its interface type
    Repository(t reflect.Type) (any, error)
}
```

## Connector Registration

### Registration Pattern

```go
// Global registry
var connectors = make(map[string]func(config any) (Connector, error))

// Register a connector factory
func Register(t string, fn func(config any) (Connector, error)) {
    connectors[t] = fn
}

// Create connector instance
func NewConnector(t string, config any) (Connector, error) {
    factory, ok := connectors[t]
    if !ok {
        return nil, fmt.Errorf("unknown connector type: %s", t)
    }
    return factory(config)
}
```

### EmbedMD Registration

```go
// internal/datastore/embedmd/service.go
const connectorType = "embedmd"

func init() {
    datastore.Register(connectorType, func(cfg any) (datastore.Connector, error) {
        emdbCfg, ok := cfg.(*EmbedMDConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for embedmd connector")
        }
        return NewEmbedMDService(emdbCfg)
    })
}
```

## Specification System

### Spec Structure

```go
// internal/datastore/repos.go
type Spec struct {
    ArtifactTypes  map[string]*SpecType
    ContextTypes   map[string]*SpecType
    ExecutionTypes map[string]*SpecType
    Others         []any  // Non-type-mapped repositories
}
```

### SpecType

```go
type SpecType struct {
    // Init function for repository creation
    // Signature: func(db *gorm.DB, typeID int32) RepositoryInterface
    InitFn any

    // Property schema for this type
    Properties map[string]PropertyType
}
```

### Property Types

```go
type PropertyType int

const (
    PropertyTypeString PropertyType = iota
    PropertyTypeInt
    PropertyTypeDouble
    PropertyTypeBool
    PropertyTypeBytes
    PropertyTypeProto
)
```

## Specification Building

### Builder Pattern

```go
// Create specification for Model Registry
func BuildSpec() *datastore.Spec {
    spec := &datastore.Spec{
        ContextTypes:   make(map[string]*datastore.SpecType),
        ArtifactTypes:  make(map[string]*datastore.SpecType),
        ExecutionTypes: make(map[string]*datastore.SpecType),
    }

    // Register context types
    spec.ContextTypes[defaults.RegisteredModelTypeName] = &datastore.SpecType{
        InitFn: service.NewRegisteredModelRepository,
        Properties: map[string]datastore.PropertyType{
            "description": datastore.PropertyTypeString,
            "owner":       datastore.PropertyTypeString,
            "state":       datastore.PropertyTypeString,
        },
    }

    spec.ContextTypes[defaults.ServingEnvironmentTypeName] = &datastore.SpecType{
        InitFn: service.NewServingEnvironmentRepository,
        Properties: map[string]datastore.PropertyType{
            "description": datastore.PropertyTypeString,
        },
    }

    // Register artifact types
    spec.ArtifactTypes[defaults.ModelVersionTypeName] = &datastore.SpecType{
        InitFn: service.NewModelVersionRepository,
        Properties: map[string]datastore.PropertyType{
            "description":         datastore.PropertyTypeString,
            "author":              datastore.PropertyTypeString,
            "state":               datastore.PropertyTypeString,
            "registered_model_id": datastore.PropertyTypeInt,
        },
    }

    // Register execution types
    spec.ExecutionTypes[defaults.InferenceServiceTypeName] = &datastore.SpecType{
        InitFn: service.NewInferenceServiceRepository,
        Properties: map[string]datastore.PropertyType{
            "description":            datastore.PropertyTypeString,
            "runtime":                datastore.PropertyTypeString,
            "state":                  datastore.PropertyTypeString,
            "serving_environment_id": datastore.PropertyTypeInt,
            "registered_model_id":    datastore.PropertyTypeInt,
            "model_version_id":       datastore.PropertyTypeInt,
        },
    }

    return spec
}
```

## EmbedMD Implementation

### Configuration

```go
// internal/datastore/embedmd/service.go
type EmbedMDConfig struct {
    DatabaseType string        // "mysql" or "postgres"
    DatabaseDSN  string        // Connection string
    TLSConfig    *tls.TLSConfig // Optional TLS configuration
    DB           *gorm.DB      // Optional pre-connected instance
}
```

### Service Implementation

```go
type EmbedMDService struct {
    config *EmbedMDConfig
}

func NewEmbedMDService(config *EmbedMDConfig) (*EmbedMDService, error) {
    return &EmbedMDService{config: config}, nil
}

func (s *EmbedMDService) Type() string {
    return connectorType
}

func (s *EmbedMDService) Connect(spec *datastore.Spec) (datastore.RepoSet, error) {
    // Connect to database
    db, err := s.connectDB()
    if err != nil {
        return nil, err
    }

    // Run migrations
    if err := s.migrate(db); err != nil {
        return nil, err
    }

    // Create repository set
    return newRepoSet(db, spec)
}
```

### RepoSet Implementation

```go
// internal/datastore/embedmd/repos.go
type repoSetImpl struct {
    db        *gorm.DB
    spec      *datastore.Spec
    nameIDMap map[string]int32
    repos     map[reflect.Type]any
}

func newRepoSet(db *gorm.DB, spec *datastore.Spec) (*repoSetImpl, error) {
    repoSet := &repoSetImpl{
        db:        db,
        spec:      spec,
        nameIDMap: make(map[string]int32),
        repos:     make(map[reflect.Type]any),
    }

    // Initialize types and repositories
    if err := repoSet.initTypes(); err != nil {
        return nil, err
    }

    if err := repoSet.initRepos(); err != nil {
        return nil, err
    }

    return repoSet, nil
}

func (r *repoSetImpl) TypeMap() map[string]int32 {
    return r.nameIDMap
}

func (r *repoSetImpl) Repository(t reflect.Type) (any, error) {
    repo, ok := r.repos[t]
    if !ok {
        return nil, fmt.Errorf("repository not found for type: %v", t)
    }
    return repo, nil
}
```

### Type Initialization

```go
func (r *repoSetImpl) initTypes() error {
    // Ensure all required types exist in database
    for typeName := range r.spec.ContextTypes {
        typeID, err := r.ensureType(typeName, "CONTEXT")
        if err != nil {
            return err
        }
        r.nameIDMap[typeName] = typeID
    }

    for typeName := range r.spec.ArtifactTypes {
        typeID, err := r.ensureType(typeName, "ARTIFACT")
        if err != nil {
            return err
        }
        r.nameIDMap[typeName] = typeID
    }

    for typeName := range r.spec.ExecutionTypes {
        typeID, err := r.ensureType(typeName, "EXECUTION")
        if err != nil {
            return err
        }
        r.nameIDMap[typeName] = typeID
    }

    return nil
}

func (r *repoSetImpl) ensureType(name string, kind string) (int32, error) {
    var t schema.Type
    result := r.db.Where("name = ?", name).First(&t)

    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            // Create type
            t = schema.Type{Name: name, TypeKind: kind}
            if err := r.db.Create(&t).Error; err != nil {
                return 0, err
            }
        } else {
            return 0, result.Error
        }
    }

    return t.ID, nil
}
```

### Repository Initialization

```go
func (r *repoSetImpl) initRepos() error {
    // Initialize context type repositories
    for typeName, specType := range r.spec.ContextTypes {
        typeID := r.nameIDMap[typeName]
        repo, err := r.callInitFn(specType.InitFn, typeID)
        if err != nil {
            return err
        }
        r.repos[reflect.TypeOf(repo)] = repo
    }

    // Similar for artifact and execution types...
    return nil
}

func (r *repoSetImpl) callInitFn(initFn any, typeID int32) (any, error) {
    fn := reflect.ValueOf(initFn)

    // Build arguments based on function signature
    args := []reflect.Value{
        reflect.ValueOf(r.db),
        reflect.ValueOf(typeID),
    }

    // Call the init function
    results := fn.Call(args)

    return results[0].Interface(), nil
}
```

## Usage in Proxy Server

```go
// cmd/proxy.go
func newModelRegistryService(cfg *ProxyConfig) (*core.ModelRegistryService, error) {
    // Create connector configuration
    embedMDConfig := &embedmd.EmbedMDConfig{
        DatabaseType: cfg.EmbedMD.DatabaseType,
        DatabaseDSN:  cfg.EmbedMD.DatabaseDSN,
        TLSConfig:    cfg.EmbedMD.TLSConfig,
    }

    // Create connector
    connector, err := datastore.NewConnector("embedmd", embedMDConfig)
    if err != nil {
        return nil, err
    }

    // Build specification
    spec := BuildSpec()

    // Connect and get repository set
    repoSet, err := connector.Connect(spec)
    if err != nil {
        return nil, err
    }

    // Create mapper with type map
    mapper := mapper.NewEmbedMDMapper(repoSet.TypeMap())

    // Create service
    return core.NewModelRegistryService(repoSet, mapper)
}
```

## Adding New Backends

To add a new datastore backend:

1. **Implement Connector interface:**

```go
type NewBackendConnector struct {
    config *NewBackendConfig
}

func (c *NewBackendConnector) Type() string {
    return "new-backend"
}

func (c *NewBackendConnector) Connect(spec *Spec) (RepoSet, error) {
    // Connect to backend
    // Initialize repositories
    // Return RepoSet implementation
}
```

2. **Register in init:**

```go
func init() {
    datastore.Register("new-backend", func(cfg any) (datastore.Connector, error) {
        return NewBackendConnector(cfg.(*NewBackendConfig))
    })
}
```

3. **Implement RepoSet:**

```go
type newBackendRepoSet struct {
    // Backend-specific fields
}

func (r *newBackendRepoSet) TypeMap() map[string]int32 {
    // Return type name to ID mapping
}

func (r *newBackendRepoSet) Repository(t reflect.Type) (any, error) {
    // Return repository for given type
}
```

---

[Back to Backend Index](./README.md) | [Previous: Repository Pattern](./repository-pattern.md) | [Next: Database Layer](./database-layer.md)
