# Middleware Layer

This document covers the middleware components including validation, routing, health checks, and CORS.

## Overview

**Location:**
- `internal/server/middleware/` - Validation and routing
- `internal/proxy/` - Dynamic router and health checks
- `internal/server/openapi/routers.go` - CORS configuration

## Middleware Stack

```
HTTP Request
    │
    ▼
┌─────────────────────────────────────────────────┐
│              CORS Middleware                     │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│           Logging Middleware                     │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│         Validation Middleware                    │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│           Dynamic Router                         │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│          OpenAPI Controllers                     │
└─────────────────────────────────────────────────┘
```

## Validation Middleware

### Purpose

Prevents malicious input from reaching the database by validating all string parameters.

### Implementation

```go
// internal/server/middleware/validation.go
func ValidationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate query parameters
        for key, values := range r.URL.Query() {
            for _, value := range values {
                if err := validateStringParameter(key, value); err != nil {
                    http.Error(w, err.Error(), http.StatusBadRequest)
                    return
                }
            }
        }

        // Validate request body if present
        if r.Body != nil && r.ContentLength > 0 {
            body, err := io.ReadAll(r.Body)
            if err != nil {
                http.Error(w, "Failed to read request body", http.StatusBadRequest)
                return
            }

            if err := validateRequestBody(body); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }

            // Restore body for downstream handlers
            r.Body = io.NopCloser(bytes.NewBuffer(body))
        }

        next.ServeHTTP(w, r)
    })
}
```

### Null Byte Prevention

```go
func validateStringParameter(paramName, paramValue string) error {
    // Check for null bytes (SQL injection prevention)
    if strings.Contains(paramValue, "\x00") {
        return fmt.Errorf("parameter %s contains invalid null byte", paramName)
    }

    // Check for unicode null
    if strings.Contains(paramValue, "\\u0000") {
        return fmt.Errorf("parameter %s contains invalid unicode null", paramName)
    }

    return nil
}

func validateRequestBody(body []byte) error {
    // Check for null bytes in body
    if bytes.Contains(body, []byte{0x00}) {
        return errors.New("request body contains invalid null byte")
    }

    return nil
}
```

### Router Wrapping

```go
// internal/server/middleware/router.go
func WrapWithValidation(routers ...openapi.Router) http.Handler {
    baseRouter := openapi.NewRouter(routers...)
    return ValidationMiddleware(baseRouter)
}
```

## Dynamic Router

### Purpose

Allows swapping the HTTP handler at runtime, enabling graceful service initialization.

### Implementation

```go
// internal/proxy/dynamic_router.go
type dynamicRouter struct {
    mu     sync.RWMutex
    router http.Handler
}

func NewDynamicRouter() *dynamicRouter {
    return &dynamicRouter{
        router: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Service initializing..."))
        }),
    }
}

func (d *dynamicRouter) SetRouter(router http.Handler) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.router = router
}

func (d *dynamicRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    d.mu.RLock()
    router := d.router
    d.mu.RUnlock()
    router.ServeHTTP(w, r)
}
```

### Usage Pattern

```go
// cmd/proxy.go
func runProxy(cfg *ProxyConfig) error {
    // Create dynamic router (returns 503 initially)
    router := proxy.NewDynamicRouter()

    // Start server immediately
    go func() {
        http.ListenAndServe(address, router)
    }()

    // Initialize service in background
    go func() {
        service, err := newModelRegistryService(cfg)
        if err != nil {
            return
        }

        // Swap to real router when ready
        newRouter := middleware.WrapWithValidation(
            openapi.NewModelRegistryServiceAPIController(service),
        )
        router.SetRouter(newRouter)
    }()

    return nil
}
```

## Health Check System

### HealthChecker Interface

```go
// internal/proxy/readiness.go
type HealthChecker interface {
    Check() HealthCheck
}

type HealthCheck struct {
    Name    string                 `json:"name"`
    Status  string                 `json:"status"`  // "pass" or "fail"
    Message string                 `json:"message,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

### Database Health Checker

```go
type DatabaseHealthChecker struct {
    connector func() (*db.Connector, error)
}

func (d *DatabaseHealthChecker) Check() HealthCheck {
    check := HealthCheck{
        Name:   "database",
        Status: "fail",
    }

    connector, err := d.connector()
    if err != nil {
        check.Message = err.Error()
        return check
    }

    // Test connection with query
    sqlDB, err := connector.DB().DB()
    if err != nil {
        check.Message = err.Error()
        return check
    }

    if err := sqlDB.Ping(); err != nil {
        check.Message = err.Error()
        return check
    }

    // Check migration status
    var version int
    var dirty bool
    sqlDB.QueryRow("SELECT version, dirty FROM schema_migrations LIMIT 1").
        Scan(&version, &dirty)

    check.Status = "pass"
    check.Details = map[string]interface{}{
        "schemaVersion": version,
        "dirty":         dirty,
    }

    return check
}
```

