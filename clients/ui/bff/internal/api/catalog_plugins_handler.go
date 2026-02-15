package api

import (
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/model-registry/ui/bff/internal/constants"
	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/model-registry/ui/bff/internal/models"
)

type CatalogPluginListEnvelope Envelope[*models.CatalogPluginList, None]

func (app *App) GetAllCatalogPluginsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	catalogPlugins, err := app.repositories.ModelCatalogClient.GetAllCatalogPlugins(client)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	pluginsList := CatalogPluginListEnvelope{
		Data: catalogPlugins,
	}

	err = app.WriteJSON(w, http.StatusOK, pluginsList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
