# BFF Handlers

This document covers the API handler patterns in the BFF.

## Handler Pattern

All handlers follow a consistent pattern:

```go
func (app *App) HandlerName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    // 1. Get client from context
    client, ok := r.Context().Value(constants.ClientKey).(ClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("client not found"))
        return
    }

    // 2. Extract parameters
    param := ps.ByName("param_name")

    // 3. Call repository
    data, err := app.repositories.Method(client, params)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    // 4. Handle not found
    if data == nil {
        app.notFoundResponse(w, r)
        return
    }

    // 5. Return response
    response := Envelope{Data: data}
    err = app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

## Response Envelopes

```go
// Generic envelope types
type None = map[string]interface{}

type Envelope[TData any, TError any] struct {
    Data  TData  `json:"data,omitempty"`
    Error TError `json:"error,omitempty"`
}

// Specific envelopes
type RegisteredModelListEnvelope Envelope[*models.RegisteredModelList, None]
type RegisteredModelEnvelope Envelope[*models.RegisteredModel, None]
type ModelVersionListEnvelope Envelope[*models.ModelVersionList, None]
type McpServerListEnvelope Envelope[*models.McpServerList, None]
```

## Model Registry Handlers

### GetAllRegisteredModelsHandler

```go
// internal/api/registered_models_handler.go
func (app *App) GetAllRegisteredModelsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("model registry REST client not found"))
        return
    }

    registeredModels, err := app.repositories.ModelRegistryClient.GetAllRegisteredModels(client, r.URL.Query())
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    modelList := RegisteredModelListEnvelope{
        Data: registeredModels,
    }

    err = app.WriteJSON(w, http.StatusOK, modelList, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### CreateRegisteredModelHandler

```go
func (app *App) CreateRegisteredModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("model registry REST client not found"))
        return
    }

    // Parse request body
    var input models.RegisteredModelCreate
    err := app.ReadJSON(w, r, &input)
    if err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    // Validate input
    if err := validation.ValidateRegisteredModel(&input); err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    // Create model
    registeredModel, err := app.repositories.ModelRegistryClient.CreateRegisteredModel(client, input)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := RegisteredModelEnvelope{
        Data: registeredModel,
    }

    err = app.WriteJSON(w, http.StatusCreated, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### UpdateRegisteredModelHandler

```go
func (app *App) UpdateRegisteredModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("model registry REST client not found"))
        return
    }

    modelId := ps.ByName(RegisteredModelId)

    var input models.RegisteredModelUpdate
    err := app.ReadJSON(w, r, &input)
    if err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    registeredModel, err := app.repositories.ModelRegistryClient.UpdateRegisteredModel(client, modelId, input)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := RegisteredModelEnvelope{
        Data: registeredModel,
    }

    err = app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

## Model Catalog Handlers

### GetAllCatalogModelsAcrossSourcesHandler

```go
// internal/api/catalog_models_handler.go
func (app *App) GetAllCatalogModelsAcrossSourcesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
        return
    }

    catalogModels, err := app.repositories.ModelCatalogClient.GetAllCatalogModels(client, r.URL.Query())
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := CatalogModelListEnvelope{
        Data: catalogModels,
    }

    err = app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### GetCatalogFilterListHandler

```go
func (app *App) GetCatalogFilterListHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
        return
    }

    filterOptions, err := app.repositories.ModelCatalogClient.GetFilterOptions(client)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := FilterOptionsEnvelope{
        Data: filterOptions,
    }

    err = app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

## MCP Catalog Handlers

### GetAllMcpServersHandler

```go
// internal/api/mcp_server_handler.go
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

    serverList := McpServerListEnvelope{
        Data: mcpServers,
    }

    err = app.WriteJSON(w, http.StatusOK, serverList, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### GetMcpServerHandler

```go
func (app *App) GetMcpServerHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
        return
    }

    serverId := ps.ByName(McpServerId)

    mcpServer, err := app.repositories.ModelCatalogClient.GetMcpServer(client, serverId)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    if mcpServer == nil {
        app.notFoundResponse(w, r)
        return
    }

    serverEnvelope := McpServerEnvelope{
        Data: mcpServer,
    }

    err = app.WriteJSON(w, http.StatusOK, serverEnvelope, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

## Kubernetes Handlers

### UserHandler

```go
// internal/api/user_handler.go
func (app *App) UserHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    user, err := app.repositories.User.GetUser(r.Context(), app.kubernetesClientFactory)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := UserEnvelope{
        Data: user,
    }

    err = app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### GetAllModelRegistriesHandler

```go
// internal/api/model_registry_handler.go
func (app *App) GetAllModelRegistriesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    namespace, ok := r.Context().Value(constants.NamespaceContextKey).(string)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("namespace not found in context"))
        return
    }

    registries, err := app.repositories.ModelRegistry.GetAllModelRegistries(
        r.Context(),
        app.kubernetesClientFactory,
        namespace,
    )
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := ModelRegistryListEnvelope{
        Data: registries,
    }

    err = app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

### HealthcheckHandler

```go
// internal/api/healthcheck_handler.go
func (app *App) HealthcheckHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    health := app.repositories.HealthCheck.HealthCheck(r.Context(), app.kubernetesClientFactory)

    response := HealthCheckEnvelope{
        Data: health,
    }

    err := app.WriteJSON(w, http.StatusOK, response, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
```

## JSON Helpers

```go
// internal/api/helpers.go
func (app *App) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers http.Header) error {
    js, err := json.Marshal(data)
    if err != nil {
        return err
    }

    js = append(js, '\n')

    for key, value := range headers {
        w.Header()[key] = value
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    w.Write(js)

    return nil
}

func (app *App) ReadJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
    maxBytes := 1_048_576 // 1MB
    r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    err := dec.Decode(dst)
    if err != nil {
        // Handle specific JSON errors with better messages
        var syntaxError *json.SyntaxError
        var unmarshalTypeError *json.UnmarshalTypeError
        var maxBytesError *http.MaxBytesError

        switch {
        case errors.As(err, &syntaxError):
            return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
        case errors.As(err, &unmarshalTypeError):
            return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
        case errors.As(err, &maxBytesError):
            return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
        default:
            return err
        }
    }

    return nil
}
```

---

[Back to BFF Index](./README.md) | [Previous: Architecture](./architecture.md) | [Next: Repositories](./repositories.md)
