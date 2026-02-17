package plugin

import (
	"net/http"
	"strings"
)

// Role represents a user's access level for catalog management.
type Role string

const (
	// RoleViewer has read-only access (browse entities, view sources, view diagnostics).
	RoleViewer Role = "viewer"

	// RoleOperator has read access plus management operations (manage sources, trigger refresh).
	RoleOperator Role = "operator"
)

// TODO(production): The X-User-Role header is for development/testing only.
// In production, configure a real role extractor via WithRoleExtractor() server option
// that integrates with your authentication system (OIDC, Kubernetes RBAC, etc.).

// RoleHeader is the HTTP header used to extract the user's role.
const RoleHeader = "X-User-Role"

// RoleExtractor is a function that extracts a Role from an HTTP request.
// The default extractor reads the X-User-Role header.
type RoleExtractor func(r *http.Request) Role

// TODO(production): Replace with a production-grade role extractor.
// Use the WithRoleExtractor server option to inject an extractor that reads roles from:
//   - OIDC tokens (e.g., parsing JWT claims)
//   - Kubernetes user info (from API server authentication)
//   - External authorization services (e.g., OPA, Casbin)
// The RequireRole middleware design is correct and extensible — only the
// extractor function needs to change for production use.

// DefaultRoleExtractor reads the role from the X-User-Role header.
// Returns RoleViewer if the header is missing or unrecognized.
func DefaultRoleExtractor(r *http.Request) Role {
	header := strings.TrimSpace(strings.ToLower(r.Header.Get(RoleHeader)))
	switch header {
	case string(RoleOperator):
		return RoleOperator
	default:
		return RoleViewer
	}
}

// RequireRole returns middleware that enforces a minimum role.
// If the user's role is insufficient, it responds with 403 Forbidden.
func RequireRole(role Role, extractor RoleExtractor) func(http.Handler) http.Handler {
	if extractor == nil {
		extractor = DefaultRoleExtractor
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := extractor(r)
			if !hasRole(userRole, role) {
				http.Error(w, `{"error":"forbidden","message":"insufficient permissions"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hasRole checks whether userRole satisfies the required role.
// Operator can do everything Viewer can do plus management operations.
func hasRole(userRole, required Role) bool {
	switch required {
	case RoleViewer:
		// Everyone has at least viewer access
		return true
	case RoleOperator:
		return userRole == RoleOperator
	default:
		return false
	}
}
