# BFF Architecture

This document covers the architecture and design of the BFF layer.

## Overview

The BFF (Backend for Frontend) serves as an API gateway between the React frontend and backend services.

```
┌─────────────────────────────────────────────────────────────────┐
│                         BFF Architecture                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐                                                │
│  │   Frontend   │                                                │
│  │  (React SPA) │                                                │
│  └──────┬───────┘                                                │
│         │ HTTP                                                   │
│         ▼                                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                        BFF Server                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │   │
│  │  │  Middleware  │  │   Handlers   │  │ Repositories │    │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘    │   │
│  │         │                 │                  │            │   │
│  │         └────────┬────────┴──────────────────┘            │   │
│  │                  │                                         │   │
│  │  ┌──────────────┴──────────────┐                          │   │
│  │  │    Kubernetes Integration   │                          │   │
│  │  └─────────────────────────────┘                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                    │
│                              ▼                                    │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐        │
│  │ Model Registry│  │ Model Catalog │  │  Kubernetes   │        │
│  │     API       │  │     API       │  │     API       │        │
│  └───────────────┘  └───────────────┘  └───────────────┘        │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### App Structure

```go
// internal/api/app.go
type App struct {
    config                  config.EnvConfig
    logger                  *slog.Logger
    kubernetesClientFactory k8s.KubernetesClientFactory
    repositories            *repositories.Repositories
    testEnv                 *envtest.Environment  // For mock mode
    rootCAs                 *x509.CertPool        // For TLS
}
```

### Initialization Flow

```go
func NewApp(cfg config.EnvConfig, logger *slog.Logger) (*App, error) {
    // 1. Initialize CA pool for TLS
    if len(cfg.BundlePaths) > 0 {
        rootCAs = loadCACertificates(cfg.BundlePaths)
    }

    // 2. Create Kubernetes client factory
    if cfg.MockK8Client {
        k8sFactory, testEnv = k8mocks.NewMockedKubernetesClientFactory(...)
    } else {
        k8sFactory = k8s.NewKubernetesClientFactory(cfg, logger)
    }

    // 3. Create Model Registry client
    if cfg.MockMRClient {
        mrClient = mocks.NewModelRegistryClient(logger)
    } else {
        mrClient = repositories.NewModelRegistryClient(logger)
    }

    // 4. Create Model Catalog client
    if cfg.MockMRCatalogClient {
        modelCatalogClient = mocks.NewModelCatalogClientMock(logger)
    } else {
        modelCatalogClient = repositories.NewModelCatalogClient(logger)
    }

    // 5. Create repositories
    repositories := repositories.NewRepositories(mrClient, modelCatalogClient)

    return &App{
        config:                  cfg,
        logger:                  logger,
        kubernetesClientFactory: k8sFactory,
        repositories:            repositories,
        rootCAs:                 rootCAs,
    }, nil
}
```

## Request Flow

### Middleware Chain

```
Request → RecoverPanic → EnableTelemetry → EnableCORS → InjectRequestIdentity
                                                               │
                                                               ▼
                                                        AttachNamespace
                                                               │
                                                               ▼
                                                    RequireAccessToMRService
                                                               │
                                                               ▼
                                                  AttachModelRegistryRESTClient
                                                               │
                                                               ▼
                                                           Handler
                                                               │
                                                               ▼
                                                           Response
```

### Middleware Functions

```go
// Panic recovery
func (app *App) RecoverPanic(next http.Handler) http.Handler

// Telemetry and logging
func (app *App) EnableTelemetry(next http.Handler) http.Handler

// CORS headers
func (app *App) EnableCORS(next http.Handler) http.Handler

// Extract user identity from request
func (app *App) InjectRequestIdentity(next http.Handler) http.Handler

// Attach namespace from header to context
func (app *App) AttachNamespace(next httprouter.Handle) httprouter.Handle

// RBAC check for Model Registry access
func (app *App) RequireAccessToMRService(next httprouter.Handle) httprouter.Handle

// Attach HTTP client for Model Registry API
func (app *App) AttachModelRegistryRESTClient(next httprouter.Handle) httprouter.Handle

