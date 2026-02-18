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

// TenancyNamespacesResponse is the response envelope for the tenancy namespaces endpoint.
type TenancyNamespacesResponse struct {
	Namespaces []string `json:"namespaces"`
	Mode       string   `json:"mode"`
}

type TenancyNamespacesEnvelope Envelope[*TenancyNamespacesResponse, None]

// GetTenancyNamespacesHandler proxies namespace listing to the catalog server.
func (app *App) GetTenancyNamespacesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	responseData, err := client.GET("/api/tenancy/v1alpha1/namespaces")
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, fmt.Errorf("error fetching tenancy namespaces: %w", err))
		}
		return
	}

	var nsResponse TenancyNamespacesResponse
	if err := json.Unmarshal(responseData, &nsResponse); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding tenancy response: %w", err))
		return
	}

	envelope := TenancyNamespacesEnvelope{
		Data: &nsResponse,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
