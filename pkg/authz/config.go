package authz

// AuthzMode selects the authorization backend.
type AuthzMode string

const (
	// AuthzModeNone disables authorization checks (dev/backward compat).
	AuthzModeNone AuthzMode = "none"
	// AuthzModeSAR uses Kubernetes SubjectAccessReview for authorization.
	AuthzModeSAR AuthzMode = "sar"
)
