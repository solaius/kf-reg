package governance

import (
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a chi router with governance API routes.
func NewRouter(store *GovernanceStore, auditStore *AuditStore) chi.Router {
	return NewRouterFull(store, auditStore, nil, nil, nil, nil)
}

// NewRouterWithApprovals creates a chi router with governance and approval routes.
// If approvalStore and evaluator are non-nil, approval endpoints and gated lifecycle
// transitions are enabled.
func NewRouterWithApprovals(
	store *GovernanceStore,
	auditStore *AuditStore,
	approvalStore *ApprovalStore,
	evaluator *ApprovalEvaluator,
) chi.Router {
	return NewRouterFull(store, auditStore, approvalStore, evaluator, nil, nil)
}

// NewRouterFull creates a chi router with all governance routes: governance CRUD,
// lifecycle actions, approval endpoints, version management, and environment bindings.
func NewRouterFull(
	store *GovernanceStore,
	auditStore *AuditStore,
	approvalStore *ApprovalStore,
	evaluator *ApprovalEvaluator,
	versionStore *VersionStore,
	bindingStore *BindingStore,
) chi.Router {
	r := chi.NewRouter()

	lifecycleHandler := NewLifecycleActionHandler(store, auditStore)
	if approvalStore != nil && evaluator != nil {
		lifecycleHandler.SetApprovalEngine(approvalStore, evaluator)
	}

	// Build a combined action handler that dispatches both lifecycle and promotion actions.
	var promotionHandler *PromotionActionHandler
	if versionStore != nil && bindingStore != nil {
		promotionHandler = NewPromotionActionHandler(store, versionStore, bindingStore, auditStore)
	}

	// Asset governance routes.
	r.Route("/assets/{plugin}/{kind}/{name}", func(r chi.Router) {
		r.Get("/", getGovernanceHandler(store))
		r.Patch("/", patchGovernanceHandler(store, auditStore))
		r.Get("/history", getHistoryHandler(store, auditStore))
		r.Post("/actions/{action}", combinedActionHandler(lifecycleHandler, promotionHandler))

		// Version routes.
		if versionStore != nil {
			r.Get("/versions", listVersionsHandler(versionStore, store))
			r.Post("/versions", createVersionHandler(versionStore, store, auditStore))
		}

		// Binding routes.
		if bindingStore != nil {
			r.Get("/bindings", listBindingsHandler(bindingStore))
			if versionStore != nil {
				r.Put("/bindings/{environment}", setBindingHandler(bindingStore, versionStore, store, auditStore))
			}
		}
	})

	// Approval routes.
	if approvalStore != nil && evaluator != nil {
		r.Route("/approvals", func(r chi.Router) {
			r.Get("/", listApprovalsHandler(approvalStore))
			r.Get("/{id}", getApprovalHandler(approvalStore))
			r.Post("/{id}/decisions", submitDecisionHandler(approvalStore, evaluator, lifecycleHandler, auditStore))
			r.Post("/{id}/cancel", cancelApprovalHandler(approvalStore, auditStore))
		})

		r.Get("/policies", listPoliciesHandler(evaluator))
	}

	return r
}
