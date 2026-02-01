# Core Service Layer

The Core Service layer implements the business logic for the Model Registry, acting as the central coordinator between the API layer and the data access layer.

## Overview

**Location:** `internal/core/`

**Main File:** `modelregistry_service.go`

**Interface:** `pkg/api/api.go` - `ModelRegistryApi`

## ModelRegistryApi Interface

The core API interface defines all domain operations:

```go
// pkg/api/api.go
type ModelRegistryApi interface {
    // RegisteredModel operations
    UpsertRegisteredModel(registeredModel *openapi.RegisteredModel) (*openapi.RegisteredModel, error)
    GetRegisteredModelById(id string) (*openapi.RegisteredModel, error)
    GetRegisteredModelByParams(name *string, externalId *string) (*openapi.RegisteredModel, error)
    GetRegisteredModels(listOptions ListOptions) (*openapi.RegisteredModelList, error)

    // ModelVersion operations
    UpsertModelVersion(modelVersion *openapi.ModelVersion, registeredModelId *string) (*openapi.ModelVersion, error)
    GetModelVersionById(id string) (*openapi.ModelVersion, error)
    GetModelVersionByParams(versionName *string, registeredModelId *string, externalId *string) (*openapi.ModelVersion, error)
    GetModelVersions(listOptions ListOptions, registeredModelId *string) (*openapi.ModelVersionList, error)

    // Artifact operations
    UpsertArtifact(artifact *openapi.Artifact) (*openapi.Artifact, error)
    GetArtifactById(id string) (*openapi.Artifact, error)
    GetArtifacts(artifactType openapi.ArtifactTypeQueryParam, listOptions ListOptions, parentResourceId *string) (*openapi.ArtifactList, error)

    // InferenceService operations
    UpsertInferenceService(inferenceService *openapi.InferenceService) (*openapi.InferenceService, error)
    GetInferenceServiceById(id string) (*openapi.InferenceService, error)
    GetInferenceServices(listOptions ListOptions, servingEnvironmentId *string, runtime *string) (*openapi.InferenceServiceList, error)

    // ServingEnvironment operations
    UpsertServingEnvironment(servingEnvironment *openapi.ServingEnvironment) (*openapi.ServingEnvironment, error)
    GetServingEnvironmentById(id string) (*openapi.ServingEnvironment, error)
    GetServingEnvironments(listOptions ListOptions) (*openapi.ServingEnvironmentList, error)

    // ServeModel operations
    UpsertServeModel(serveModel *openapi.ServeModel) (*openapi.ServeModel, error)
    GetServeModelById(id string) (*openapi.ServeModel, error)
    GetServeModels(listOptions ListOptions, inferenceServiceId *string) (*openapi.ServeModelList, error)

    // Experiment operations
    UpsertExperiment(experiment *openapi.Experiment) (*openapi.Experiment, error)
    GetExperimentById(id string) (*openapi.Experiment, error)
    GetExperiments(listOptions ListOptions) (*openapi.ExperimentList, error)

    // ExperimentRun operations
    UpsertExperimentRun(experimentRun *openapi.ExperimentRun) (*openapi.ExperimentRun, error)
    GetExperimentRunById(id string) (*openapi.ExperimentRun, error)
    GetExperimentRuns(listOptions ListOptions, experimentId *string) (*openapi.ExperimentRunList, error)

    // Relationship navigation
    GetRegisteredModelByInferenceService(inferenceServiceId string) (*openapi.RegisteredModel, error)
    GetModelVersionByInferenceService(inferenceServiceId string) (*openapi.ModelVersion, error)
    GetModelArtifactByInferenceService(inferenceServiceId string) (*openapi.ModelArtifact, error)
}
```

## ModelRegistryService Implementation

### Constructor

