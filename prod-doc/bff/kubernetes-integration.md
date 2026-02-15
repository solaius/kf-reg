# Kubernetes Integration

This document covers the Kubernetes client integration in the BFF.

## Overview

The BFF integrates with Kubernetes for:

- User authentication and authorization
- Model Registry CRD management
- Namespace listing
- RBAC access control

## Client Factory

```go
// internal/integrations/kubernetes/factory.go
type KubernetesClientFactory interface {
    GetKubernetesClient() (KubernetesClient, error)
    GetKubernetesClientForToken(token string) (KubernetesClient, error)
}

type kubernetesClientFactory struct {
    config *rest.Config
    logger *slog.Logger
}

func NewKubernetesClientFactory(cfg config.EnvConfig, logger *slog.Logger) (KubernetesClientFactory, error) {
    // Get in-cluster config or kubeconfig
    config, err := getKubeConfig(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
    }

    return &kubernetesClientFactory{
        config: config,
        logger: logger,
    }, nil
}

func (f *kubernetesClientFactory) GetKubernetesClient() (KubernetesClient, error) {
    clientset, err := kubernetes.NewForConfig(f.config)
    if err != nil {
        return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
    }

    return NewKubernetesClient(clientset, f.logger), nil
}

func (f *kubernetesClientFactory) GetKubernetesClientForToken(token string) (KubernetesClient, error) {
    configCopy := rest.CopyConfig(f.config)
    configCopy.BearerToken = token
    configCopy.BearerTokenFile = ""

    clientset, err := kubernetes.NewForConfig(configCopy)
    if err != nil {
        return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
    }

    return NewKubernetesClient(clientset, f.logger), nil
}
```

## Kubernetes Client

```go
// internal/integrations/kubernetes/client.go
type KubernetesClient interface {
    // Namespaces
    ListNamespaces(ctx context.Context) (*v1.NamespaceList, error)
    GetAccessibleNamespaces(ctx context.Context, userInfo *UserInfo) ([]string, error)

    // Model Registries (CRD)
    ListModelRegistries(ctx context.Context, namespace string) (*ModelRegistryList, error)
    GetModelRegistry(ctx context.Context, namespace, name string) (*ModelRegistry, error)
    CreateModelRegistry(ctx context.Context, mr *ModelRegistry) (*ModelRegistry, error)
    UpdateModelRegistry(ctx context.Context, mr *ModelRegistry) (*ModelRegistry, error)
    DeleteModelRegistry(ctx context.Context, namespace, name string) error

    // RBAC
    CanAccessModelRegistry(ctx context.Context, namespace, name, verb string) (bool, error)
    CreateSelfSubjectAccessReview(ctx context.Context, ssar *authv1.SelfSubjectAccessReview) (*authv1.SelfSubjectAccessReview, error)

    // User info
    GetUserInfo(ctx context.Context) (*UserInfo, error)
}
```

## Implementation

```go
// internal/integrations/kubernetes/internal_k8s_client.go
type internalKubernetesClient struct {
    clientset kubernetes.Interface
    logger    *slog.Logger
}

func NewKubernetesClient(clientset kubernetes.Interface, logger *slog.Logger) KubernetesClient {
    return &internalKubernetesClient{
        clientset: clientset,
        logger:    logger,
    }
}

func (c *internalKubernetesClient) ListNamespaces(ctx context.Context) (*v1.NamespaceList, error) {
    return c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
}

func (c *internalKubernetesClient) GetAccessibleNamespaces(ctx context.Context, userInfo *UserInfo) ([]string, error) {
    namespaces, err := c.ListNamespaces(ctx)
    if err != nil {
        return nil, err
    }

    accessible := make([]string, 0)
    for _, ns := range namespaces.Items {
        // Check if user can access this namespace
        ssar := &authv1.SelfSubjectAccessReview{
            Spec: authv1.SelfSubjectAccessReviewSpec{
                ResourceAttributes: &authv1.ResourceAttributes{
                    Namespace: ns.Name,
                    Verb:      "get",
                    Resource:  "modelregistries",
                    Group:     "modelregistry.kubeflow.org",
                },
            },
        }

        result, err := c.CreateSelfSubjectAccessReview(ctx, ssar)
        if err != nil {
            c.logger.Warn("Failed to check access", slog.String("namespace", ns.Name), slog.Any("error", err))
            continue
        }

        if result.Status.Allowed {
            accessible = append(accessible, ns.Name)
        }
    }

    return accessible, nil
}

func (c *internalKubernetesClient) CanAccessModelRegistry(ctx context.Context, namespace, name, verb string) (bool, error) {
    ssar := &authv1.SelfSubjectAccessReview{
        Spec: authv1.SelfSubjectAccessReviewSpec{
            ResourceAttributes: &authv1.ResourceAttributes{
                Namespace: namespace,
                Name:      name,
                Verb:      verb,
                Resource:  "modelregistries",
                Group:     "modelregistry.kubeflow.org",
            },
        },
    }

    result, err := c.CreateSelfSubjectAccessReview(ctx, ssar)
    if err != nil {
        return false, err
    }

    return result.Status.Allowed, nil
}

func (c *internalKubernetesClient) CreateSelfSubjectAccessReview(ctx context.Context, ssar *authv1.SelfSubjectAccessReview) (*authv1.SelfSubjectAccessReview, error) {
    return c.clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
}
```

