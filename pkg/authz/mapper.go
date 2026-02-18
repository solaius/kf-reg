package authz

import (
	"net/http"
	"strings"
)

// ResourceMapping maps an HTTP request to a catalog resource and verb for authorization.
type ResourceMapping struct {
	Resource string
	Verb     string
}

// UnknownMapping is returned when no known pattern matches the request.
// Callers should deny requests with this mapping by default.
var UnknownMapping = ResourceMapping{Resource: "", Verb: ""}

// MapRequest maps an HTTP method and URL path to a ResourceMapping.
// The mapper uses path segment patterns to determine the appropriate
// catalog resource and verb for authorization checks.
func MapRequest(method, path string) ResourceMapping {
	// Normalize the path: trim trailing slash.
	path = strings.TrimRight(path, "/")

	// Check specific patterns from most specific to least specific.

	// Action execution: POST *:action
	if method == http.MethodPost && strings.HasSuffix(path, ":action") {
		return ResourceMapping{Resource: ResourceActions, Verb: VerbExecute}
	}

	// Validation: POST *:validate
	if method == http.MethodPost && strings.HasSuffix(path, ":validate") {
		return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbUpdate}
	}

	// Rollback: POST *:rollback
	if method == http.MethodPost && strings.HasSuffix(path, ":rollback") {
		return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbUpdate}
	}

	// Management routes (contain /management/)
	if strings.Contains(path, "/management/") {
		return mapManagementRoute(method, path)
	}

	// Plugin capabilities: GET /api/plugins/*/capabilities
	if method == http.MethodGet && strings.HasPrefix(path, "/api/plugins/") && strings.HasSuffix(path, "/capabilities") {
		return ResourceMapping{Resource: ResourceCapabilities, Verb: VerbGet}
	}

	// Plugin list: GET /api/plugins
	if method == http.MethodGet && path == "/api/plugins" {
		return ResourceMapping{Resource: ResourcePlugins, Verb: VerbList}
	}

	// Governance routes: /api/governance/*
	if strings.HasPrefix(path, "/api/governance/") {
		return mapGovernanceRoute(method, path)
	}

	// Audit routes: /api/audit/*
	if strings.HasPrefix(path, "/api/audit/") {
		return mapAuditRoute(method)
	}

	// Entity routes (catalog entity endpoints).
	if strings.Contains(path, "/entities/") {
		return mapEntityRoute(method, path)
	}
	if strings.HasSuffix(path, "/entities") {
		return ResourceMapping{Resource: ResourceAssets, Verb: VerbList}
	}

	// Default: unknown pattern.
	return UnknownMapping
}

// mapManagementRoute handles routes containing /management/.
func mapManagementRoute(method, path string) ResourceMapping {
	// Extract the management sub-path.
	idx := strings.Index(path, "/management/")
	if idx < 0 {
		return UnknownMapping
	}
	subPath := path[idx+len("/management"):]

	switch method {
	case http.MethodGet:
		// GET /management/sources
		if subPath == "/sources" || strings.HasPrefix(subPath, "/sources?") {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbList}
		}
		// GET /management/diagnostics
		if subPath == "/diagnostics" {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbGet}
		}
		// GET /management/actions/*
		if strings.HasPrefix(subPath, "/actions/") || subPath == "/actions" {
			return ResourceMapping{Resource: ResourceActions, Verb: VerbList}
		}
		// GET /management/sources/*/revisions
		if strings.HasPrefix(subPath, "/sources/") && strings.HasSuffix(subPath, "/revisions") {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbGet}
		}
		// GET /management/entities/* (entity getter)
		if strings.HasPrefix(subPath, "/entities/") || subPath == "/entities" {
			return ResourceMapping{Resource: ResourceAssets, Verb: VerbGet}
		}
		return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbGet}

	case http.MethodPost:
		// POST /management/apply-source
		if subPath == "/apply-source" {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbCreate}
		}
		// POST /management/validate-source
		if subPath == "/validate-source" {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbUpdate}
		}
		// POST /management/refresh or /management/refresh/*
		if strings.HasPrefix(subPath, "/refresh") {
			return ResourceMapping{Resource: ResourceJobs, Verb: VerbCreate}
		}
		// POST /management/sources/*/enable
		if strings.HasPrefix(subPath, "/sources/") && strings.HasSuffix(subPath, "/enable") {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbUpdate}
		}
		return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbCreate}

	case http.MethodDelete:
		// DELETE /management/sources/*
		if strings.HasPrefix(subPath, "/sources/") {
			return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbDelete}
		}
		return ResourceMapping{Resource: ResourceCatalogSources, Verb: VerbDelete}
	}

	return UnknownMapping
}

// mapGovernanceRoute handles /api/governance/* routes.
func mapGovernanceRoute(method, _ string) ResourceMapping {
	switch method {
	case http.MethodGet:
		return ResourceMapping{Resource: ResourceApprovals, Verb: VerbList}
	case http.MethodPost:
		return ResourceMapping{Resource: ResourceApprovals, Verb: VerbApprove}
	default:
		return ResourceMapping{Resource: ResourceApprovals, Verb: VerbGet}
	}
}

// mapAuditRoute handles /api/audit/* routes.
func mapAuditRoute(method string) ResourceMapping {
	switch method {
	case http.MethodGet:
		return ResourceMapping{Resource: ResourceAudit, Verb: VerbList}
	default:
		return ResourceMapping{Resource: ResourceAudit, Verb: VerbGet}
	}
}

// mapEntityRoute handles routes containing /entities/.
func mapEntityRoute(method, _ string) ResourceMapping {
	switch method {
	case http.MethodGet:
		return ResourceMapping{Resource: ResourceAssets, Verb: VerbGet}
	case http.MethodPost:
		return ResourceMapping{Resource: ResourceAssets, Verb: VerbCreate}
	case http.MethodPut:
		return ResourceMapping{Resource: ResourceAssets, Verb: VerbUpdate}
	case http.MethodDelete:
		return ResourceMapping{Resource: ResourceAssets, Verb: VerbDelete}
	default:
		return ResourceMapping{Resource: ResourceAssets, Verb: VerbGet}
	}
}
