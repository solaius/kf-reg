# BFF Repositories

This document covers the data access layer in the BFF.

## Overview

Repositories abstract data access to external services:

- **Model Registry Client**: Calls Model Registry API
- **Model Catalog Client**: Calls Catalog Service API
- **Kubernetes Repositories**: Access K8s resources

## Repository Structure

```go
// internal/repositories/repositories.go
type Repositories struct {
    ModelRegistryClient ModelRegistryClientInterface
    ModelCatalogClient  ModelCatalogClientInterface
    User                *UserRepository
    ModelRegistry       *ModelRegistryRepository
    Namespace           *NamespaceRepository
    HealthCheck         *HealthCheckRepository
}

func NewRepositories(
    mrClient ModelRegistryClientInterface,
    catalogClient ModelCatalogClientInterface,
) *Repositories {
    return &Repositories{
        ModelRegistryClient: mrClient,
        ModelCatalogClient:  catalogClient,
        User:                &UserRepository{},
        ModelRegistry:       &ModelRegistryRepository{},
        Namespace:           &NamespaceRepository{},
        HealthCheck:         &HealthCheckRepository{},
    }
}
```

## Model Registry Client

### Interface

```go
// internal/repositories/model_registry_client.go
type ModelRegistryClientInterface interface {
    // Registered Models
    GetAllRegisteredModels(client HTTPClientInterface, params url.Values) (*models.RegisteredModelList, error)
    GetRegisteredModel(client HTTPClientInterface, id string) (*models.RegisteredModel, error)
    CreateRegisteredModel(client HTTPClientInterface, data models.RegisteredModelCreate) (*models.RegisteredModel, error)
    UpdateRegisteredModel(client HTTPClientInterface, id string, data models.RegisteredModelUpdate) (*models.RegisteredModel, error)

    // Model Versions
    GetAllModelVersions(client HTTPClientInterface, params url.Values) (*models.ModelVersionList, error)
    GetModelVersion(client HTTPClientInterface, id string) (*models.ModelVersion, error)
    CreateModelVersion(client HTTPClientInterface, data models.ModelVersionCreate) (*models.ModelVersion, error)
    UpdateModelVersion(client HTTPClientInterface, id string, data models.ModelVersionUpdate) (*models.ModelVersion, error)
    GetAllModelVersionsForRegisteredModel(client HTTPClientInterface, modelId string, params url.Values) (*models.ModelVersionList, error)

    // Model Artifacts
    GetAllModelArtifacts(client HTTPClientInterface, params url.Values) (*models.ModelArtifactList, error)
    GetModelArtifact(client HTTPClientInterface, id string) (*models.ModelArtifact, error)
    CreateModelArtifact(client HTTPClientInterface, data models.ModelArtifactCreate) (*models.ModelArtifact, error)
    UpdateModelArtifact(client HTTPClientInterface, id string, data models.ModelArtifactUpdate) (*models.ModelArtifact, error)
}
```

### Implementation

```go
type ModelRegistryClient struct {
    logger *slog.Logger
}

func NewModelRegistryClient(logger *slog.Logger) (*ModelRegistryClient, error) {
    return &ModelRegistryClient{logger: logger}, nil
}

func (c *ModelRegistryClient) GetAllRegisteredModels(client HTTPClientInterface, params url.Values) (*models.RegisteredModelList, error) {
    path := "/api/model_registry/v1alpha3/registered_models"
    if len(params) > 0 {
        path = path + "?" + params.Encode()
    }

    resp, err := client.GET(path)
    if err != nil {
        return nil, fmt.Errorf("failed to get registered models: %w", err)
    }

    var result models.RegisteredModelList
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal registered models: %w", err)
    }

    return &result, nil
}

func (c *ModelRegistryClient) CreateRegisteredModel(client HTTPClientInterface, data models.RegisteredModelCreate) (*models.RegisteredModel, error) {
    path := "/api/model_registry/v1alpha3/registered_models"

    body, err := json.Marshal(data)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal registered model: %w", err)
    }

    resp, err := client.POST(path, body)
    if err != nil {
        return nil, fmt.Errorf("failed to create registered model: %w", err)
    }

    var result models.RegisteredModel
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal registered model: %w", err)
    }

    return &result, nil
}
```

## Model Catalog Client

### Interface

```go
// internal/repositories/model_catalog.go
type ModelCatalogClientInterface interface {
    // Catalog Models
    GetAllCatalogModels(client HTTPClientInterface, params url.Values) (*models.CatalogModelList, error)
    GetCatalogModel(client HTTPClientInterface, sourceId, modelName string) (*models.CatalogModel, error)
    GetCatalogModelArtifacts(client HTTPClientInterface, sourceId, modelName string, params url.Values) (*models.CatalogArtifactList, error)
    GetPerformanceArtifacts(client HTTPClientInterface, sourceId, modelName string) (*models.PerformanceArtifactList, error)

    // Sources
    GetAllCatalogSources(client HTTPClientInterface, params url.Values) (*models.CatalogSourceList, error)

    // Filters
    GetFilterOptions(client HTTPClientInterface) (*models.FilterOptions, error)

    // MCP Servers
    GetAllMcpServers(client HTTPClientInterface, params url.Values) (*models.McpServerList, error)
    GetMcpServer(client HTTPClientInterface, serverId string) (*models.McpServer, error)
    GetMcpFilterOptions(client HTTPClientInterface) (*models.FilterOptionsList, error)
    GetAllMcpSources(client HTTPClientInterface, params url.Values) (*models.McpCatalogSourceList, error)
}
```

### Implementation

