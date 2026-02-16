package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/model-registry/ui/bff/internal/constants"
	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/model-registry/ui/bff/internal/models"
)

type McpServerListEnvelope Envelope[*models.McpServerList, None]
type McpServerEnvelope Envelope[*models.McpServer, None]

func (app *App) GetMcpServersHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	// Resolve MCP plugin base path
	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, "mcp")
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving MCP plugin base path: %w", err))
		return
	}

	mcpServers, err := app.repositories.ModelCatalogClient.GetMcpServers(client, basePath)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := McpServerListEnvelope{
		Data: mcpServers,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetMcpServerHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	serverName := ps.ByName(McpServerName)

	basePath, err := app.repositories.ModelCatalogClient.ResolvePluginBasePath(client, "mcp")
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error resolving MCP plugin base path: %w", err))
		return
	}

	mcpServer, err := app.repositories.ModelCatalogClient.GetMcpServer(client, basePath, serverName)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := McpServerEnvelope{
		Data: mcpServer,
	}

	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
