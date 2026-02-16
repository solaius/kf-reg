package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRoleExtractor(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected Role
	}{
		{"empty header defaults to viewer", "", RoleViewer},
		{"viewer header", "viewer", RoleViewer},
		{"operator header", "operator", RoleOperator},
		{"uppercase operator", "Operator", RoleOperator},
		{"unknown role defaults to viewer", "admin", RoleViewer},
		{"whitespace trimmed", "  operator  ", RoleOperator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.header != "" {
				req.Header.Set(RoleHeader, tt.header)
			}
			role := DefaultRoleExtractor(req)
			assert.Equal(t, tt.expected, role)
		})
	}
}

func TestHasRole(t *testing.T) {
	tests := []struct {
		name     string
		user     Role
		required Role
		expected bool
	}{
		{"viewer satisfies viewer", RoleViewer, RoleViewer, true},
		{"operator satisfies viewer", RoleOperator, RoleViewer, true},
		{"operator satisfies operator", RoleOperator, RoleOperator, true},
		{"viewer does not satisfy operator", RoleViewer, RoleOperator, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasRole(tt.user, tt.required))
		})
	}
}

func TestRequireRole(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("viewer can access viewer-level endpoint", func(t *testing.T) {
		middleware := RequireRole(RoleViewer, DefaultRoleExtractor)
		handler := middleware(okHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RoleHeader, "viewer")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("operator can access operator-level endpoint", func(t *testing.T) {
		middleware := RequireRole(RoleOperator, DefaultRoleExtractor)
		handler := middleware(okHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RoleHeader, "operator")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("viewer cannot access operator-level endpoint", func(t *testing.T) {
		middleware := RequireRole(RoleOperator, DefaultRoleExtractor)
		handler := middleware(okHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RoleHeader, "viewer")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("no header defaults to viewer and blocks operator endpoints", func(t *testing.T) {
		middleware := RequireRole(RoleOperator, DefaultRoleExtractor)
		handler := middleware(okHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("nil extractor uses default", func(t *testing.T) {
		middleware := RequireRole(RoleOperator, nil)
		handler := middleware(okHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RoleHeader, "operator")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
