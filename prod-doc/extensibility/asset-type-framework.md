# Asset Type Framework

This document describes the framework for implementing new asset types in the Kubeflow Model Registry.

## Core Concepts

### Entity Hierarchy

The Model Registry uses a hierarchical entity model derived from ML Metadata (MLMD):

```
┌─────────────────────────────────────────────────────────────────┐
│                     Entity Hierarchy                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Context (Container)                                              │
│  ├── RegisteredModel                                              │
│  │   └── ModelVersion (child context)                            │
│  │       └── ModelArtifact (linked artifact)                     │
│  │                                                                │
│  ├── ServingEnvironment                                           │
│  │   └── InferenceService (child context)                        │
│  │                                                                │
│  ├── Experiment                                                   │
│  │   └── ExperimentRun (child context)                           │
│  │       └── Artifacts (linked artifacts)                        │
│  │                                                                │
│  └── [New Asset Types...]                                        │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Property System

Every entity supports typed properties:

```go
type CustomProperties map[string]PropertyValue

type PropertyValue struct {
    // One of:
    BoolValue   *bool    `json:"bool_value,omitempty"`
    IntValue    *int64   `json:"int_value,omitempty"`
    DoubleValue *float64 `json:"double_value,omitempty"`
    StringValue *string  `json:"string_value,omitempty"`
}
```

**Usage**:
```go
model.CustomProperties = CustomProperties{
    "accuracy": PropertyValue{DoubleValue: ptr(0.95)},
    "framework": PropertyValue{StringValue: ptr("tensorflow")},
}
```

### State Management

Entities support state transitions:

```go
type State string

const (
    StateLive     State = "LIVE"
    StateArchived State = "ARCHIVED"
)
```

## Database Model Pattern

### GORM Entity Definition

```go
// internal/db/models/new_entity.go

type NewEntity struct {
    ID          string `gorm:"primaryKey;type:varchar(255)"`
    Name        string `gorm:"type:varchar(255);not null"`
    Description string `gorm:"type:text"`
    ExternalID  string `gorm:"type:varchar(255)"`
    State       string `gorm:"type:varchar(50);default:'LIVE'"`

    // Relationships
    ParentID    string `gorm:"type:varchar(255)"`

    // Custom properties stored as JSON
    CustomProperties string `gorm:"type:json"`

    // Timestamps
    CreateTimeSinceEpoch     int64 `gorm:"autoCreateTime:milli"`
    LastUpdateTimeSinceEpoch int64 `gorm:"autoUpdateTime:milli"`
}

func (NewEntity) TableName() string {
    return "new_entities"
}
```

### Migration

```sql
-- internal/datastore/embedmd/mysql/migrations/000X_add_new_entity.up.sql

CREATE TABLE IF NOT EXISTS new_entities (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    external_id VARCHAR(255),
    state VARCHAR(50) DEFAULT 'LIVE',
    parent_id VARCHAR(255),
    custom_properties JSON,
    create_time_since_epoch BIGINT,
    last_update_time_since_epoch BIGINT,

    INDEX idx_name (name),
    INDEX idx_parent_id (parent_id),
    INDEX idx_state (state)
);
```

## Repository Pattern

### Generic Repository

The codebase provides a generic repository base:

```go
// internal/db/service/generic_repository.go

type GenericRepository[E any, T any] struct {
    db            *gorm.DB
    tableName     string
    schemaToModel func(E) (T, error)
    modelToSchema func(T) (E, error)
}

func NewGenericRepository[E any, T any](
    db *gorm.DB,
    tableName string,
    schemaToModel func(E) (T, error),
    modelToSchema func(T) (E, error),
) *GenericRepository[E, T] {
    return &GenericRepository[E, T]{
        db:            db,
        tableName:     tableName,
        schemaToModel: schemaToModel,
        modelToSchema: modelToSchema,
    }
}
```

### Implementing a New Repository

```go
// internal/db/service/new_entity_repository.go

type NewEntityRepository struct {
    *GenericRepository[models.NewEntity, openapi.NewEntity]
}

func NewNewEntityRepository(db *gorm.DB) *NewEntityRepository {
    return &NewEntityRepository{
        GenericRepository: NewGenericRepository(
            db,
            "new_entities",
            mapNewEntitySchemaToModel,
            mapNewEntityModelToSchema,
        ),
    }
}

func (r *NewEntityRepository) GetByName(name string) (*openapi.NewEntity, error) {
    var entity models.NewEntity
    result := r.db.Where("name = ?", name).First(&entity)
    if result.Error != nil {
        return nil, result.Error
    }
    return r.schemaToModel(entity)
}
```

## OpenAPI Specification

### Entity Definition

```yaml
# api/openapi/model-registry.yaml

components:
  schemas:
    NewEntity:
      type: object
      required:
        - name
      properties:
        id:
          type: string
          readOnly: true
        name:
          type: string
          minLength: 1
          maxLength: 255
        description:
          type: string
        externalId:
          type: string
        state:
          $ref: '#/components/schemas/State'
        customProperties:
          $ref: '#/components/schemas/CustomProperties'
        createTimeSinceEpoch:
          type: integer
          format: int64
          readOnly: true
        lastUpdateTimeSinceEpoch:
          type: integer
          format: int64
          readOnly: true

    NewEntityCreate:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        description:
          type: string
        externalId:
          type: string
        customProperties:
          $ref: '#/components/schemas/CustomProperties'

    NewEntityUpdate:
      type: object
      properties:
        description:
          type: string
        state:
          $ref: '#/components/schemas/State'
        customProperties:
          $ref: '#/components/schemas/CustomProperties'
