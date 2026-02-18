package tenancy

import "context"

// ctxKey is an unexported type used as the context key for TenantContext.
type ctxKey struct{}

// TenantContext carries the resolved tenant information through request context.
type TenantContext struct {
	Namespace string
	User      string
	Groups    []string
}

// WithTenant returns a new context with the given TenantContext attached.
func WithTenant(ctx context.Context, tc TenantContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, tc)
}

// TenantFromContext retrieves the TenantContext from the context.
// Returns the zero value and false if no tenant is set.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	tc, ok := ctx.Value(ctxKey{}).(TenantContext)
	return tc, ok
}

// NamespaceFromContext is a convenience function that returns the namespace
// from the context, or "" if no tenant context is set.
func NamespaceFromContext(ctx context.Context) string {
	tc, ok := TenantFromContext(ctx)
	if !ok {
		return ""
	}
	return tc.Namespace
}
