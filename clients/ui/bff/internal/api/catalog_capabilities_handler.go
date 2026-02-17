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

// GetPluginCapabilitiesHandler proxies V2 capabilities for a specific plugin.
func (app *App) GetPluginCapabilitiesHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	pluginName := ps.ByName(CatalogPluginName)

	capabilities, err := app.repositories.ModelCatalogClient.GetPluginCapabilities(client, pluginName)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching plugin capabilities: %w", err))
		}
		return
	}

	envelope := Envelope[json.RawMessage, None]{
		Data: capabilities,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
