package repositories

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
)

const (
	pluginCapabilitiesPathFmt = "/api/plugins/%s/capabilities"
	pluginEntityPathFmt       = "/api/%s_catalog/v1alpha1/%s"
	pluginEntityGetPathFmt    = "/api/%s_catalog/v1alpha1/%s/%s"
	pluginEntityActionPathFmt = "/api/%s_catalog/v1alpha1/management/entities/%s:action"
	pluginSourceActionPathFmt = "/api/%s_catalog/v1alpha1/management/sources/%s:action"
)

// CatalogEntitiesInterface defines methods for generic catalog entity browsing.
type CatalogEntitiesInterface interface {
	GetPluginCapabilities(client httpclient.HTTPClientInterface, pluginName string) (json.RawMessage, error)
	GetCatalogEntityList(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, queryParams url.Values) (json.RawMessage, error)
	GetCatalogEntity(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, entityName string) (json.RawMessage, error)
	PostCatalogEntityAction(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, entityName string, body io.Reader) (json.RawMessage, error)
	PostCatalogSourceAction(client httpclient.HTTPClientInterface, pluginName string, sourceId string, body io.Reader) (json.RawMessage, error)
}

// CatalogEntities implements CatalogEntitiesInterface.
type CatalogEntities struct {
	CatalogEntitiesInterface
}

func (a CatalogEntities) GetPluginCapabilities(client httpclient.HTTPClientInterface, pluginName string) (json.RawMessage, error) {
	path := fmt.Sprintf(pluginCapabilitiesPathFmt, pluginName)

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching plugin capabilities: %w", err)
	}

	return json.RawMessage(responseData), nil
}

func (a CatalogEntities) GetCatalogEntityList(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, queryParams url.Values) (json.RawMessage, error) {
	path := fmt.Sprintf(pluginEntityPathFmt, pluginName, entityPlural)
	path = UrlWithPageParams(path, queryParams)

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching entity list: %w", err)
	}

	return json.RawMessage(responseData), nil
}

func (a CatalogEntities) GetCatalogEntity(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, entityName string) (json.RawMessage, error) {
	path := fmt.Sprintf(pluginEntityGetPathFmt, pluginName, entityPlural, entityName)

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	return json.RawMessage(responseData), nil
}

func (a CatalogEntities) PostCatalogEntityAction(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, entityName string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(pluginEntityActionPathFmt, pluginName, entityName)

	responseData, err := client.POST(path, body)
	if err != nil {
		return nil, fmt.Errorf("error executing entity action: %w", err)
	}

	return json.RawMessage(responseData), nil
}

func (a CatalogEntities) PostCatalogSourceAction(client httpclient.HTTPClientInterface, pluginName string, sourceId string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(pluginSourceActionPathFmt, pluginName, sourceId)

	responseData, err := client.POST(path, body)
	if err != nil {
		return nil, fmt.Errorf("error executing source action: %w", err)
	}

	return json.RawMessage(responseData), nil
}
