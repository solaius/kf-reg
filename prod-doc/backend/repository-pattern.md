# Repository Pattern

The Model Registry uses a **Generic Repository Pattern** with Go generics for type-safe, reusable data access.

## Overview

**Location:** `internal/db/service/`

**Key File:** `generic_repository.go`

## GenericRepository

### Type Parameters

```go
type GenericRepository[
    TEntity any,           // Domain entity type (e.g., models.RegisteredModel)
    TSchema SchemaEntity,  // GORM schema type (e.g., schema.Context)
    TProp PropertyEntity,  // Property type (e.g., schema.ContextProperty)
    TListOpts BaseListOptions, // List options type
] struct {
    config GenericRepositoryConfig[TEntity, TSchema, TProp, TListOpts]
}
```

### Configuration

```go
type GenericRepositoryConfig[TEntity, TSchema, TProp, TListOpts any] struct {
    // Required
    EntityToSchema       func(entity TEntity) TSchema
    SchemaToEntity       func(schema TSchema, props []TProp) TEntity
    EntityToProperties   func(entity TEntity, schemaID int32) []TProp

    // Optional
    ApplyListFilters          func(opts TListOpts, db *gorm.DB) *gorm.DB
    CreatePaginationToken     func(entity TEntity, orderBy string) string
    ApplyCustomOrdering       func(db *gorm.DB, orderBy string, sortOrder string) *gorm.DB
    PreserveHistoricalTimes   bool // For catalog loading
}
```

## Repository Interface Pattern

Each entity type defines its repository interface:

```go
// internal/db/models/registered_model.go
type RegisteredModelRepository interface {
    Create(entity *RegisteredModel) (*RegisteredModel, error)
    Update(entity *RegisteredModel) (*RegisteredModel, error)
    GetByID(id int32) (*RegisteredModel, error)
    GetByName(name string) (*RegisteredModel, error)
    List(opts RegisteredModelListOptions) ([]RegisteredModel, *Pagination, error)
}
```

## Mapper Functions

### Entity to Schema

Converts domain entity to GORM schema for database operations:

```go
func mapRegisteredModelToContext(model models.RegisteredModel) schema.Context {
    ctx := schema.Context{
        TypeID:                   *model.TypeID,
        Name:                     model.Attributes.Name,
        ExternalID:               model.Attributes.ExternalID,
        CreateTimeSinceEpoch:     model.Attributes.CreateTimeSinceEpoch,
        LastUpdateTimeSinceEpoch: model.Attributes.LastUpdateTimeSinceEpoch,
    }
    if model.ID != nil {
        ctx.ID = *model.ID
    }
    return ctx
}
```

### Schema to Entity

Converts GORM schema back to domain entity:

```go
func mapContextToRegisteredModel(
    ctx schema.Context,
    props []schema.ContextProperty,
    typeID int32,
) models.RegisteredModel {
    model := models.RegisteredModel{
        BaseEntity: models.BaseEntity[models.RegisteredModelAttributes]{
            ID:     &ctx.ID,
            TypeID: &typeID,
            Attributes: &models.RegisteredModelAttributes{
                Name:                     ctx.Name,
                ExternalID:               ctx.ExternalID,
                CreateTimeSinceEpoch:     ctx.CreateTimeSinceEpoch,
                LastUpdateTimeSinceEpoch: ctx.LastUpdateTimeSinceEpoch,
            },
        },
    }

    // Extract properties
    properties, customProperties := extractProperties(props)
    model.Properties = &properties
    model.CustomProperties = &customProperties

    return model
}
```

### Entity to Properties

Converts entity properties to database property rows:

```go
func mapRegisteredModelToContextProperties(
    model models.RegisteredModel,
    contextID int32,
) []schema.ContextProperty {
    var props []schema.ContextProperty

    // Standard properties
    for _, prop := range *model.Properties {
        props = append(props, schema.ContextProperty{
            ContextID:        contextID,
            Name:             prop.Name,
            IsCustomProperty: false,
            // Set appropriate value field based on type
        })
    }

    // Custom properties
    for _, prop := range *model.CustomProperties {
        props = append(props, schema.ContextProperty{
            ContextID:        contextID,
            Name:             prop.Name,
            IsCustomProperty: true,
            // Set appropriate value field based on type
        })
    }

    return props
}
```

## CRUD Operations

### Create

```go
func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) Create(
    entity *TEntity,
) (*TEntity, error) {
    db := r.config.DB

    // Convert entity to schema
    schemaEntity := r.config.EntityToSchema(*entity)

    // Create in database
    if err := db.Create(&schemaEntity).Error; err != nil {
        return nil, err
    }

    // Create properties
    props := r.config.EntityToProperties(*entity, schemaEntity.GetID())
    if len(props) > 0 {
        if err := db.Create(&props).Error; err != nil {
            return nil, err
        }
    }

    // Fetch and return complete entity
    return r.GetByID(schemaEntity.GetID())
}
```

### Update

```go
func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) Update(
    entity *TEntity,
) (*TEntity, error) {
    db := r.config.DB

    // Convert and update schema
    schemaEntity := r.config.EntityToSchema(*entity)
    if err := db.Save(&schemaEntity).Error; err != nil {
        return nil, err
    }

    // Delete old properties
    if err := db.Where("context_id = ?", schemaEntity.GetID()).Delete(&TProp{}).Error; err != nil {
        return nil, err
    }

    // Create new properties
    props := r.config.EntityToProperties(*entity, schemaEntity.GetID())
    if len(props) > 0 {
        if err := db.Create(&props).Error; err != nil {
            return nil, err
        }
    }

    return r.GetByID(schemaEntity.GetID())
}
```

