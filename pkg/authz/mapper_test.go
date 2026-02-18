package authz

import (
	"net/http"
	"testing"
)

func TestMapRequest(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		wantResource string
		wantVerb     string
	}{
		// Plugin routes
		{
			name:         "list plugins",
			method:       http.MethodGet,
			path:         "/api/plugins",
			wantResource: ResourcePlugins,
			wantVerb:     VerbList,
		},
		{
			name:         "get plugin capabilities",
			method:       http.MethodGet,
			path:         "/api/plugins/mcp/capabilities",
			wantResource: ResourceCapabilities,
			wantVerb:     VerbGet,
		},

		// Management source routes
		{
			name:         "list sources",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/management/sources",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbList,
		},
		{
			name:         "apply source",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/apply-source",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbCreate,
		},
		{
			name:         "delete source",
			method:       http.MethodDelete,
			path:         "/api/mcp_catalog/v1alpha1/management/sources/my-source",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbDelete,
		},
		{
			name:         "enable source",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/sources/my-source/enable",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbUpdate,
		},
		{
			name:         "validate source",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/validate-source",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbUpdate,
		},

		// Refresh routes
		{
			name:         "refresh all",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/refresh",
			wantResource: ResourceJobs,
			wantVerb:     VerbCreate,
		},
		{
			name:         "refresh single source",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/refresh/my-source",
			wantResource: ResourceJobs,
			wantVerb:     VerbCreate,
		},

		// Action routes
		{
			name:         "source action",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/sources/my-source:action",
			wantResource: ResourceActions,
			wantVerb:     VerbExecute,
		},
		{
			name:         "entity action",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/entities/my-entity:action",
			wantResource: ResourceActions,
			wantVerb:     VerbExecute,
		},

		// Validate and rollback pseudo-methods
		{
			name:         "detailed validate",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/sources/my-source:validate",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbUpdate,
		},
		{
			name:         "rollback",
			method:       http.MethodPost,
			path:         "/api/mcp_catalog/v1alpha1/management/sources/my-source:rollback",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbUpdate,
		},

		// Diagnostics
		{
			name:         "diagnostics",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/management/diagnostics",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbGet,
		},

		// Action discovery
		{
			name:         "action list source",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/management/actions/source",
			wantResource: ResourceActions,
			wantVerb:     VerbList,
		},
		{
			name:         "action list asset",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/management/actions/asset",
			wantResource: ResourceActions,
			wantVerb:     VerbList,
		},

		// Entity routes
		{
			name:         "list entities",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/entities",
			wantResource: ResourceAssets,
			wantVerb:     VerbList,
		},
		{
			name:         "get entity",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/management/entities/my-entity",
			wantResource: ResourceAssets,
			wantVerb:     VerbGet,
		},

		// Governance routes
		{
			name:         "list governance items",
			method:       http.MethodGet,
			path:         "/api/governance/v1alpha1/approvals",
			wantResource: ResourceApprovals,
			wantVerb:     VerbList,
		},
		{
			name:         "post governance action",
			method:       http.MethodPost,
			path:         "/api/governance/v1alpha1/approvals/123/approve",
			wantResource: ResourceApprovals,
			wantVerb:     VerbApprove,
		},

		// Audit routes
		{
			name:         "list audit events",
			method:       http.MethodGet,
			path:         "/api/audit/v1alpha1/events",
			wantResource: ResourceAudit,
			wantVerb:     VerbList,
		},

		// Revisions
		{
			name:         "list revisions",
			method:       http.MethodGet,
			path:         "/api/mcp_catalog/v1alpha1/management/sources/my-source/revisions",
			wantResource: ResourceCatalogSources,
			wantVerb:     VerbGet,
		},

		// Unknown
		{
			name:         "unknown endpoint",
			method:       http.MethodGet,
			path:         "/healthz",
			wantResource: "",
			wantVerb:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapRequest(tt.method, tt.path)
			if got.Resource != tt.wantResource {
				t.Errorf("Resource = %q, want %q", got.Resource, tt.wantResource)
			}
			if got.Verb != tt.wantVerb {
				t.Errorf("Verb = %q, want %q", got.Verb, tt.wantVerb)
			}
		})
	}
}
