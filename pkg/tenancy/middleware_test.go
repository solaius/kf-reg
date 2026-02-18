package tenancy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		mode       TenancyMode
		url        string
		header     string
		wantStatus int
		wantNS     string // expected namespace in context (empty if error expected)
	}{
		{
			name:       "single mode: no namespace param -> default",
			mode:       ModeSingle,
			url:        "/api/test",
			wantStatus: http.StatusOK,
			wantNS:     "default",
		},
		{
			name:       "single mode: namespace param provided -> still default",
			mode:       ModeSingle,
			url:        "/api/test?namespace=team-a",
			wantStatus: http.StatusOK,
			wantNS:     "default",
		},
		{
			name:       "namespace mode: namespace from query param",
			mode:       ModeNamespace,
			url:        "/api/test?namespace=team-a",
			wantStatus: http.StatusOK,
			wantNS:     "team-a",
		},
		{
			name:       "namespace mode: namespace from header",
			mode:       ModeNamespace,
			url:        "/api/test",
			header:     "team-b",
			wantStatus: http.StatusOK,
			wantNS:     "team-b",
		},
		{
			name:       "namespace mode: both query and header -> query wins",
			mode:       ModeNamespace,
			url:        "/api/test?namespace=from-query",
			header:     "from-header",
			wantStatus: http.StatusOK,
			wantNS:     "from-query",
		},
		{
			name:       "namespace mode: missing namespace -> 400",
			mode:       ModeNamespace,
			url:        "/api/test",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "namespace mode: invalid namespace (special chars) -> 400",
			mode:       ModeNamespace,
			url:        "/api/test?namespace=team_a!@#",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "namespace mode: invalid namespace (uppercase) -> 400",
			mode:       ModeNamespace,
			url:        "/api/test?namespace=Team-A",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedNS string
			handler := NewMiddleware(tt.mode)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedNS = NamespaceFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.header != "" {
				r.Header.Set(NamespaceHeader, tt.header)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				if capturedNS != tt.wantNS {
					t.Errorf("namespace in context = %q, want %q", capturedNS, tt.wantNS)
				}
			}

			if tt.wantStatus == http.StatusBadRequest {
				// Verify the error response is proper JSON.
				var errBody map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errBody); err != nil {
					t.Fatalf("failed to decode error body: %v", err)
				}
				if errBody["error"] != "bad_request" {
					t.Errorf("error field = %q, want %q", errBody["error"], "bad_request")
				}
				if errBody["message"] == "" {
					t.Error("expected non-empty message in error response")
				}
				if ct := w.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want %q", ct, "application/json")
				}
			}
		})
	}
}

func TestMiddleware_WithCustomResolver(t *testing.T) {
	// Test using Middleware() directly with a custom resolver.
	resolver := SingleTenantResolver{}
	handler := Middleware(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ns := NamespaceFromContext(r.Context())
		if ns != "default" {
			t.Errorf("expected namespace 'default', got %q", ns)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
