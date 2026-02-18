package audit

import (
	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/catalog/governance"
)

// Router creates a chi.Router for the audit API.
// When authorizer is non-nil, endpoints require audit:list and audit:get permissions.
func Router(store *governance.AuditStore, authorizer authz.Authorizer) chi.Router {
	r := chi.NewRouter()

	listHandler := ListEventsHandler(store)
	getHandler := GetEventHandler(store)

	if authorizer != nil {
		r.Get("/events", authz.RequirePermission(authorizer, "audit", "list")(listHandler).ServeHTTP)
		r.Get("/events/{eventId}", authz.RequirePermission(authorizer, "audit", "get")(getHandler).ServeHTTP)
	} else {
		r.Get("/events", listHandler)
		r.Get("/events/{eventId}", getHandler)
	}

	return r
}
