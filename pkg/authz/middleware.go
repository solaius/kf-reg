package authz

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// RequirePermission returns middleware that enforces a specific resource/verb
// permission check. It retrieves the identity from context (via IdentityMiddleware)
// and the namespace from context (via tenancy middleware), then calls the authorizer.
func RequirePermission(authorizer Authorizer, resource, verb string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, _ := IdentityFromContext(r.Context())
			ns := tenancy.NamespaceFromContext(r.Context())

			req := AuthzRequest{
				User:      id.User,
				Groups:    id.Groups,
				Resource:  resource,
				Verb:      verb,
				Namespace: ns,
			}

			allowed, err := authorizer.Authorize(r.Context(), req)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "internal_error",
					"message": "authorization check failed",
				})
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "forbidden",
					"message": fmt.Sprintf("insufficient permissions for %s/%s in namespace %s", resource, verb, ns),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthzMiddleware returns middleware that auto-maps the HTTP method and URL path
// to a (resource, verb) pair and performs the authorization check. This can be
// mounted as global middleware on all routes.
func AuthzMiddleware(authorizer Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mapping := MapRequest(r.Method, r.URL.Path)

			// If we cannot map the request, deny by default.
			if mapping == UnknownMapping {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "forbidden",
					"message": "unknown endpoint, access denied",
				})
				return
			}

			id, _ := IdentityFromContext(r.Context())
			ns := tenancy.NamespaceFromContext(r.Context())

			req := AuthzRequest{
				User:      id.User,
				Groups:    id.Groups,
				Resource:  mapping.Resource,
				Verb:      mapping.Verb,
				Namespace: ns,
			}

			allowed, err := authorizer.Authorize(r.Context(), req)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "internal_error",
					"message": "authorization check failed",
				})
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "forbidden",
					"message": fmt.Sprintf("insufficient permissions for %s/%s in namespace %s", mapping.Resource, mapping.Verb, ns),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
