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

// GetCatalogEntityListHandler proxies entity list requests to the plugin's endpoint.
func (app *App) GetCatalogEntityListHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	entityPlural := ps.ByName(CatalogEntityPlural)

	result, err := app.repositories.ModelCatalogClient.GetCatalogEntityList(client, pluginName, entityPlural, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching entity list: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{
		Data: result,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetCatalogEntityHandler proxies single entity get requests.
func (app *App) GetCatalogEntityHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	entityPlural := ps.ByName(CatalogEntityPlural)
	entityName := ps.ByName(CatalogEntityName)

	result, err := app.repositories.ModelCatalogClient.GetCatalogEntity(client, pluginName, entityPlural, entityName)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching entity: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{
		Data: result,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// PostCatalogEntityActionHandler proxies entity action requests.
func (app *App) PostCatalogEntityActionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	entityPlural := ps.ByName(CatalogEntityPlural)
	entityName := ps.ByName(CatalogEntityName)

	result, err := app.repositories.ModelCatalogClient.PostCatalogEntityAction(client, pluginName, entityPlural, entityName, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error executing entity action: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{
		Data: result,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// PostCatalogSourceActionHandler proxies source action requests.
func (app *App) PostCatalogSourceActionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	result, err := app.repositories.ModelCatalogClient.PostCatalogSourceAction(client, pluginName, sourceId, r.Body)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error executing source action: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{
		Data: result,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