// Attach HTTP client for Model Catalog API
func (app *App) AttachModelCatalogRESTClient(next httprouter.Handle) httprouter.Handle
```

## Route Registration

```go
func (app *App) Routes() http.Handler {
    // API router
    apiRouter := httprouter.New()
    apiRouter.NotFound = http.HandlerFunc(app.notFoundResponse)
    apiRouter.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

    // Model Registry routes
    apiRouter.GET(RegisteredModelListPath,
        app.AttachNamespace(
            app.RequireAccessToMRService(
                app.AttachModelRegistryRESTClient(
                    app.GetAllRegisteredModelsHandler))))

    // Model Catalog routes
    apiRouter.GET(CatalogModelListPath,
        app.AttachNamespace(
            app.AttachModelCatalogRESTClient(
                app.GetAllCatalogModelsAcrossSourcesHandler)))

    // MCP Catalog routes
    apiRouter.GET(McpServerListPath,
        app.AttachNamespace(
            app.AttachModelCatalogRESTClient(
                app.GetAllMcpServersHandler)))

    // Kubernetes routes
    apiRouter.GET(UserPath, app.UserHandler)
    apiRouter.GET(ModelRegistryListPath,
        app.AttachNamespace(
            app.RequireListServiceAccessInNamespace(
                app.GetAllModelRegistriesHandler)))

    // Combine with static file server
    appMux := http.NewServeMux()
    appMux.Handle(ApiPathPrefix+"/", apiRouter)

    // Static files and SPA fallback
    staticDir := http.Dir(app.config.StaticAssetsDir)
    appMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if _, err := staticDir.Open(r.URL.Path); err == nil {
            http.FileServer(staticDir).ServeHTTP(w, r)
            return
        }
        http.ServeFile(w, r, path.Join(app.config.StaticAssetsDir, "index.html"))
    })

    // Health check endpoint
    combinedMux := http.NewServeMux()
    combinedMux.Handle(HealthCheckPath, healthcheckMux)
    combinedMux.Handle("/", app.RecoverPanic(app.EnableTelemetry(app.EnableCORS(app.InjectRequestIdentity(appMux)))))

    return combinedMux
}
```

## Configuration

### Environment Configuration

```go
// internal/config/environment.go
type EnvConfig struct {
    Port                  int
    DevMode               bool
    StaticAssetsDir       string
    DeploymentMode        DeploymentMode
    MockK8Client          bool
    MockMRClient          bool
    MockMRCatalogClient   bool
    ModelRegistryBaseURL  string
    ModelCatalogBaseURL   string
    BundlePaths           []string
}
```

### Deployment Modes

```go
type DeploymentMode string

const (
    DeploymentModeKubeflow   DeploymentMode = "kubeflow"
    DeploymentModeStandalone DeploymentMode = "standalone"
    DeploymentModeFederated  DeploymentMode = "federated"
)

func (dm DeploymentMode) IsKubeflowMode() bool {
    return dm == DeploymentModeKubeflow
}
```

## Error Handling

### Error Response Types

```go
// internal/api/errors.go
func (app *App) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
    env := envelope{"error": message}
    err := app.WriteJSON(w, status, env, nil)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
    }
}

func (app *App) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
    app.logError(r, err)
    message := "the server encountered a problem and could not process your request"
    app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *App) notFoundResponse(w http.ResponseWriter, r *http.Request) {
    message := "the requested resource could not be found"
    app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *App) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
    message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
    app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (app *App) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
    app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *App) forbiddenResponse(w http.ResponseWriter, r *http.Request, message string) {
    app.errorResponse(w, r, http.StatusForbidden, message)
}
```

## Testing

### Mock Support

The BFF supports mock mode for development and testing:

```go
// Start with mocks
export MOCK_K8S_CLIENT=true
export MOCK_MR_CLIENT=true
export MOCK_MR_CATALOG_CLIENT=true
go run cmd/main.go
```

### Test Structure

```go
// internal/api/suite_test.go
func TestHandlers(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "API Handlers Suite")
}

// Handler test example
var _ = Describe("GetAllRegisteredModelsHandler", func() {
    var app *App
    var recorder *httptest.ResponseRecorder

    BeforeEach(func() {
        app = setupTestApp()
        recorder = httptest.NewRecorder()
    })

    It("returns registered models", func() {
        req := httptest.NewRequest("GET", "/api/v1/model_registry/test/registered_models", nil)
        app.Routes().ServeHTTP(recorder, req)

        Expect(recorder.Code).To(Equal(http.StatusOK))
    })
})
```

---

[Back to BFF Index](./README.md) | [Next: Handlers](./handlers.md)
