package audit

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/catalog/governance"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// responseCapture wraps http.ResponseWriter to capture the status code.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rc *responseCapture) WriteHeader(code int) {
	if !rc.written {
		rc.statusCode = code
		rc.written = true
	}
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if !rc.written {
		rc.statusCode = http.StatusOK
		rc.written = true
	}
	return rc.ResponseWriter.Write(b)
}

// AuditMiddleware creates middleware that captures audit events for management actions.
// It wraps the ResponseWriter to capture the status code, then records an
// AuditEventRecord after the handler completes.
func AuditMiddleware(store *governance.AuditStore, cfg *AuditConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if audit is disabled.
			if cfg == nil || !cfg.Enabled || store == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Skip non-management endpoints.
			if !isManagementEndpoint(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			startTime := time.Now()

			// Wrap ResponseWriter to capture status code.
			capture := &responseCapture{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Serve the request.
			next.ServeHTTP(capture, r)

			// After handler completes, record the audit event.
			statusCode := capture.statusCode

			// Determine outcome.
			outcome := outcomeFromStatus(statusCode)

			// Skip denied actions if LogDenied is false.
			if outcome == "denied" && !cfg.LogDenied {
				return
			}

			// Extract context values.
			ctx := r.Context()
			ns := tenancy.NamespaceFromContext(ctx)
			if ns == "" {
				ns = "default"
			}

			actor := "anonymous"
			var groups []string
			if id, ok := authz.IdentityFromContext(ctx); ok {
				actor = id.User
				groups = id.Groups
			}

			// Get request ID from chi middleware.
			requestID := middleware.GetReqID(ctx)

			// Get correlation ID from header, fall back to request ID.
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = requestID
			}

			// Extract path-based metadata.
			pluginName := extractPlugin(r.URL.Path)
			resourceType := extractResourceType(r.URL.Path)
			resourceIDs := extractResourceIDs(r.URL.Path)
			actionVerb := extractActionVerb(r.Method, r.URL.Path)

			event := &governance.AuditEventRecord{
				ID:            uuid.New().String(),
				Namespace:     ns,
				CorrelationID: correlationID,
				EventType:     "management",
				Actor:         actor,
				RequestID:     requestID,
				Plugin:        pluginName,
				ResourceType:  resourceType,
				ResourceIDs:   governance.JSONStringSlice(resourceIDs),
				Action:        actionVerb,
				ActionVerb:    actionVerb,
				Outcome:       outcome,
				StatusCode:    statusCode,
				CreatedAt:     startTime,
				EventMetadata: governance.JSONAny{
					"method":   r.Method,
					"path":     r.URL.Path,
					"duration": time.Since(startTime).String(),
					"groups":   groups,
				},
			}

			// Best-effort write: don't fail the request if audit write fails.
			if err := store.Append(event); err != nil {
				logger.Error("failed to write audit event", "error", err, "requestID", requestID)
			}
		})
	}
}

// outcomeFromStatus maps HTTP status codes to audit outcomes.
func outcomeFromStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code == http.StatusForbidden:
		return "denied"
	default:
		return "failure"
	}
}
