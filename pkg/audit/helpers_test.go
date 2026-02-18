package audit

import (
	"testing"
)

func TestExtractPlugin(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "mcp catalog path",
			path: "/api/mcp_catalog/v1alpha1/mcpservers",
			want: "mcp",
		},
		{
			name: "model catalog path",
			path: "/api/model_catalog/v1alpha1/models",
			want: "model",
		},
		{
			name: "governance path",
			path: "/api/governance/v1alpha1/assets/mcp/mcpserver/test",
			want: "governance",
		},
		{
			name: "audit path",
			path: "/api/audit/v1alpha1/events",
			want: "audit",
		},
		{
			name: "plugins path",
			path: "/api/plugins",
			want: "plugins",
		},
		{
			name: "root path",
			path: "/livez",
			want: "",
		},
		{
			name: "knowledge catalog",
			path: "/api/knowledge_catalog/v1alpha1/sources",
			want: "knowledge",
		},
		{
			name: "management path",
			path: "/api/mcp_catalog/v1alpha1/management/sources",
			want: "mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlugin(tt.path)
			if got != tt.want {
				t.Errorf("extractPlugin(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractResourceType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "management sources",
			path: "/api/mcp_catalog/v1alpha1/management/sources",
			want: "sources",
		},
		{
			name: "management source by ID",
			path: "/api/mcp_catalog/v1alpha1/management/sources/hf-models",
			want: "sources",
		},
		{
			name: "management refresh",
			path: "/api/mcp_catalog/v1alpha1/management/refresh/hf-models",
			want: "refresh",
		},
		{
			name: "management apply-source",
			path: "/api/mcp_catalog/v1alpha1/management/apply-source",
			want: "apply-source",
		},
		{
			name: "entity action",
			path: "/api/mcp_catalog/v1alpha1/management/entities/test:action",
			want: "entities",
		},
		{
			name: "governance assets",
			path: "/api/governance/v1alpha1/assets/mcp/mcpserver/test",
			want: "assets",
		},
		{
			name: "governance approvals",
			path: "/api/governance/v1alpha1/approvals/abc-123",
			want: "approvals",
		},
		{
			name: "empty for health",
			path: "/livez",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceType(tt.path)
			if got != tt.want {
				t.Errorf("extractResourceType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractResourceIDs(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "source by ID",
			path: "/api/mcp_catalog/v1alpha1/management/sources/hf-models",
			want: []string{"hf-models"},
		},
		{
			name: "source action",
			path: "/api/mcp_catalog/v1alpha1/management/sources/hf-models:action",
			want: []string{"hf-models"},
		},
		{
			name: "entity action",
			path: "/api/mcp_catalog/v1alpha1/management/entities/filesystem:action",
			want: []string{"filesystem"},
		},
		{
			name: "no IDs for apply",
			path: "/api/mcp_catalog/v1alpha1/management/apply-source",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceIDs(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("extractResourceIDs(%q) = %v, want %v", tt.path, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractResourceIDs(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractActionVerb(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{
			name:   "apply source",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/apply-source",
			want:   "apply-source",
		},
		{
			name:   "refresh",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/refresh/hf-models",
			want:   "refresh",
		},
		{
			name:   "source action",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/sources/hf-models:action",
			want:   "execute-action",
		},
		{
			name:   "validate",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/sources/hf-models:validate",
			want:   "validate",
		},
		{
			name:   "rollback",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/sources/hf-models:rollback",
			want:   "rollback",
		},
		{
			name:   "delete source",
			method: "DELETE",
			path:   "/api/mcp_catalog/v1alpha1/management/sources/hf-models",
			want:   "delete",
		},
		{
			name:   "enable source",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/sources/hf-models/enable",
			want:   "enable-source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractActionVerb(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("extractActionVerb(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsManagementEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{
			name:   "POST apply-source",
			method: "POST",
			path:   "/api/mcp_catalog/v1alpha1/management/apply-source",
			want:   true,
		},
		{
			name:   "DELETE source",
			method: "DELETE",
			path:   "/api/mcp_catalog/v1alpha1/management/sources/hf-models",
			want:   true,
		},
		{
			name:   "PUT update",
			method: "PUT",
			path:   "/api/governance/v1alpha1/assets/mcp/mcpserver/test",
			want:   true,
		},
		{
			name:   "PATCH governance",
			method: "PATCH",
			path:   "/api/governance/v1alpha1/assets/mcp/mcpserver/test",
			want:   true,
		},
		{
			name:   "GET browse - not audited",
			method: "GET",
			path:   "/api/mcp_catalog/v1alpha1/mcpservers",
			want:   false,
		},
		{
			name:   "GET health - not audited",
			method: "GET",
			path:   "/livez",
			want:   false,
		},
		{
			name:   "GET readyz - not audited",
			method: "GET",
			path:   "/readyz",
			want:   false,
		},
		{
			name:   "POST health - not audited",
			method: "POST",
			path:   "/healthz",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isManagementEndpoint(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("isManagementEndpoint(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}
