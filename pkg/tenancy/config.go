// Package tenancy provides multi-tenant context resolution and middleware
// for the catalog server. It supports single-tenant (backward compatible)
// and namespace-based multi-tenant modes.
package tenancy

// TenancyMode controls how tenant context is resolved.
type TenancyMode string

const (
	// ModeSingle uses "default" namespace for all requests (backward compat).
	ModeSingle TenancyMode = "single"
	// ModeNamespace requires namespace per request (multi-tenant).
	ModeNamespace TenancyMode = "namespace"
)
