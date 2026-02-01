# Security Analysis

This document analyzes security considerations in the Kubeflow Model Registry codebase.

## Overview

The security posture of the Model Registry is generally good, with standard security practices implemented throughout. No critical vulnerabilities were identified.

## Authentication

### Kubernetes Integration

**Implementation**: `clients/ui/bff/internal/integrations/kubernetes/`

The BFF supports two authentication modes:

1. **Internal Mode** (default):
   - Uses service account credentials
   - User identity passed via headers (`kubeflow-userid`, `kubeflow-groups`)
   - Suitable for Kubeflow dashboard integration

2. **User Token Mode**:
   - Uses Bearer token from `Authorization` header
   - Supports OIDC flows
   - Configurable header/prefix

**Assessment**: Flexible authentication supporting multiple deployment scenarios.

### Token Handling

```go
func (f *kubernetesClientFactory) GetKubernetesClientForToken(token string) (KubernetesClient, error) {
    configCopy := rest.CopyConfig(f.config)
    configCopy.BearerToken = token
    configCopy.BearerTokenFile = ""  // Clear file to use token
    // ...
}
```

**Strengths**:
- Tokens not logged
- Config copied to avoid mutation
- File path cleared when using direct token

**Recommendations**:
- Consider token validation before use
- Add token expiry handling

## Authorization

### RBAC Integration

**Implementation**: Uses Kubernetes RBAC via SelfSubjectAccessReview

```go
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
    // ...
}
```

**Strengths**:
- Leverages Kubernetes native RBAC
- Fine-grained resource-level checks
- Consistent authorization model

**Assessment**: Good integration with Kubernetes security model.

### Middleware Authorization

```go
func (app *App) RequireAccessToMRService(next httprouter.Handle) httprouter.Handle {
    return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        // Extract context values
        namespace := r.Context().Value(constants.NamespaceContextKey).(string)
        registryName := ps.ByName(ModelRegistryId)

        // Check access
        allowed, err := client.CanAccessModelRegistry(r.Context(), namespace, registryName, "get")
        if !allowed {
            app.forbiddenResponse(w, r, "access denied to model registry")
            return
        }
        // ...
    }
}
```

**Assessment**: Authorization consistently applied via middleware.

## Input Validation

### API Input Validation

**Implementation**: `internal/core/middleware/validation/`

```go
func NullByteCheck(s string) error {
    if strings.Contains(s, "\x00") {
        return ErrNullByte
    }
    return nil
}
```

**Strengths**:
- Null byte injection prevention
- Applied at middleware level
- Consistent across endpoints

**Gaps**:
- Limited schema validation for complex inputs
- MCP YAML content not fully validated

### Request Body Limits

```go
func (app *App) ReadJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
    maxBytes := 1_048_576 // 1MB
    r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
    // ...
}
```

**Assessment**: Body size limits prevent DoS via large payloads.

### JSON Parsing Security

```go
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()  // Reject unknown fields
```

**Assessment**: Strict JSON parsing prevents injection of unexpected fields.

## SQL Injection Prevention

### GORM Usage

The project uses GORM with parameterized queries:

```go
func (r *GenericRepository[E, T]) Get(id string) (*T, error) {
    var entity E
    result := r.db.First(&entity, "id = ?", id)  // Parameterized
    // ...
}
```

**Assessment**: SQL injection prevented through ORM usage.

### Raw Query Concerns

Some filter expressions use string building:

```go
// Filter parsing uses safe query builder
query = query.Where(condition.Field, condition.Value)
```

**Assessment**: Filter system uses query builder, not raw string concatenation.

## Cross-Site Scripting (XSS) Prevention

### Frontend Practices

React's JSX automatically escapes output:

```tsx
// Safe: React escapes content
<div>{model.description}</div>

// Dangerous: Should be avoided
<div dangerouslySetInnerHTML={{__html: content}} />
```

**Assessment**: Standard React patterns prevent XSS. No uses of `dangerouslySetInnerHTML` found.

