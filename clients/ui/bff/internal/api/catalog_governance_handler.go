package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/model-registry/ui/bff/internal/constants"
	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
)

// GetGovernanceHandler retrieves governance metadata for an asset.
func (app *App) GetGovernanceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)

	result, err := app.repositories.ModelCatalogClient.GetGovernance(client, plugin, kind, name)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching governance: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// PatchGovernanceHandler updates governance metadata for an asset.
func (app *App) PatchGovernanceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)

	result, err := app.repositories.ModelCatalogClient.PatchGovernance(client, plugin, kind, name, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error patching governance: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGovernanceHistoryHandler retrieves audit history for an asset.
func (app *App) GetGovernanceHistoryHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)

	result, err := app.repositories.ModelCatalogClient.GetGovernanceHistory(client, plugin, kind, name, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching governance history: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// PostGovernanceActionHandler executes a governance action on an asset.
func (app *App) PostGovernanceActionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)
	action := ps.ByName(GovernanceActionName)

	result, err := app.repositories.ModelCatalogClient.PostGovernanceAction(client, plugin, kind, name, action, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error executing governance action: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGovernanceVersionsHandler lists versions for an asset.
func (app *App) GetGovernanceVersionsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)

	result, err := app.repositories.ModelCatalogClient.ListVersions(client, plugin, kind, name, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error listing versions: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CreateGovernanceVersionHandler creates a new version for an asset.
func (app *App) CreateGovernanceVersionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)

	result, err := app.repositories.ModelCatalogClient.CreateVersion(client, plugin, kind, name, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error creating version: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusCreated, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetGovernanceBindingsHandler lists environment bindings for an asset.
func (app *App) GetGovernanceBindingsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)

	result, err := app.repositories.ModelCatalogClient.ListBindings(client, plugin, kind, name)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error listing bindings: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// SetGovernanceBindingHandler sets an environment binding for an asset.
func (app *App) SetGovernanceBindingHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	plugin := ps.ByName(GovernancePluginName)
	kind := ps.ByName(GovernanceKindName)
	name := ps.ByName(GovernanceAssetName)
	env := ps.ByName(GovernanceEnvName)

	result, err := app.repositories.ModelCatalogClient.SetBinding(client, plugin, kind, name, env, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error setting binding: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetApprovalsHandler lists approval requests.
func (app *App) GetApprovalsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	result, err := app.repositories.ModelCatalogClient.ListApprovals(client, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error listing approvals: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetApprovalHandler retrieves a single approval request.
func (app *App) GetApprovalHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	id := ps.ByName(GovernanceApprovalId)

	result, err := app.repositories.ModelCatalogClient.GetApproval(client, id)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching approval: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// PostApprovalDecisionHandler submits a decision on an approval request.
func (app *App) PostApprovalDecisionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	id := ps.ByName(GovernanceApprovalId)

	result, err := app.repositories.ModelCatalogClient.PostApprovalDecision(client, id, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error submitting approval decision: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CancelApprovalHandler cancels a pending approval request.
func (app *App) CancelApprovalHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	id := ps.ByName(GovernanceApprovalId)

	result, err := app.repositories.ModelCatalogClient.CancelApproval(client, id, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error canceling approval: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetPoliciesHandler lists governance policies.
func (app *App) GetPoliciesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	result, err := app.repositories.ModelCatalogClient.ListPolicies(client)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error listing policies: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{Data: result}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
