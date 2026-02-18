package tenancy

import (
	"encoding/json"
	"net/http"
)

// Middleware returns HTTP middleware that resolves tenant context using the
// provided TenantResolver and stores it in the request context. On resolution
// failure it responds with a 400 JSON error.
func Middleware(resolver TenantResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tc, err := resolver.Resolve(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "bad_request",
					"message": err.Error(),
				})
				return
			}

			ctx := WithTenant(r.Context(), tc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewMiddleware is a convenience function that creates middleware with the
// appropriate resolver for the given TenancyMode.
func NewMiddleware(mode TenancyMode) func(http.Handler) http.Handler {
	var resolver TenantResolver
	switch mode {
	case ModeNamespace:
		resolver = NamespaceTenantResolver{}
	default:
		resolver = SingleTenantResolver{}
	}
	return Middleware(resolver)
}
