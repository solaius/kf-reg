package jobs

import (
	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/model-registry/pkg/authz"
)

// Router creates a chi.Router for the job status API.
// When authorizer is non-nil, endpoints require jobs:list, jobs:get, and jobs:create permissions.
func Router(store *JobStore, authorizer authz.Authorizer) chi.Router {
	r := chi.NewRouter()

	listHandler := ListJobsHandler(store)
	getHandler := GetJobHandler(store)
	cancelHandler := CancelJobHandler(store)

	if authorizer != nil {
		r.Get("/refresh", authz.RequirePermission(authorizer, "jobs", "list")(listHandler).ServeHTTP)
		r.Get("/refresh/{jobId}", authz.RequirePermission(authorizer, "jobs", "get")(getHandler).ServeHTTP)
		r.Post("/refresh/{jobId}:cancel", authz.RequirePermission(authorizer, "jobs", "create")(cancelHandler).ServeHTTP)
	} else {
		r.Get("/refresh", listHandler)
		r.Get("/refresh/{jobId}", getHandler)
		r.Post("/refresh/{jobId}:cancel", cancelHandler)
	}

	return r
}