```go
// internal/core/modelregistry_service.go
type ModelRegistryService struct {
    registeredModelRepo     models.RegisteredModelRepository
    modelVersionRepo        models.ModelVersionRepository
    modelArtifactRepo       models.ArtifactRepository
    docArtifactRepo         models.ArtifactRepository
    servingEnvironmentRepo  models.ServingEnvironmentRepository
    inferenceServiceRepo    models.InferenceServiceRepository
    serveModelRepo          models.ServeModelRepository
    experimentRepo          models.ExperimentRepository
    experimentRunRepo       models.ExperimentRunRepository
    mapper                  *mapper.EmbedMDMapper
}

func NewModelRegistryService(repoSet datastore.RepoSet, mapper *mapper.EmbedMDMapper) (*ModelRegistryService, error) {
    // Repository retrieval via reflection
    registeredModelRepo, err := getRepo[models.RegisteredModelRepository](repoSet)
    if err != nil {
        return nil, err
    }
    // ... retrieve other repositories

    return &ModelRegistryService{
        registeredModelRepo: registeredModelRepo,
        // ... other repositories
        mapper: mapper,
    }, nil
}
```

### Dependency Injection Pattern

Repositories are injected via the constructor, enabling:
- **Testability** - Mock repositories for unit tests
- **Flexibility** - Swap implementations without changing service
- **Clarity** - Explicit dependencies

### Repository Retrieval

```go
func getRepo[T any](repoSet datastore.RepoSet) (T, error) {
    var t T
    repo, err := repoSet.Repository(reflect.TypeOf(&t).Elem())
    if err != nil {
        return t, err
    }
    return repo.(T), nil
}
```

## Business Logic Patterns

### Upsert Pattern

All entity operations use an Upsert pattern (create or update):

```go
func (b *ModelRegistryService) UpsertRegisteredModel(
    registeredModel *openapi.RegisteredModel,
) (*openapi.RegisteredModel, error) {
    // 1. Determine if update or create
    if registeredModel.Id != nil {
        // Update existing
        existing, err := b.GetRegisteredModelById(*registeredModel.Id)
        if err != nil {
            return nil, err
        }

        // Apply update mask (preserve read-only fields)
        registeredModel.CreateTimeSinceEpoch = existing.CreateTimeSinceEpoch
    }

    // 2. Convert OpenAPI model to domain model
    entity, err := b.mapper.MapFromRegisteredModel(registeredModel)
    if err != nil {
        return nil, err
    }

    // 3. Save via repository
    var saved *models.RegisteredModel
    if entity.ID != nil {
        saved, err = b.registeredModelRepo.Update(entity)
    } else {
        saved, err = b.registeredModelRepo.Create(entity)
    }
    if err != nil {
        return nil, err
    }

    // 4. Convert back to OpenAPI model
    return b.mapper.MapToRegisteredModel(saved)
}
```

### Error Handling

```go
// Domain errors
var (
    ErrBadRequest = errors.New("bad request")
    ErrNotFound   = errors.New("not found")
    ErrConflict   = errors.New("conflict")
)

// Error wrapping with context
if errors.Is(err, gorm.ErrDuplicatedKey) {
    return nil, fmt.Errorf("%w: registered model with name '%s' already exists", api.ErrConflict, *model.Name)
}

// Error translation to HTTP status
func ErrToStatus(err error) int {
    switch {
    case errors.Is(err, ErrBadRequest):
        return http.StatusBadRequest
    case errors.Is(err, ErrNotFound):
        return http.StatusNotFound
    case errors.Is(err, ErrConflict):
        return http.StatusConflict
    default:
        return http.StatusInternalServerError
    }
}
```

### Relationship Navigation

For entities linked via properties (not foreign keys):

```go
func (b *ModelRegistryService) GetRegisteredModelByInferenceService(
    inferenceServiceId string,
) (*openapi.RegisteredModel, error) {
    // 1. Get the inference service
    inferenceService, err := b.GetInferenceServiceById(inferenceServiceId)
    if err != nil {
        return nil, err
    }

    // 2. Extract registered_model_id from properties
    registeredModelId := inferenceService.RegisteredModelId
    if registeredModelId == nil {
        return nil, fmt.Errorf("%w: inference service has no registered model", api.ErrNotFound)
    }

    // 3. Fetch the registered model
    return b.GetRegisteredModelById(*registeredModelId)
}
```

