package authz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/model-registry/pkg/tenancy"
)

func TestRequirePermission_Allowed(t *testing.T) {
	authorizer := &NoopAuthorizer{}

	handler := RequirePermission(authorizer, ResourceAssets, VerbGet)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := WithIdentity(req.Context(), Identity{User: "alice", Groups: []string{"team-a"}})
	ctx = tenancy.WithTenant(ctx, tenancy.TenantContext{Namespace: "team-a"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	authorizer := &denyAuthorizer{}

	handler := RequirePermission(authorizer, ResourceCatalogSources, VerbDelete)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called when denied")
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodDelete, "/test", nil)
	ctx := WithIdentity(req.Context(), Identity{User: "bob"})
	ctx = tenancy.WithTenant(ctx, tenancy.TenantContext{Namespace: "team-a"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "forbidden" {
		t.Errorf("error = %q, want %q", body["error"], "forbidden")
	}
	if body["message"] == "" {
		t.Error("expected non-empty message in response")
	}
}

func TestAuthzMiddleware_Allowed(t *testing.T) {
	authorizer := &NoopAuthorizer{}

	handler := AuthzMiddleware(authorizer)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	ctx := WithIdentity(req.Context(), Identity{User: "alice"})
	ctx = tenancy.WithTenant(ctx, tenancy.TenantContext{Namespace: "default"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAuthzMiddleware_UnknownEndpoint(t *testing.T) {
	authorizer := &NoopAuthorizer{}

	handler := AuthzMiddleware(authorizer)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called for unknown endpoint")
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/unknown/path", nil)
	ctx := WithIdentity(req.Context(), Identity{User: "alice"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

// denyAuthorizer always denies requests.
type denyAuthorizer struct{}

func (d *denyAuthorizer) Authorize(_ context.Context, _ AuthzRequest) (bool, error) {
	return false, nil
}
