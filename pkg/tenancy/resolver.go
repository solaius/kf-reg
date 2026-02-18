package tenancy

import (
	"fmt"
	"net/http"
	"regexp"
)

// maxNamespaceLen is the maximum length for a namespace, following K8s conventions.
const maxNamespaceLen = 63

// namespaceRe validates namespace format: lowercase alphanumeric and hyphens,
// must start and end with an alphanumeric character (K8s DNS label convention).
var namespaceRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// NamespaceQueryParam is the query parameter name used for namespace resolution.
const NamespaceQueryParam = "namespace"

// NamespaceHeader is the HTTP header used for namespace resolution.
const NamespaceHeader = "X-Namespace"

// TenantResolver resolves the tenant context from an HTTP request.
type TenantResolver interface {
	Resolve(r *http.Request) (TenantContext, error)
}

// SingleTenantResolver always returns the "default" namespace.
type SingleTenantResolver struct{}

// Resolve always returns a TenantContext with Namespace "default".
func (s SingleTenantResolver) Resolve(_ *http.Request) (TenantContext, error) {
	return TenantContext{Namespace: "default"}, nil
}

// NamespaceTenantResolver reads the namespace from the request query parameter
// or header. In multi-tenant mode namespace is always required.
type NamespaceTenantResolver struct{}

// Resolve extracts the namespace from the request. It checks the query parameter
// first, then falls back to the X-Namespace header. Returns an error if the
// namespace is missing or invalid.
func (n NamespaceTenantResolver) Resolve(r *http.Request) (TenantContext, error) {
	ns := r.URL.Query().Get(NamespaceQueryParam)
	if ns == "" {
		ns = r.Header.Get(NamespaceHeader)
	}

	if ns == "" {
		return TenantContext{}, fmt.Errorf("namespace is required in multi-tenant mode (use ?namespace= query param or X-Namespace header)")
	}

	if err := validateNamespace(ns); err != nil {
		return TenantContext{}, err
	}

	return TenantContext{Namespace: ns}, nil
}

// validateNamespace checks that a namespace string conforms to K8s DNS label rules:
// lowercase alphanumeric and hyphens, 1-63 characters, starts and ends with alphanumeric.
func validateNamespace(ns string) error {
	if len(ns) > maxNamespaceLen {
		return fmt.Errorf("namespace %q exceeds maximum length of %d characters", ns, maxNamespaceLen)
	}
	if !namespaceRe.MatchString(ns) {
		return fmt.Errorf("namespace %q is invalid: must consist of lowercase alphanumeric characters or hyphens, and must start and end with an alphanumeric character", ns)
	}
	return nil
}
