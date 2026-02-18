package tenancy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSingleTenantResolver(t *testing.T) {
	resolver := SingleTenantResolver{}

	// Should always return "default" regardless of request contents.
	tests := []struct {
		name string
		url  string
	}{
		{"no params", "/api/test"},
		{"with namespace param", "/api/test?namespace=team-a"},
		{"with other params", "/api/test?foo=bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			tc, err := resolver.Resolve(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.Namespace != "default" {
				t.Errorf("Namespace = %q, want %q", tc.Namespace, "default")
			}
		})
	}
}

func TestNamespaceTenantResolver(t *testing.T) {
	resolver := NamespaceTenantResolver{}

	tests := []struct {
		name      string
		url       string
		header    string
		wantNS    string
		wantError bool
	}{
		{
			name:   "namespace from query param",
			url:    "/api/test?namespace=team-a",
			wantNS: "team-a",
		},
		{
			name:   "namespace from header",
			url:    "/api/test",
			header: "team-b",
			wantNS: "team-b",
		},
		{
			name:   "query param takes precedence over header",
			url:    "/api/test?namespace=from-query",
			header: "from-header",
			wantNS: "from-query",
		},
		{
			name:      "missing namespace",
			url:       "/api/test",
			wantError: true,
		},
		{
			name:      "invalid namespace - uppercase",
			url:       "/api/test?namespace=Team-A",
			wantError: true,
		},
		{
			name:      "invalid namespace - special chars",
			url:       "/api/test?namespace=team_a!",
			wantError: true,
		},
		{
			name:      "invalid namespace - starts with hyphen",
			url:       "/api/test?namespace=-team",
			wantError: true,
		},
		{
			name:      "invalid namespace - ends with hyphen",
			url:       "/api/test?namespace=team-",
			wantError: true,
		},
		{
			name:   "valid namespace - single char",
			url:    "/api/test?namespace=a",
			wantNS: "a",
		},
		{
			name:   "valid namespace - with hyphens",
			url:    "/api/test?namespace=my-team-ns",
			wantNS: "my-team-ns",
		},
		{
			name:   "valid namespace - numeric",
			url:    "/api/test?namespace=123",
			wantNS: "123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.header != "" {
				r.Header.Set(NamespaceHeader, tt.header)
			}

			tc, err := resolver.Resolve(r)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.Namespace != tt.wantNS {
				t.Errorf("Namespace = %q, want %q", tc.Namespace, tt.wantNS)
			}
		})
	}
}

func TestValidateNamespace_TooLong(t *testing.T) {
	// 64 characters exceeds the 63-char max.
	long := "a"
	for i := 0; i < 63; i++ {
		long += "b"
	}
	resolver := NamespaceTenantResolver{}
	r := httptest.NewRequest(http.MethodGet, "/api/test?namespace="+long, nil)
	_, err := resolver.Resolve(r)
	if err == nil {
		t.Fatal("expected error for namespace exceeding 63 chars")
	}
}

func TestValidateNamespace_ExactlyMaxLength(t *testing.T) {
	// 63 characters should be valid.
	ns := "a"
	for i := 0; i < 62; i++ {
		ns += "b"
	}
	resolver := NamespaceTenantResolver{}
	r := httptest.NewRequest(http.MethodGet, "/api/test?namespace="+ns, nil)
	tc, err := resolver.Resolve(r)
	if err != nil {
		t.Fatalf("unexpected error for 63-char namespace: %v", err)
	}
	if tc.Namespace != ns {
		t.Errorf("Namespace = %q, want %q", tc.Namespace, ns)
	}
}
