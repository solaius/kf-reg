package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/model-registry/ui/bff/internal/constants"
	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/model-registry/ui/bff/internal/models"
)

type SourceInfoListEnvelope Envelope[*models.SourceInfoList, None]
type SourceInfoEnvelope Envelope[*models.SourceInfo, None]
type ValidationResultEnvelope Envelope[*models.ValidationResult, None]
type RefreshResultEnvelope Envelope[*models.RefreshResult, None]
type PluginDiagnosticsEnvelope Envelope[*models.PluginDiagnostics, None]
type DetailedValidationResultEnvelope Envelope[*models.DetailedValidationResult, None]
type RevisionListEnvelope Envelope[*models.RevisionList, None]
type RollbackResultEnvelope Envelope[*models.RollbackResult, None]

func (app *App) GetPluginSourcesHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	sourceList, err := app.repositories.ModelCatalogClient.GetPluginSources(client, basePath)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := SourceInfoListEnvelope{
		Data: sourceList,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) ValidatePluginSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	var requestBody struct {
		Data models.SourceConfigPayload `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	result, err := app.repositories.ModelCatalogClient.ValidatePluginSourceConfig(client, basePath, requestBody.Data)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := ValidationResultEnvelope{
		Data: result,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) ApplyPluginSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	var requestBody struct {
		Data models.SourceConfigPayload `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	sourceInfo, err := app.repositories.ModelCatalogClient.ApplyPluginSourceConfig(client, basePath, requestBody.Data)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := SourceInfoEnvelope{
		Data: sourceInfo,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) EnablePluginSourceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	var requestBody struct {
		Data models.SourceEnableRequest `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	sourceInfo, err := app.repositories.ModelCatalogClient.EnablePluginSource(client, basePath, sourceId, requestBody.Data)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := SourceInfoEnvelope{
		Data: sourceInfo,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) DeletePluginSourceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	err = app.repositories.ModelCatalogClient.DeletePluginSource(client, basePath, sourceId)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *App) RefreshPluginHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	result, err := app.repositories.ModelCatalogClient.RefreshPlugin(client, basePath)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := RefreshResultEnvelope{
		Data: result,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) RefreshPluginSourceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	result, err := app.repositories.ModelCatalogClient.RefreshPluginSource(client, basePath, sourceId)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := RefreshResultEnvelope{
		Data: result,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetPluginDiagnosticsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	diagnostics, err := app.repositories.ModelCatalogClient.GetPluginDiagnostics(client, basePath)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := PluginDiagnosticsEnvelope{
		Data: diagnostics,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) ValidatePluginSourceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	var requestBody struct {
		Data models.SourceConfigPayload `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	result, err := app.repositories.ModelCatalogClient.ValidatePluginSource(client, basePath, sourceId, requestBody.Data)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := DetailedValidationResultEnvelope{
		Data: result,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetPluginSourceRevisionsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	revisionList, err := app.repositories.ModelCatalogClient.GetPluginSourceRevisions(client, basePath, sourceId)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := RevisionListEnvelope{
		Data: revisionList,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) RollbackPluginSourceHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)
	sourceId := ps.ByName(CatalogSourceId)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, pluginName)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving plugin base path: %w", err))
		return
	}

	var requestBody struct {
		Data models.RollbackRequest `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	result, err := app.repositories.ModelCatalogClient.RollbackPluginSource(client, basePath, sourceId, requestBody.Data)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := RollbackResultEnvelope{
		Data: result,
	}

	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