### Child Entity Handling

Child entities (ModelVersion, ExperimentRun) store names with parent prefix:

```go
func (b *ModelRegistryService) UpsertModelVersion(
    modelVersion *openapi.ModelVersion,
    registeredModelId *string,
) (*openapi.ModelVersion, error) {
    // Validate parent exists
    if registeredModelId == nil {
        return nil, fmt.Errorf("%w: registeredModelId is required", api.ErrBadRequest)
    }

    _, err := b.GetRegisteredModelById(*registeredModelId)
    if err != nil {
        return nil, fmt.Errorf("%w: parent registered model not found", api.ErrBadRequest)
    }

    // Name is stored as "parentId:versionName"
    modelVersion.RegisteredModelId = registeredModelId

    // ... continue with upsert
}
```

## List Operations

### ListOptions

```go
type ListOptions struct {
    PageSize      *int32
    OrderBy       *string
    SortOrder     *string
    NextPageToken *string
    FilterQuery   *string
}
```

### Paginated Results

```go
func (b *ModelRegistryService) GetRegisteredModels(
    listOptions ListOptions,
) (*openapi.RegisteredModelList, error) {
    // Build list options for repository
    opts := models.RegisteredModelListOptions{
        Pagination: &models.Pagination{
            PageSize:      listOptions.PageSize,
            OrderBy:       listOptions.OrderBy,
            SortOrder:     listOptions.SortOrder,
            NextPageToken: listOptions.NextPageToken,
        },
        FilterQuery: listOptions.FilterQuery,
    }

    // Fetch from repository
    entities, pagination, err := b.registeredModelRepo.List(opts)
    if err != nil {
        return nil, err
    }

    // Convert to OpenAPI models
    items := make([]openapi.RegisteredModel, len(entities))
    for i, entity := range entities {
        item, err := b.mapper.MapToRegisteredModel(&entity)
        if err != nil {
            return nil, err
        }
        items[i] = *item
    }

    return &openapi.RegisteredModelList{
        Items:         items,
        NextPageToken: pagination.NextPageToken,
        PageSize:      pagination.PageSize,
        Size:          int32(len(items)),
    }, nil
}
```

## File Organization

```
internal/core/
├── modelregistry_service.go      # Main service struct and constructor
├── registered_model.go           # RegisteredModel operations
├── model_version.go              # ModelVersion operations
├── artifact.go                   # Artifact operations
├── inference_service.go          # InferenceService operations
├── serving_environment.go        # ServingEnvironment operations
├── serve_model.go                # ServeModel operations
├── experiment.go                 # Experiment operations
├── experiment_run.go             # ExperimentRun operations
└── *_test.go                     # Unit tests
```

## Testing

### Unit Tests

```go
func TestUpsertRegisteredModel(t *testing.T) {
    // Create mock repositories
    mockRepo := &mocks.MockRegisteredModelRepository{}
    mockMapper := &mocks.MockMapper{}

    // Create service with mocks
    service := &ModelRegistryService{
        registeredModelRepo: mockRepo,
        mapper:             mockMapper,
    }

    // Test create
    model := &openapi.RegisteredModel{Name: ptr("test-model")}
    mockMapper.On("MapFromRegisteredModel", model).Return(&models.RegisteredModel{}, nil)
    mockRepo.On("Create", mock.Anything).Return(&models.RegisteredModel{ID: ptr(int32(1))}, nil)
    mockMapper.On("MapToRegisteredModel", mock.Anything).Return(&openapi.RegisteredModel{Id: ptr("1")}, nil)

    result, err := service.UpsertRegisteredModel(model)
    assert.NoError(t, err)
    assert.Equal(t, "1", *result.Id)
}
```

---

[Back to Backend Index](./README.md) | [Next: Repository Pattern](./repository-pattern.md)
