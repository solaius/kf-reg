package conformance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// TestPhase8RBACEndpointMapping verifies that known endpoints map to valid
// resource/verb tuples and no endpoint maps to the unknown mapping.
func TestPhase8RBACEndpointMapping(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		resource string
		verb     string
	}{
		// Plugin discovery endpoints.
		{"GET", "/api/plugins", authz.ResourcePlugins, authz.VerbList},
		{"GET", "/api/plugins/mcp/capabilities", authz.ResourceCapabilities, authz.VerbGet},

		// Management source endpoints.
		{"GET", "/api/mcp_catalog/v1alpha1/management/sources", authz.ResourceCatalogSources, authz.VerbList},
		{"POST", "/api/mcp_catalog/v1alpha1/management/apply-source", authz.ResourceCatalogSources, authz.VerbCreate},
		{"POST", "/api/mcp_catalog/v1alpha1/management/validate-source", authz.ResourceCatalogSources, authz.VerbUpdate},
		{"POST", "/api/mcp_catalog/v1alpha1/management/refresh/src1", authz.ResourceJobs, authz.VerbCreate},
		{"DELETE", "/api/mcp_catalog/v1alpha1/management/sources/src1", authz.ResourceCatalogSources, authz.VerbDelete},
		{"POST", "/api/mcp_catalog/v1alpha1/management/sources/src1/enable", authz.ResourceCatalogSources, authz.VerbUpdate},
		{"GET", "/api/mcp_catalog/v1alpha1/management/sources/src1/revisions", authz.ResourceCatalogSources, authz.VerbGet},
		{"GET", "/api/mcp_catalog/v1alpha1/management/diagnostics", authz.ResourceCatalogSources, authz.VerbGet},

		// Entity endpoints.
		{"GET", "/api/mcp_catalog/v1alpha1/management/entities/test", authz.ResourceAssets, authz.VerbGet},

		// Action endpoints.
		{"POST", "/api/mcp_catalog/v1alpha1/mcpservers/test:action", authz.ResourceActions, authz.VerbExecute},
		{"POST", "/api/mcp_catalog/v1alpha1/sources/test:validate", authz.ResourceCatalogSources, authz.VerbUpdate},
		{"POST", "/api/mcp_catalog/v1alpha1/sources/test:rollback", authz.ResourceCatalogSources, authz.VerbUpdate},

		// Governance routes.
		{"GET", "/api/governance/v1alpha1/assets/mcp/mcpserver/test", authz.ResourceApprovals, authz.VerbList},
		{"POST", "/api/governance/v1alpha1/assets/mcp/mcpserver/test/actions/lifecycle.setState", authz.ResourceApprovals, authz.VerbApprove},

		// Audit routes.
		{"GET", "/api/audit/v1alpha1/events", authz.ResourceAudit, authz.VerbList},

		// Management actions.
		{"GET", "/api/mcp_catalog/v1alpha1/management/actions", authz.ResourceActions, authz.VerbList},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			mapping := authz.MapRequest(tt.method, tt.path)
			if mapping == authz.UnknownMapping {
				t.Errorf("endpoint %s %s maps to UnknownMapping", tt.method, tt.path)
				return
			}
			if mapping.Resource != tt.resource {
				t.Errorf("resource = %q, want %q", mapping.Resource, tt.resource)
			}
			if mapping.Verb != tt.verb {
				t.Errorf("verb = %q, want %q", mapping.Verb, tt.verb)
			}
		})
	}
}

// TestPhase8RBACNoopAuthorizer verifies that NoopAuthorizer allows all requests.
func TestPhase8RBACNoopAuthorizer(t *testing.T) {
	authorizer := &authz.NoopAuthorizer{}

	tests := []authz.AuthzRequest{
		{User: "alice", Resource: "plugins", Verb: "list", Namespace: "default"},
		{User: "bob", Resource: "catalogsources", Verb: "create", Namespace: "team-a"},
		{User: "anonymous", Resource: "actions", Verb: "execute", Namespace: "team-b"},
	}

	for _, req := range tests {
		allowed, err := authorizer.Authorize(context.Background(), req)
		if err != nil {
			t.Fatalf("NoopAuthorizer.Authorize returned error: %v", err)
		}
		if !allowed {
			t.Errorf("NoopAuthorizer should allow %s/%s for %s, but denied", req.Resource, req.Verb, req.User)
		}
	}
}

// denyAllAuthorizer denies all requests.
type denyAllAuthorizer struct{}

func (d *denyAllAuthorizer) Authorize(_ context.Context, _ authz.AuthzRequest) (bool, error) {
	return false, nil
}

