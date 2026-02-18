package conformance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// TestPhase8BackwardCompatSingleTenant verifies that in single-tenant mode
// (the default), existing API patterns work without any namespace parameter.
func TestPhase8BackwardCompatSingleTenant(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	t.Run("plugins list works without namespace", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/api/plugins")
		if err != nil {
			t.Fatalf("GET /api/plugins failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("capabilities work without namespace", func(t *testing.T) {
		var response pluginsResponse
		getJSON(t, "/api/plugins", &response)

		for _, p := range response.Plugins {
			resp, err := http.Get(serverURL + "/api/plugins/" + p.Name + "/capabilities")
			if err != nil {
				t.Fatalf("GET capabilities failed: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("capabilities for %s returned %d", p.Name, resp.StatusCode)
			}
		}
	})

	t.Run("entity list endpoints work without namespace", func(t *testing.T) {
		var response pluginsResponse
		getJSON(t, "/api/plugins", &response)

		for _, p := range response.Plugins {
			if p.CapabilitiesV2 == nil {
				continue
			}
			for _, entity := range p.CapabilitiesV2.Entities {
				resp, err := http.Get(serverURL + entity.Endpoints.List)
				if err != nil {
					t.Fatalf("GET %s failed: %v", entity.Endpoints.List, err)
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("entity list %s returned %d", entity.Endpoints.List, resp.StatusCode)
				}
			}
		}
	})

	t.Run("health endpoints work without namespace", func(t *testing.T) {
		for _, path := range []string{"/livez", "/readyz", "/healthz"} {
			resp, err := http.Get(serverURL + path)
			if err != nil {
				t.Fatalf("GET %s failed: %v", path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
			}
		}
	})
}

// TestPhase8BackwardCompatAuthzNone verifies that when no authorizer is
// configured (the default), all requests are permitted.
func TestPhase8BackwardCompatAuthzNone(t *testing.T) {
	// Build a test server with NoopAuthorizer, simulating the default config.
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

	// Request without any auth headers should succeed (anonymous + noop authorizer).
	resp, err := http.Get(ts.URL + "/test")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with NoopAuthorizer, got %d", resp.StatusCode)
	}
}

// TestPhase8BackwardCompatTenancyModes verifies that tenant mode selection
// works correctly and single mode is backward-compatible.
func TestPhase8BackwardCompatTenancyModes(t *testing.T) {
	makeHandler := func(mode tenancy.TenancyMode) http.Handler {
		return tenancy.NewMiddleware(mode)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ns := tenancy.NamespaceFromContext(r.Context())
			_ = json.NewEncoder(w).Encode(map[string]string{"namespace": ns})
		}))
	}

	t.Run("single mode backward compat", func(t *testing.T) {
		ts := httptest.NewServer(makeHandler(tenancy.ModeSingle))
		defer ts.Close()

		// Request without namespace should succeed in single mode.
		resp, err := http.Get(ts.URL + "/api/plugins")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result["namespace"] != "default" {
			t.Errorf("namespace = %q, want %q", result["namespace"], "default")
		}
	})

	t.Run("namespace mode requires param", func(t *testing.T) {
		ts := httptest.NewServer(makeHandler(tenancy.ModeNamespace))
		defer ts.Close()

		// Request without namespace should return 400 in namespace mode.
		resp, err := http.Get(ts.URL + "/api/plugins")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 without namespace, got %d", resp.StatusCode)
		}

		// Request with namespace should succeed.
		resp, err = http.Get(ts.URL + "/api/plugins?namespace=team-a")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 with namespace, got %d", resp.StatusCode)
		}
	})
}

// TestPhase8BackwardCompatIdentityMiddleware verifies that the identity
// middleware defaults to "anonymous" when no headers are set, preserving
// backward compatibility for unauthenticated deployments.
func TestPhase8BackwardCompatIdentityMiddleware(t *testing.T) {
	handler := authz.IdentityMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := authz.IdentityFromContext(r.Context())
		if !ok {
			t.Error("identity not set")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"user": id.User})
	}))

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result["user"] != "anonymous" {
		t.Errorf("user = %q, want %q", result["user"], "anonymous")
	}
}

// TestPhase8BackwardCompatContextPropagation verifies that absence of
// tenant context does not cause panics in downstream code.
func TestPhase8BackwardCompatContextPropagation(t *testing.T) {
	// NamespaceFromContext on a plain context should return "".
	ns := tenancy.NamespaceFromContext(context.Background())
	if ns != "" {
		t.Errorf("NamespaceFromContext on empty context = %q, want empty", ns)
	}

	// IdentityFromContext on a plain context should return false.
	_, ok := authz.IdentityFromContext(context.Background())
	if ok {
		t.Error("IdentityFromContext should return false on empty context")
	}
}