### GetByID

```go
func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) GetByID(
    id int32,
) (*TEntity, error) {
    db := r.config.DB

    // Fetch schema entity
    var schemaEntity TSchema
    if err := db.Where("id = ?", id).First(&schemaEntity).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, api.ErrNotFound
        }
        return nil, err
    }

    // Fetch properties
    var props []TProp
    if err := db.Where("context_id = ?", id).Find(&props).Error; err != nil {
        return nil, err
    }

    // Convert to entity
    entity := r.config.SchemaToEntity(schemaEntity, props)
    return &entity, nil
}
```

### List

```go
func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) List(
    opts TListOpts,
) ([]TEntity, *models.Pagination, error) {
    db := r.config.DB

    // Apply filters
    if r.config.ApplyListFilters != nil {
        db = r.config.ApplyListFilters(opts, db)
    }

    // Apply pagination
    db = scopes.PaginateWithOptions(
        &[]TSchema{},
        opts.GetPagination(),
        db,
        r.config.TablePrefix,
        r.config.AllowedOrderByColumns,
    )

    // Execute query
    var schemas []TSchema
    if err := db.Find(&schemas).Error; err != nil {
        return nil, nil, err
    }

    // Convert to entities
    entities := make([]TEntity, len(schemas))
    for i, schema := range schemas {
        props, _ := r.fetchPropertiesForSchema(schema.GetID())
        entities[i] = r.config.SchemaToEntity(schema, props)
    }

    // Build pagination response
    pagination := r.buildPagination(entities, opts)

    return entities, pagination, nil
}
```

## Pagination

### Cursor-Based Pagination

```go
// internal/db/scopes/paginate.go
func PaginateWithOptions(
    value any,
    pagination *models.Pagination,
    db *gorm.DB,
    tablePrefix string,
    customAllowedColumns map[string]string,
) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        // Validate orderBy column
        orderBy := validateOrderBy(pagination.OrderBy, customAllowedColumns)
        sortOrder := validateSortOrder(pagination.SortOrder)

        // Apply ordering
        db = db.Order(fmt.Sprintf("%s.%s %s", tablePrefix, orderBy, sortOrder))

        // Apply cursor
        if pagination.NextPageToken != nil {
            cursor, _ := DecodeCursor(*pagination.NextPageToken)
            if sortOrder == "ASC" {
                db = db.Where(fmt.Sprintf("%s.%s > ?", tablePrefix, orderBy), cursor.Value)
            } else {
                db = db.Where(fmt.Sprintf("%s.%s < ?", tablePrefix, orderBy), cursor.Value)
            }
        }

        // Apply limit
        db = db.Limit(int(*pagination.PageSize) + 1) // Fetch one extra to detect next page

        return db
    }
}
```

### Cursor Encoding

```go
func EncodeCursor(id int32, value string) string {
    return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d:%s", id, value)))
}

func DecodeCursor(token string) (*Cursor, error) {
    // Validate size
    if len(token) > 1024 {
        return nil, errors.New("cursor too large")
    }

    decoded, err := base64.StdEncoding.DecodeString(token)
    if err != nil {
        return nil, err
    }

    parts := strings.SplitN(string(decoded), ":", 2)
    id, _ := strconv.ParseInt(parts[0], 10, 32)

    return &Cursor{
        ID:    int32(id),
        Value: parts[1],
    }, nil
}
```

## List Options

### Base Interface

```go
type BaseListOptions interface {
    GetPagination() *Pagination
    GetFilterQuery() *string
}
```

### Entity-Specific Options

```go
type RegisteredModelListOptions struct {
    Pagination  *Pagination
    FilterQuery *string
}

type ModelVersionListOptions struct {
    Pagination        *Pagination
    FilterQuery       *string
    RegisteredModelID *string  // Parent filter
}

type InferenceServiceListOptions struct {
    Pagination           *Pagination
    FilterQuery          *string
    ServingEnvironmentID *string
    Runtime              *string
}
```

## Filtering

### Filter Application

```go
func applyRegisteredModelFilters(opts RegisteredModelListOptions, db *gorm.DB) *gorm.DB {
    // Apply filter query (advanced filtering)
    if opts.FilterQuery != nil && *opts.FilterQuery != "" {
        queryBuilder := filter.NewQueryBuilder(
            filter.EntityTypeContext,
            filter.RestEntityTypeRegisteredModel,
            "Context",
            entityMappings,
        )
        db = queryBuilder.ApplyFilter(db, *opts.FilterQuery)
    }

    return db
}
```

## Repository Implementations

### RegisteredModelRepository

```go
type RegisteredModelRepositoryImpl struct {
    *GenericRepository[
        models.RegisteredModel,
        schema.Context,
        schema.ContextProperty,
        models.RegisteredModelListOptions,
    ]
    db     *gorm.DB
    typeID int32
}

func NewRegisteredModelRepository(db *gorm.DB, typeID int32) *RegisteredModelRepositoryImpl {
    return &RegisteredModelRepositoryImpl{
        GenericRepository: NewGenericRepository(GenericRepositoryConfig{
            DB:               db,
            TypeID:           typeID,
            EntityToSchema:   mapRegisteredModelToContext,
            SchemaToEntity:   mapContextToRegisteredModel,
            EntityToProperties: mapRegisteredModelToContextProperties,
            ApplyListFilters: applyRegisteredModelFilters,
            TablePrefix:      "Context",
        }),
        db:     db,
        typeID: typeID,
    }
}
```

---

[Back to Backend Index](./README.md) | [Previous: Core Service](./core-service.md) | [Next: Datastore Abstraction](./datastore-abstraction.md)