// TestPhase8RBACDenyAllMiddleware verifies that a deny-all authorizer returns 403
// with the correct JSON body format.
func TestPhase8RBACDenyAllMiddleware(t *testing.T) {
	authorizer := &denyAllAuthorizer{}

	// Build a handler chain: identity -> tenancy -> authz -> dummy handler.
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := authz.IdentityMiddleware()(
		tenancy.NewMiddleware(tenancy.ModeSingle)(
			authz.RequirePermission(authorizer, authz.ResourceCatalogSources, authz.VerbCreate)(inner),
		),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/management/apply-source", nil)
	req.Header.Set("X-Remote-User", "bob")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result["error"] != "forbidden" {
		t.Errorf("error = %q, want %q", result["error"], "forbidden")
	}
	if result["message"] == "" {
		t.Error("expected non-empty message in 403 response")
	}
}

// TestPhase8RBACAllowMiddleware verifies that NoopAuthorizer allows requests
// through the middleware chain.
func TestPhase8RBACAllowMiddleware(t *testing.T) {
	authorizer := &authz.NoopAuthorizer{}

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := authz.IdentityMiddleware()(
		tenancy.NewMiddleware(tenancy.ModeSingle)(
			authz.RequirePermission(authorizer, authz.ResourceCatalogSources, authz.VerbCreate)(inner),
		),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/management/apply-source", nil)
	req.Header.Set("X-Remote-User", "alice")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestPhase8RBACIdentityExtraction verifies identity middleware extracts user and groups.
func TestPhase8RBACIdentityExtraction(t *testing.T) {
	handler := authz.IdentityMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := authz.IdentityFromContext(r.Context())
		if !ok {
			t.Error("identity not set in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":   id.User,
			"groups": id.Groups,
		})
	}))

	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("extracts user and groups from headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
		req.Header.Set("X-Remote-User", "alice@example.com")
		req.Header.Set("X-Remote-Group", "admins, ml-team, editors")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			User   string   `json:"user"`
			Groups []string `json:"groups"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.User != "alice@example.com" {
			t.Errorf("user = %q, want %q", result.User, "alice@example.com")
		}
		if len(result.Groups) != 3 {
			t.Errorf("groups count = %d, want 3", len(result.Groups))
		}
	})

	t.Run("defaults to anonymous when no user header", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/test")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			User string `json:"user"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.User != "anonymous" {
			t.Errorf("user = %q, want %q", result.User, "anonymous")
		}
	})
}

// TestPhase8RBACCacheTTL verifies that the CachedAuthorizer respects TTL expiry.
func TestPhase8RBACCacheTTL(t *testing.T) {
	// Create a toggling authorizer that changes behavior.
	toggle := &toggleAuthorizer{allowed: true}
	cached := authz.NewCachedAuthorizer(toggle, 50*time.Millisecond)

	req := authz.AuthzRequest{
		User:      "alice",
		Resource:  "plugins",
		Verb:      "list",
		Namespace: "default",
	}

	// First call: allowed (cached).
	allowed, err := cached.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected first call to be allowed")
	}

	// Change the inner authorizer to deny.
	toggle.allowed = false

	// Immediate second call: should still be allowed (cached).
	allowed, err = cached.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected cached call to still be allowed")
	}

	// Wait for TTL to expire.
	time.Sleep(100 * time.Millisecond)

	// Third call: should now be denied (cache expired).
	allowed, err = cached.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected call after TTL expiry to be denied")
	}
}

// TestPhase8RBACNamespaceScopedDenial verifies that authorization checks
// include the namespace from the request context. A user authorized for
// team-a should be denied when accessing team-b resources.
func TestPhase8RBACNamespaceScopedDenial(t *testing.T) {
	// Authorizer that allows team-a but denies team-b.
	authorizer := &namespaceScopedAuthorizer{
		allowedNamespaces: map[string]bool{"team-a": true},
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := authz.IdentityMiddleware()(
		tenancy.NewMiddleware(tenancy.ModeNamespace)(
			authz.RequirePermission(authorizer, authz.ResourceCatalogSources, authz.VerbList)(inner),
		),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("allowed namespace returns 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/management/sources?namespace=team-a", nil)
		req.Header.Set("X-Remote-User", "alice")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for team-a, got %d", resp.StatusCode)
		}
	})

	t.Run("denied namespace returns 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/management/sources?namespace=team-b", nil)
		req.Header.Set("X-Remote-User", "alice")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for team-b, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["error"] != "forbidden" {
			t.Errorf("error = %q, want %q", result["error"], "forbidden")
		}
	})
}

// TestPhase8RBACAuthzMiddlewareAutoMapping verifies that AuthzMiddleware
// automatically maps endpoints to resource/verb and denies unknown endpoints.
func TestPhase8RBACAuthzMiddlewareAutoMapping(t *testing.T) {
	authorizer := &authz.NoopAuthorizer{}

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := authz.IdentityMiddleware()(
		tenancy.NewMiddleware(tenancy.ModeSingle)(
			authz.AuthzMiddleware(authorizer)(inner),
		),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("known endpoint allowed", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/plugins")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for /api/plugins, got %d", resp.StatusCode)
		}
	})

	t.Run("unknown endpoint denied", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/unknown/path")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for unknown endpoint, got %d", resp.StatusCode)
		}
	})
}

// namespaceScopedAuthorizer allows requests only for specific namespaces.
type namespaceScopedAuthorizer struct {
	allowedNamespaces map[string]bool
}

func (a *namespaceScopedAuthorizer) Authorize(_ context.Context, req authz.AuthzRequest) (bool, error) {
	return a.allowedNamespaces[req.Namespace], nil
}

// toggleAuthorizer is a test authorizer whose result can be toggled.
type toggleAuthorizer struct {
	allowed bool
}

func (a *toggleAuthorizer) Authorize(_ context.Context, _ authz.AuthzRequest) (bool, error) {
	return a.allowed, nil
}