## User Info Extraction

```go
// internal/integrations/kubernetes/types.go
type UserInfo struct {
    Username string
    UID      string
    Groups   []string
}

// Extract from request headers (set by Kubernetes API server or auth proxy)
func ExtractUserInfo(r *http.Request) *UserInfo {
    return &UserInfo{
        Username: r.Header.Get("X-Forwarded-User"),
        UID:      r.Header.Get("X-Forwarded-Uid"),
        Groups:   strings.Split(r.Header.Get("X-Forwarded-Groups"), ","),
    }
}
```

## Middleware Integration

```go
// internal/api/middleware.go

// InjectRequestIdentity extracts user identity from request headers
func (app *App) InjectRequestIdentity(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userInfo := k8s.ExtractUserInfo(r)
        ctx := context.WithValue(r.Context(), constants.UserInfoContextKey, userInfo)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// RequireAccessToMRService checks RBAC before allowing access
func (app *App) RequireAccessToMRService(next httprouter.Handle) httprouter.Handle {
    return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        namespace := r.Context().Value(constants.NamespaceContextKey).(string)
        registryName := ps.ByName(ModelRegistryId)

        client, err := app.kubernetesClientFactory.GetKubernetesClient()
        if err != nil {
            app.serverErrorResponse(w, r, err)
            return
        }

        allowed, err := client.CanAccessModelRegistry(r.Context(), namespace, registryName, "get")
        if err != nil {
            app.serverErrorResponse(w, r, err)
            return
        }

        if !allowed {
            app.forbiddenResponse(w, r, "access denied to model registry")
            return
        }

        next(w, r, ps)
    }
}

// AttachNamespace extracts namespace from header and adds to context
func (app *App) AttachNamespace(next httprouter.Handle) httprouter.Handle {
    return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        namespace := r.Header.Get("kubeflow-namespace")
        if namespace == "" {
            namespace = r.Header.Get("namespace")
        }
        if namespace == "" {
            app.badRequestResponse(w, r, errors.New("namespace header required"))
            return
        }

        ctx := context.WithValue(r.Context(), constants.NamespaceContextKey, namespace)
        next(w, r.WithContext(ctx), ps)
    }
}
```

## Mock Kubernetes Client

For development and testing:

```go
// internal/integrations/kubernetes/k8mocks/mock_factory.go
type MockedKubernetesClientFactory struct {
    clientset kubernetes.Interface
    testEnv   *envtest.Environment
    logger    *slog.Logger
}

func NewMockedKubernetesClientFactory(
    clientset kubernetes.Interface,
    testEnv *envtest.Environment,
    cfg config.EnvConfig,
    logger *slog.Logger,
) (KubernetesClientFactory, error) {
    return &MockedKubernetesClientFactory{
        clientset: clientset,
        testEnv:   testEnv,
        logger:    logger,
    }, nil
}

func SetupEnvTest(input TestEnvInput) (*envtest.Environment, kubernetes.Interface, error) {
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{
            filepath.Join("..", "..", "..", "config", "crd", "bases"),
        },
    }

    cfg, err := testEnv.Start()
    if err != nil {
        return nil, nil, err
    }

    clientset, err := kubernetes.NewForConfig(cfg)
    if err != nil {
        return nil, nil, err
    }

    return testEnv, clientset, nil
}
```

## Configuration

Environment variables for Kubernetes integration:

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Path to kubeconfig file | In-cluster config |
| `MOCK_K8S_CLIENT` | Use mock client | `false` |
| `IN_CLUSTER` | Force in-cluster config | Auto-detected |

---

[Back to BFF Index](./README.md) | [Previous: Repositories](./repositories.md)