```

### Endpoint Definition

```yaml
paths:
  /new_entities:
    get:
      operationId: getNewEntities
      summary: List all entities
      parameters:
        - $ref: '#/components/parameters/pageSize'
        - $ref: '#/components/parameters/orderBy'
        - $ref: '#/components/parameters/sortOrder'
        - $ref: '#/components/parameters/nextPageToken'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NewEntityList'
    post:
      operationId: createNewEntity
      summary: Create entity
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewEntityCreate'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NewEntity'

  /new_entities/{id}:
    get:
      operationId: getNewEntity
      summary: Get entity by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NewEntity'
    patch:
      operationId: updateNewEntity
      summary: Update entity
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewEntityUpdate'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NewEntity'
```

## Service Layer Pattern

### Service Interface

```go
// internal/core/new_entity_service.go

type NewEntityService interface {
    GetNewEntities(opts ListOptions) (*openapi.NewEntityList, error)
    GetNewEntity(id string) (*openapi.NewEntity, error)
    CreateNewEntity(entity *openapi.NewEntityCreate) (*openapi.NewEntity, error)
    UpdateNewEntity(id string, update *openapi.NewEntityUpdate) (*openapi.NewEntity, error)
    DeleteNewEntity(id string) error
}

type newEntityService struct {
    repo *service.NewEntityRepository
}

func NewNewEntityService(repo *service.NewEntityRepository) NewEntityService {
    return &newEntityService{repo: repo}
}
```

### Implementation

```go
func (s *newEntityService) GetNewEntities(opts ListOptions) (*openapi.NewEntityList, error) {
    entities, nextToken, err := s.repo.List(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to list entities: %w", err)
    }

    return &openapi.NewEntityList{
        Items:         entities,
        NextPageToken: nextToken,
        PageSize:      int32(opts.PageSize),
        Size:          int32(len(entities)),
    }, nil
}

func (s *newEntityService) CreateNewEntity(create *openapi.NewEntityCreate) (*openapi.NewEntity, error) {
    // Validate
    if create.Name == "" {
        return nil, ErrNameRequired
    }

    // Check uniqueness
    existing, _ := s.repo.GetByName(create.Name)
    if existing != nil {
        return nil, ErrDuplicateName
    }

    // Create
    entity := &openapi.NewEntity{
        Name:             create.Name,
        Description:      create.Description,
        CustomProperties: create.CustomProperties,
        State:            openapi.StateLive,
    }

    return s.repo.Create(entity)
}
```

## Catalog Provider Pattern

For catalog-style entities with external sources:

### Provider Interface

```go
// catalog/internal/catalog/provider.go

type NewCatalogProvider interface {
    GetItems(source, filters string) (*NewItemList, error)
    GetItem(source, name string) (*NewItem, error)
    GetFilterOptions() (*FilterOptions, error)
    Reload() error
}
```

### YAML Provider

```go
type yamlNewCatalogProvider struct {
    sources map[string]*SourceConfig
    items   map[string][]*NewItem
    mu      sync.RWMutex
}

func (p *yamlNewCatalogProvider) loadSource(config *SourceConfig) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Parse YAML
    var items []*NewItem
    if err := yaml.Unmarshal([]byte(config.YAML), &items); err != nil {
        return err
    }

    p.items[config.ID] = items
    return nil
}
```

### Database Provider

```go
type dbNewCatalogProvider struct {
    db *gorm.DB
}

func (p *dbNewCatalogProvider) GetItems(source, filters string) (*NewItemList, error) {
    query := p.db.Model(&models.NewItem{}).Where("source_id = ?", source)

    if filters != "" {
        query = applyFilters(query, filters)
    }

    var items []*models.NewItem
    if err := query.Find(&items).Error; err != nil {
        return nil, err
    }

    return mapToAPIItems(items), nil
}
```

## Frontend Component Pattern

### Page Component

```tsx
// clients/ui/frontend/src/app/pages/newEntity/NewEntityPage.tsx

const NewEntityPage: React.FC = () => {
  const [entities, setEntities] = useState<NewEntity[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    fetchNewEntities()
      .then(setEntities)
      .catch(setError)
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return <Loading />;
  }

  if (error) {
    return <ErrorState error={error} />;
  }

  return (
    <Page>
      <PageSection>
        <Title headingLevel="h1">New Entities</Title>
      </PageSection>
      <PageSection>
        <NewEntityTable entities={entities} />
      </PageSection>
    </Page>
  );
};
```

### Table Component

```tsx
const NewEntityTable: React.FC<{ entities: NewEntity[] }> = ({ entities }) => (
  <Table aria-label="New entities">
    <Thead>
      <Tr>
        <Th>Name</Th>
        <Th>Description</Th>
        <Th>State</Th>
        <Th>Actions</Th>
      </Tr>
    </Thead>
    <Tbody>
      {entities.map((entity) => (
        <Tr key={entity.id}>
          <Td>{entity.name}</Td>
          <Td>{entity.description}</Td>
          <Td>
            <Label color={entity.state === 'LIVE' ? 'green' : 'grey'}>
              {entity.state}
            </Label>
          </Td>
          <Td>
            <ActionsColumn entity={entity} />
          </Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);
```

## Type Conversion Pattern

### Goverter Interface

```go
// internal/converter/new_entity.go

// goverter:converter
// goverter:extend MapCustomProperties
type NewEntityConverter interface {
    // goverter:map CreateTimeSinceEpoch | TimestampToInt64
    // goverter:map LastUpdateTimeSinceEpoch | TimestampToInt64
    SchemaToModel(schema *models.NewEntity) (*openapi.NewEntity, error)

    // goverter:map ID | GenerateID
    ModelToSchema(model *openapi.NewEntityCreate) (*models.NewEntity, error)
}
```

---

[Back to Extensibility Index](./README.md) | [Next: Adding New Assets](./adding-new-assets.md)
