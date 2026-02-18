package authz

import "context"

// NoopAuthorizer always allows all requests. Used when CATALOG_AUTHZ_MODE=none.
type NoopAuthorizer struct{}

// Authorize always returns true.
func (n *NoopAuthorizer) Authorize(_ context.Context, _ AuthzRequest) (bool, error) {
	return true, nil
}