### Model Registry Health Checker

```go
type ModelRegistryHealthChecker struct {
    serviceHolder *ModelRegistryServiceHolder
}

func (m *ModelRegistryHealthChecker) Check() HealthCheck {
    check := HealthCheck{
        Name:    "model_registry",
        Status:  "fail",
        Details: make(map[string]interface{}),
    }

    service := m.serviceHolder.Get()
    if service == nil {
        check.Message = "service not initialized"
        return check
    }

    // Test basic operations
    entities := []struct {
        name   string
        listFn func() (int, error)
    }{
        {"registered_models", func() (int, error) {
            list, err := service.GetRegisteredModels(api.ListOptions{})
            return int(list.Size), err
        }},
        {"model_versions", func() (int, error) {
            list, err := service.GetModelVersions(api.ListOptions{}, nil)
            return int(list.Size), err
        }},
        // ... more entities
    }

    for _, entity := range entities {
        count, err := entity.listFn()
        if err != nil {
            check.Details[entity.name] = map[string]interface{}{
                "status": "fail",
                "error":  err.Error(),
            }
        } else {
            check.Details[entity.name] = map[string]interface{}{
                "status": "pass",
                "count":  count,
            }
        }
    }

    check.Status = "pass"
    return check
}
```

### Conditional Health Checker

For services that may not be initialized yet:

```go
type ConditionalModelRegistryHealthChecker struct {
    serviceHolder *ModelRegistryServiceHolder
    delegate      *ModelRegistryHealthChecker
}

func (c *ConditionalModelRegistryHealthChecker) Check() HealthCheck {
    if c.serviceHolder.Get() == nil {
        return HealthCheck{
            Name:    "model_registry",
            Status:  "fail",
            Message: "service not yet initialized",
        }
    }

    return c.delegate.Check()
}
```

### Readiness Handler

```go
func GeneralReadinessHandler(additionalCheckers ...HealthChecker) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        allChecks := make([]HealthCheck, 0)
        overallStatus := "pass"

        for _, checker := range additionalCheckers {
            check := checker.Check()
            allChecks = append(allChecks, check)

            if check.Status != "pass" {
                overallStatus = "fail"
            }
        }

        response := map[string]interface{}{
            "status":   overallStatus,
            "checks":   allChecks,
            "duration": time.Since(start).String(),
        }

        w.Header().Set("Content-Type", "application/json")

        if overallStatus != "pass" {
            w.WriteHeader(http.StatusServiceUnavailable)
        }

        json.NewEncoder(w).Encode(response)
    })
}
```

## CORS Configuration

### Setup

```go
// internal/server/openapi/routers.go
func NewRouter(routers ...Router) chi.Router {
    router := chi.NewRouter()

    // CORS middleware
    router.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{"https://*", "http://*"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{
            "Accept",
            "Authorization",
            "Content-Type",
            "X-CSRF-Token",
            "X-Requested-With",
        },
        ExposedHeaders:   []string{"Link"},
        AllowCredentials: false,
        MaxAge:           300, // 5 minutes
    }))

    // Logging middleware
    router.Use(Logger)

    // Mount routes
    for _, r := range routers {
        for _, route := range r.Routes() {
            router.Method(route.Method, route.Pattern, route.HandlerFunc)
        }
    }

    return router
}
```

## Logging Middleware

```go
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap response writer to capture status code
        ww := &responseWriter{ResponseWriter: w}

        next.ServeHTTP(ww, r)

        log.Printf("%s %s %d %s",
            r.Method,
            r.URL.Path,
            ww.statusCode,
            time.Since(start),
        )
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
    w.statusCode = code
    w.ResponseWriter.WriteHeader(code)
}
```

## Health Endpoints

```go
// cmd/proxy.go
func setupHealthEndpoints(mux *http.ServeMux, serviceHolder *ModelRegistryServiceHolder) {
    // Database-only health (for init)
    mux.Handle("/isDirty", GeneralReadinessHandler(
        &DatabaseHealthChecker{},
    ))

    // Full health check
    mux.Handle("/health", GeneralReadinessHandler(
        &DatabaseHealthChecker{},
        &ConditionalModelRegistryHealthChecker{
            serviceHolder: serviceHolder,
        },
    ))

    // Liveness (always pass if server is running)
    mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // Readiness
    mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
        if serviceHolder.Get() == nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Service not ready"))
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
}
```

---

[Back to Backend Index](./README.md) | [Previous: Converter/Mapper](./converter-mapper.md) | [Next: Configuration](./configuration.md)
