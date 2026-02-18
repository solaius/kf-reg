package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdentityContextRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		identity Identity
	}{
		{
			name:     "basic user",
			identity: Identity{User: "alice", Groups: []string{"team-a"}},
		},
		{
			name:     "user with multiple groups",
			identity: Identity{User: "bob", Groups: []string{"team-a", "team-b", "admins"}},
		},
		{
			name:     "user with no groups",
			identity: Identity{User: "carol", Groups: nil},
		},
		{
			name:     "empty user",
			identity: Identity{User: "", Groups: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithIdentity(context.Background(), tt.identity)
			got, ok := IdentityFromContext(ctx)
			if !ok {
				t.Fatal("expected identity in context, got none")
			}
			if got.User != tt.identity.User {
				t.Errorf("User = %q, want %q", got.User, tt.identity.User)
			}
			if len(got.Groups) != len(tt.identity.Groups) {
				t.Fatalf("Groups length = %d, want %d", len(got.Groups), len(tt.identity.Groups))
			}
			for i, g := range got.Groups {
				if g != tt.identity.Groups[i] {
					t.Errorf("Groups[%d] = %q, want %q", i, g, tt.identity.Groups[i])
				}
			}
		})
	}
}

func TestIdentityFromContextMissing(t *testing.T) {
	_, ok := IdentityFromContext(context.Background())
	if ok {
		t.Error("expected no identity in empty context")
	}
}

func TestIdentityMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		userHeader     string
		groupHeader    string
		expectedUser   string
		expectedGroups []string
	}{
		{
			name:           "both headers present",
			userHeader:     "alice",
			groupHeader:    "team-a,team-b",
			expectedUser:   "alice",
			expectedGroups: []string{"team-a", "team-b"},
		},
		{
			name:           "missing user header defaults to anonymous",
			userHeader:     "",
			groupHeader:    "team-a",
			expectedUser:   "anonymous",
			expectedGroups: []string{"team-a"},
		},
		{
			name:           "missing group header",
			userHeader:     "bob",
			groupHeader:    "",
			expectedUser:   "bob",
			expectedGroups: nil,
		},
		{
			name:           "both headers missing",
			userHeader:     "",
			groupHeader:    "",
			expectedUser:   "anonymous",
			expectedGroups: nil,
		},
		{
			name:           "groups with spaces",
			userHeader:     "carol",
			groupHeader:    " team-a , team-b , admins ",
			expectedUser:   "carol",
			expectedGroups: []string{"team-a", "team-b", "admins"},
		},
		{
			name:           "whitespace-only user defaults to anonymous",
			userHeader:     "   ",
			groupHeader:    "",
			expectedUser:   "anonymous",
			expectedGroups: nil,
		},
		{
			name:           "single group no commas",
			userHeader:     "dave",
			groupHeader:    "operators",
			expectedUser:   "dave",
			expectedGroups: []string{"operators"},
		},
		{
			name:           "groups with empty segments",
			userHeader:     "eve",
			groupHeader:    "team-a,,team-b,",
			expectedUser:   "eve",
			expectedGroups: []string{"team-a", "team-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedID Identity
			var capturedOK bool

			handler := IdentityMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedID, capturedOK = IdentityFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.userHeader != "" {
				req.Header.Set("X-Remote-User", tt.userHeader)
			}
			if tt.groupHeader != "" {
				req.Header.Set("X-Remote-Group", tt.groupHeader)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if !capturedOK {
				t.Fatal("expected identity in context after middleware")
			}
			if capturedID.User != tt.expectedUser {
				t.Errorf("User = %q, want %q", capturedID.User, tt.expectedUser)
			}
			if len(capturedID.Groups) != len(tt.expectedGroups) {
				t.Fatalf("Groups length = %d, want %d", len(capturedID.Groups), len(tt.expectedGroups))
			}
			for i, g := range capturedID.Groups {
				if g != tt.expectedGroups[i] {
					t.Errorf("Groups[%d] = %q, want %q", i, g, tt.expectedGroups[i])
				}
			}
		})
	}
}
