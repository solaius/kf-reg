package repositories

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/model-registry/ui/bff/internal/models"
)

const mcpServersPath = "/mcpservers"

// McpCatalogInterface defines methods for browsing MCP server entities.
type McpCatalogInterface interface {
	GetMcpServers(client httpclient.HTTPClientInterface, basePath string) (*models.McpServerList, error)
	GetMcpServer(client httpclient.HTTPClientInterface, basePath string, name string) (*models.McpServer, error)
}

// McpCatalog implements McpCatalogInterface.
type McpCatalog struct {
	McpCatalogInterface
}

func (a McpCatalog) GetMcpServers(client httpclient.HTTPClientInterface, basePath string) (*models.McpServerList, error) {
	path, err := url.JoinPath(basePath, mcpServersPath)
	if err != nil {
		return nil, fmt.Errorf("error building MCP servers path: %w", err)
	}

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching MCP servers: %w", err)
	}

	var serverList models.McpServerList

	if err := json.Unmarshal(responseData, &serverList); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &serverList, nil
}

func (a McpCatalog) GetMcpServer(client httpclient.HTTPClientInterface, basePath string, name string) (*models.McpServer, error) {
	path, err := url.JoinPath(basePath, mcpServersPath, name)
	if err != nil {
		return nil, fmt.Errorf("error building MCP server path: %w", err)
	}

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching MCP server: %w", err)
	}

	var server models.McpServer

	if err := json.Unmarshal(responseData, &server); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &server, nil
}
