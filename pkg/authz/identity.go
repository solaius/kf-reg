package authz

import (
	"context"
	"net/http"
	"strings"
)

// identityCtxKey is an unexported type used as the context key for Identity.
type identityCtxKey struct{}

// Identity represents the authenticated user making a request.
type Identity struct {
	User   string
	Groups []string
}

// WithIdentity returns a new context with the given Identity attached.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// IdentityFromContext retrieves the Identity from the context.
// Returns the zero value and false if no identity is set.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(Identity)
	return id, ok
}

// IdentityMiddleware returns HTTP middleware that extracts identity from
// X-Remote-User and X-Remote-Group headers and stores it in the request context.
// If X-Remote-User is missing, the user defaults to "anonymous".
// X-Remote-Group is comma-separated.
func IdentityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := strings.TrimSpace(r.Header.Get("X-Remote-User"))
			if user == "" {
				user = "anonymous"
			}

			var groups []string
			groupHeader := strings.TrimSpace(r.Header.Get("X-Remote-Group"))
			if groupHeader != "" {
				for _, g := range strings.Split(groupHeader, ",") {
					g = strings.TrimSpace(g)
					if g != "" {
						groups = append(groups, g)
					}
				}
			}

			id := Identity{User: user, Groups: groups}
			ctx := WithIdentity(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
