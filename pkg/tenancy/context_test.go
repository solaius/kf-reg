package tenancy

import (
	"context"
	"testing"
)

func TestWithTenantAndTenantFromContext(t *testing.T) {
	tc := TenantContext{
		Namespace: "team-a",
		User:      "alice",
		Groups:    []string{"developers", "admins"},
	}

	ctx := WithTenant(context.Background(), tc)
	got, ok := TenantFromContext(ctx)
	if !ok {
		t.Fatal("expected TenantFromContext to return true")
	}
	if got.Namespace != tc.Namespace {
		t.Errorf("Namespace = %q, want %q", got.Namespace, tc.Namespace)
	}
	if got.User != tc.User {
		t.Errorf("User = %q, want %q", got.User, tc.User)
	}
	if len(got.Groups) != len(tc.Groups) {
		t.Fatalf("Groups length = %d, want %d", len(got.Groups), len(tc.Groups))
	}
	for i, g := range got.Groups {
		if g != tc.Groups[i] {
			t.Errorf("Groups[%d] = %q, want %q", i, g, tc.Groups[i])
		}
	}
}

func TestTenantFromContext_Missing(t *testing.T) {
	_, ok := TenantFromContext(context.Background())
	if ok {
		t.Fatal("expected TenantFromContext to return false for empty context")
	}
}

func TestNamespaceFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "with tenant set",
			ctx:  WithTenant(context.Background(), TenantContext{Namespace: "my-ns"}),
			want: "my-ns",
		},
		{
			name: "without tenant set",
			ctx:  context.Background(),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NamespaceFromContext(tt.ctx)
			if got != tt.want {
				t.Errorf("NamespaceFromContext() = %q, want %q", got, tt.want)
			}
		})
	}
}