```go
type ModelCatalogClient struct {
    logger *slog.Logger
}

func NewModelCatalogClient(logger *slog.Logger) (*ModelCatalogClient, error) {
    return &ModelCatalogClient{logger: logger}, nil
}

func (c *ModelCatalogClient) GetAllMcpServers(client HTTPClientInterface, params url.Values) (*models.McpServerList, error) {
    path := "/api/model_catalog/v1alpha1/mcp_servers"
    if len(params) > 0 {
        path = path + "?" + params.Encode()
    }

    resp, err := client.GET(path)
    if err != nil {
        return nil, fmt.Errorf("failed to get MCP servers: %w", err)
    }

    var result models.McpServerList
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal MCP servers: %w", err)
    }

    return &result, nil
}

func (c *ModelCatalogClient) GetMcpServer(client HTTPClientInterface, serverId string) (*models.McpServer, error) {
    path := fmt.Sprintf("/api/model_catalog/v1alpha1/mcp_servers/%s", url.PathEscape(serverId))

    resp, err := client.GET(path)
    if err != nil {
        if strings.Contains(err.Error(), "404") {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to get MCP server: %w", err)
    }

    var result models.McpServer
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal MCP server: %w", err)
    }

    return &result, nil
}
```

## Kubernetes Repositories

### User Repository

```go
// internal/repositories/user.go
type UserRepository struct{}

func (r *UserRepository) GetUser(ctx context.Context, factory k8s.KubernetesClientFactory) (*models.User, error) {
    client, err := factory.GetKubernetesClient()
    if err != nil {
        return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
    }

    // Extract user info from context (set by identity middleware)
    userInfo, ok := ctx.Value(constants.UserInfoContextKey).(*k8s.UserInfo)
    if !ok {
        return nil, errors.New("user info not found in context")
    }

    // Get user's namespaces via SelfSubjectAccessReview
    namespaces, err := client.GetAccessibleNamespaces(ctx, userInfo)
    if err != nil {
        return nil, fmt.Errorf("failed to get accessible namespaces: %w", err)
    }

    return &models.User{
        Username:   userInfo.Username,
        Groups:     userInfo.Groups,
        Namespaces: namespaces,
    }, nil
}
```

### Model Registry Repository

```go
// internal/repositories/model_registry.go
type ModelRegistryRepository struct{}

func (r *ModelRegistryRepository) GetAllModelRegistries(
    ctx context.Context,
    factory k8s.KubernetesClientFactory,
    namespace string,
) (*models.ModelRegistryList, error) {
    client, err := factory.GetKubernetesClient()
    if err != nil {
        return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
    }

    // List ModelRegistry custom resources
    registries, err := client.ListModelRegistries(ctx, namespace)
    if err != nil {
        return nil, fmt.Errorf("failed to list model registries: %w", err)
    }

    // Convert to API models
    items := make([]models.ModelRegistry, 0, len(registries.Items))
    for _, r := range registries.Items {
        items = append(items, models.ModelRegistry{
            Name:        r.Name,
            Namespace:   r.Namespace,
            DisplayName: r.Spec.DisplayName,
            Status:      string(r.Status.Phase),
        })
    }

    return &models.ModelRegistryList{Items: items}, nil
}
```

### Namespace Repository

```go
// internal/repositories/namespace.go
type NamespaceRepository struct{}

func (r *NamespaceRepository) GetAllNamespaces(
    ctx context.Context,
    factory k8s.KubernetesClientFactory,
) (*models.NamespaceList, error) {
    client, err := factory.GetKubernetesClient()
    if err != nil {
        return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
    }

    namespaces, err := client.ListNamespaces(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list namespaces: %w", err)
    }

    items := make([]models.Namespace, 0, len(namespaces.Items))
    for _, ns := range namespaces.Items {
        items = append(items, models.Namespace{
            Name: ns.Name,
        })
    }

    return &models.NamespaceList{Items: items}, nil
}
```

## HTTP Client Interface

```go
// internal/integrations/httpclient/http.go
type HTTPClientInterface interface {
    GET(path string) ([]byte, error)
    POST(path string, body []byte) ([]byte, error)
    PATCH(path string, body []byte) ([]byte, error)
    DELETE(path string) error
}

type HTTPClient struct {
    baseURL string
    client  *http.Client
    headers map[string]string
}

func NewHTTPClient(baseURL string, rootCAs *x509.CertPool) *HTTPClient {
    transport := &http.Transport{}
    if rootCAs != nil {
        transport.TLSClientConfig = &tls.Config{RootCAs: rootCAs}
    }

    return &HTTPClient{
        baseURL: baseURL,
        client:  &http.Client{Transport: transport, Timeout: 30 * time.Second},
        headers: make(map[string]string),
    }
}

func (c *HTTPClient) GET(path string) ([]byte, error) {
    req, err := http.NewRequest("GET", c.baseURL+path, nil)
    if err != nil {
        return nil, err
    }

    for k, v := range c.headers {
        req.Header.Set(k, v)
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    return io.ReadAll(resp.Body)
}
```

## Mock Repositories

```go
// internal/mocks/model_registry_client_mock.go
type ModelRegistryClientMock struct {
    logger *slog.Logger
    data   *StaticDataMock
}

func NewModelRegistryClient(logger *slog.Logger) (*ModelRegistryClientMock, error) {
    return &ModelRegistryClientMock{
        logger: logger,
        data:   NewStaticDataMock(),
    }, nil
}

func (m *ModelRegistryClientMock) GetAllRegisteredModels(client HTTPClientInterface, params url.Values) (*models.RegisteredModelList, error) {
    return m.data.RegisteredModels, nil
}
```

---

[Back to BFF Index](./README.md) | [Previous: Handlers](./handlers.md) | [Next: Kubernetes Integration](./kubernetes-integration.md)
