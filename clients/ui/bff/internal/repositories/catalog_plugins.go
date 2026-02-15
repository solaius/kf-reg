package repositories

import (
	"encoding/json"
	"fmt"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/model-registry/ui/bff/internal/models"
)

const pluginsPath = "/plugins"

type CatalogPluginsInterface interface {
	GetAllCatalogPlugins(client httpclient.HTTPClientInterface) (*models.CatalogPluginList, error)
}

type CatalogPlugins struct {
	CatalogPluginsInterface
}

func (a CatalogPlugins) GetAllCatalogPlugins(client httpclient.HTTPClientInterface) (*models.CatalogPluginList, error) {
	responseData, err := client.GET(pluginsPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching plugins: %w", err)
	}

	var pluginList models.CatalogPluginList

	if err := json.Unmarshal(responseData, &pluginList); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &pluginList, nil
}
