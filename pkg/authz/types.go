// Package authz provides authorization primitives for the catalog server.
// It supports Kubernetes SubjectAccessReview-based authorization and
// a no-op mode for development and backward compatibility.
package authz

import "context"

// APIGroup is the API group for catalog resources in Kubernetes RBAC.
const APIGroup = "catalog.kubeflow.org"

// Resource names for RBAC mapping.
const (
	ResourcePlugins        = "plugins"
	ResourceCapabilities   = "capabilities"
	ResourceCatalogSources = "catalogsources"
	ResourceAssets         = "assets"
	ResourceActions        = "actions"
	ResourceJobs           = "jobs"
	ResourceApprovals      = "approvals"
	ResourceAudit          = "audit"
)

// Verb names for RBAC mapping.
const (
	VerbGet     = "get"
	VerbList    = "list"
	VerbCreate  = "create"
	VerbUpdate  = "update"
	VerbDelete  = "delete"
	VerbExecute = "execute"
	VerbApprove = "approve"
)

// AuthzRequest represents an authorization check.
type AuthzRequest struct {
	User      string
	Groups    []string
	Resource  string
	Verb      string
	Namespace string // Empty for cluster-scoped checks.
}

// Authorizer checks whether a user is authorized to perform an action.
type Authorizer interface {
	Authorize(ctx context.Context, req AuthzRequest) (bool, error)
}