### Content-Type Headers

```go
w.Header().Set("Content-Type", "application/json")
```

**Assessment**: Proper content types set for responses.

## CORS Configuration

### Default: Disabled

```go
// CORS disabled by default for security
if len(allowedOrigins) == 0 {
    // No CORS headers added
}
```

### Configurable Origins

```go
// Can be configured via environment
ALLOWED_ORIGINS="http://example.com,http://other.com"

// Or wildcard (not recommended for production)
ALLOWED_ORIGINS="*"
```

**Assessment**: Secure defaults with configurable CORS when needed.

## TLS Configuration

### Service-to-Service TLS

```go
func NewHTTPClient(baseURL string, rootCAs *x509.CertPool) *HTTPClient {
    transport := &http.Transport{}
    if rootCAs != nil {
        transport.TLSClientConfig = &tls.Config{RootCAs: rootCAs}
    }
    // ...
}
```

### Skip TLS Verification Option

```go
// For development only
INSECURE_SKIP_VERIFY=true
```

**Warning**: This option should only be used in development.

**Assessment**: TLS properly implemented with escape hatch for development.

## Secret Management

### Environment Variables

Secrets are passed via environment variables:
- `MYSQL_ROOT_PASSWORD`
- `POSTGRES_PASSWORD`
- Database credentials

**Assessment**: Standard Kubernetes secret management patterns.

### No Hardcoded Secrets

No hardcoded secrets found in source code. Sample/test configurations use placeholder values.

## Container Security

### Non-Root Execution

```dockerfile
USER 65532:65532
```

### Minimal Images

```dockerfile
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5
```

### Security Context

```yaml
securityContext:
  runAsNonRoot: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

**Assessment**: Good container security practices.

## Dependency Security

### Dependabot

The project uses Dependabot for automated dependency updates.

### Go Modules

`go.mod` and `go.sum` lock dependency versions.

### NPM Lockfile

`package-lock.json` locks JavaScript dependencies.

**Assessment**: Standard dependency management practices.

## Security Recommendations

### High Priority

1. **Add MCP YAML Schema Validation**
   - Validate structure before processing
   - Reject malformed configurations early

2. **Token Expiry Handling**
   - Refresh tokens before expiry
   - Handle expired token gracefully

### Medium Priority

3. **Rate Limiting**
   - Add rate limiting to API endpoints
   - Prevent brute force attacks

4. **Audit Logging**
   - Log security-relevant events
   - Include user identity in logs

5. **Input Validation Enhancement**
   - Add comprehensive validation for MCP definitions
   - Validate URL formats for endpoints

### Low Priority

6. **Security Headers**
   - Add standard security headers (CSP, X-Frame-Options, etc.)
   - Harden default response headers

7. **Secrets Rotation**
   - Document secrets rotation procedure
   - Consider Vault integration

## Compliance Considerations

### OWASP Top 10

| Vulnerability | Status |
|---------------|--------|
| Injection | Mitigated (ORM, parameterized queries) |
| Broken Authentication | Addressed (K8s RBAC) |
| Sensitive Data Exposure | Addressed (TLS, no secrets in code) |
| XML External Entities | N/A (JSON only) |
| Broken Access Control | Addressed (RBAC middleware) |
| Security Misconfiguration | Needs attention (documentation) |
| Cross-Site Scripting | Addressed (React escaping) |
| Insecure Deserialization | Low risk (typed JSON parsing) |
| Known Vulnerabilities | Addressed (Dependabot) |
| Insufficient Logging | Needs improvement |

## Conclusion

The Kubeflow Model Registry has a solid security foundation:

- **Strengths**: Kubernetes RBAC integration, TLS support, container security, input validation
- **Gaps**: Limited audit logging, missing rate limiting, incomplete input validation for MCP

Overall security posture: **Good** with recommendations for improvement.

---

[Back to Code Review Index](./README.md) | [Previous: Architecture Observations](./architecture-observations.md) | [Next: Testing Coverage](./testing-coverage.md)
