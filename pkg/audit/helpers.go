package audit

import (
	"strings"
)

// extractPlugin extracts the plugin name from a URL path.
// For paths like /api/mcp_catalog/v1alpha1/... it returns "mcp".
// For paths like /api/governance/v1alpha1/... it returns "governance".
// For paths like /api/audit/v1alpha1/... it returns "audit".
func extractPlugin(path string) string {
	// Trim leading slash and split.
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}

	// Expected format: api/{plugin}_catalog/{version}/...
	// or: api/governance/{version}/...
	// or: api/audit/{version}/...
	segment := parts[1] // e.g., "mcp_catalog", "governance", "audit"

	if idx := strings.Index(segment, "_catalog"); idx > 0 {
		return segment[:idx]
	}

	// Non-catalog API paths (governance, audit, plugins).
	return segment
}

// extractResourceType extracts the resource type from a URL path.
// Returns "sources", "entities", "actions", etc.
func extractResourceType(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	// Walk backwards to find the resource type segment.
	// Typical patterns:
	//   /api/{plugin}_catalog/{ver}/management/sources/{id}
	//   /api/{plugin}_catalog/{ver}/management/sources/{id}:action
	//   /api/{plugin}_catalog/{ver}/management/entities/{name}:action
	//   /api/{plugin}_catalog/{ver}/management/refresh/{id}
	//   /api/{plugin}_catalog/{ver}/management/apply-source
	//   /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/actions/{action}
	for i, p := range parts {
		switch p {
		case "sources", "entities", "actions", "refresh", "diagnostics",
			"apply-source", "validate-source", "revisions", "approvals",
			"assets", "bindings", "versions", "policies":
			return p
		case "management":
			// Look at next segment.
			if i+1 < len(parts) {
				next := parts[i+1]
				// Strip action suffix like "sources/{id}:validate".
				if colonIdx := strings.Index(next, ":"); colonIdx > 0 {
					return next[:colonIdx]
				}
				return next
			}
		}
	}

	return ""
}

// extractResourceIDs extracts resource IDs from a URL path.
// Returns IDs found in path parameter positions.
func extractResourceIDs(path string) []string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	var ids []string

	for i, p := range parts {
		// Source/entity IDs come after their type segment.
		switch p {
		case "sources", "entities", "approvals":
			if i+1 < len(parts) {
				id := parts[i+1]
				// Strip action suffix (e.g., "filesystem:action" -> "filesystem").
				if colonIdx := strings.Index(id, ":"); colonIdx > 0 {
					id = id[:colonIdx]
				}
				ids = append(ids, id)
			}
		}
	}

	return ids
}

// extractActionVerb returns a human-readable action name from the HTTP method and path.
func extractActionVerb(method, path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	// Check for :action suffix in path segments.
	for _, p := range parts {
		if colonIdx := strings.Index(p, ":"); colonIdx > 0 {
			suffix := p[colonIdx+1:]
			switch suffix {
			case "action":
				return "execute-action"
			case "validate":
				return "validate"
			case "rollback":
				return "rollback"
			}
		}
	}

	// Check for known management endpoints.
	for _, p := range parts {
		switch p {
		case "apply-source":
			return "apply-source"
		case "validate-source":
			return "validate-source"
		case "refresh":
			return "refresh"
		case "enable":
			return "enable-source"
		}
	}

	// Fall back to HTTP method mapping.
	switch method {
	case "POST":
		return "create"
	case "PUT":
		return "update"
	case "PATCH":
		return "patch"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// isManagementEndpoint returns true if the request should be audited.
// Management actions (POST, PUT, PATCH, DELETE) on management endpoints
// are audited. Pure browsing (GET) is not, except for diagnostic endpoints.
func isManagementEndpoint(method, path string) bool {
	// Never audit health endpoints.
	if isHealthEndpoint(path) {
		return false
	}

	// Mutating methods are always management actions.
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}

	// GET requests are not audited.
	return false
}

// isHealthEndpoint returns true for health-check paths.
func isHealthEndpoint(path string) bool {
	switch path {
	case "/livez", "/readyz", "/healthz":
		return true
	}
	return false
}
