package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/model-registry/ui/bff/internal/models"
)

const (
	mgmtPrefix          = "/management"
	mgmtSourcesPath     = mgmtPrefix + "/sources"
	mgmtRefreshPath     = mgmtPrefix + "/refresh"
	mgmtDiagnosticsPath = mgmtPrefix + "/diagnostics"
	mgmtValidatePath    = mgmtPrefix + "/validate-source"
	mgmtApplyPath       = mgmtPrefix + "/apply-source"
	mgmtEnableSuffix    = "/enable"
	mgmtRevisionsPath   = "/revisions"
)

// CatalogManagementInterface defines the methods for plugin management operations.
type CatalogManagementInterface interface {
	GetPluginSources(client httpclient.HTTPClientInterface, basePath string) (*models.SourceInfoList, error)
	ValidatePluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.ValidationResult, error)
	ApplyPluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.SourceInfo, error)
	EnablePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceEnableRequest) (*models.SourceInfo, error)
	DeletePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) error
	RefreshPlugin(client httpclient.HTTPClientInterface, basePath string) (*models.RefreshResult, error)
	RefreshPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RefreshResult, error)
	GetPluginDiagnostics(client httpclient.HTTPClientInterface, basePath string) (*models.PluginDiagnostics, error)
	ResolvePluginBasePath(client httpclient.HTTPClientInterface, pluginName string) (string, error)
	ValidatePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceConfigPayload) (*models.DetailedValidationResult, error)
	GetPluginSourceRevisions(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RevisionList, error)
	RollbackPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.RollbackRequest) (*models.RollbackResult, error)
}

// CatalogManagement implements CatalogManagementInterface.
type CatalogManagement struct {
	CatalogManagementInterface
}

func (a CatalogManagement) GetPluginSources(client httpclient.HTTPClientInterface, basePath string) (*models.SourceInfoList, error) {
	path, err := url.JoinPath(basePath, mgmtSourcesPath)
	if err != nil {
		return nil, fmt.Errorf("error building sources path: %w", err)
	}

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching plugin sources: %w", err)
	}

	var sourceList models.SourceInfoList

	if err := json.Unmarshal(responseData, &sourceList); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &sourceList, nil
}

func (a CatalogManagement) ValidatePluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.ValidationResult, error) {
	path, err := url.JoinPath(basePath, mgmtValidatePath)
	if err != nil {
		return nil, fmt.Errorf("error building validate path: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	responseData, err := client.POST(path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error validating source config: %w", err)
	}

	var result models.ValidationResult

	if err := json.Unmarshal(responseData, &result); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &result, nil
}

func (a CatalogManagement) ApplyPluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.SourceInfo, error) {
	path, err := url.JoinPath(basePath, mgmtApplyPath)
	if err != nil {
		return nil, fmt.Errorf("error building apply path: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	responseData, err := client.POST(path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error applying source config: %w", err)
	}

	var sourceInfo models.SourceInfo

	if err := json.Unmarshal(responseData, &sourceInfo); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &sourceInfo, nil
}

func (a CatalogManagement) EnablePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceEnableRequest) (*models.SourceInfo, error) {
	path, err := url.JoinPath(basePath, mgmtSourcesPath, sourceId, "enable")
	if err != nil {
		return nil, fmt.Errorf("error building enable path: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	responseData, err := client.POST(path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error enabling/disabling source: %w", err)
	}

	var sourceInfo models.SourceInfo

	if err := json.Unmarshal(responseData, &sourceInfo); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &sourceInfo, nil
}

func (a CatalogManagement) DeletePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) error {
	path, err := url.JoinPath(basePath, mgmtSourcesPath, sourceId)
	if err != nil {
		return fmt.Errorf("error building delete path: %w", err)
	}

	_, err = client.DELETE(path)
	if err != nil {
		return fmt.Errorf("error deleting source: %w", err)
	}

	return nil
}

func (a CatalogManagement) RefreshPlugin(client httpclient.HTTPClientInterface, basePath string) (*models.RefreshResult, error) {
	path, err := url.JoinPath(basePath, mgmtRefreshPath)
	if err != nil {
		return nil, fmt.Errorf("error building refresh path: %w", err)
	}

	responseData, err := client.POST(path, nil)
	if err != nil {
		return nil, fmt.Errorf("error refreshing plugin: %w", err)
	}

	var result models.RefreshResult

	if err := json.Unmarshal(responseData, &result); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &result, nil
}

func (a CatalogManagement) RefreshPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RefreshResult, error) {
	path, err := url.JoinPath(basePath, mgmtRefreshPath, sourceId)
	if err != nil {
		return nil, fmt.Errorf("error building refresh source path: %w", err)
	}

	responseData, err := client.POST(path, nil)
	if err != nil {
		return nil, fmt.Errorf("error refreshing source: %w", err)
	}

	var result models.RefreshResult

	if err := json.Unmarshal(responseData, &result); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &result, nil
}

func (a CatalogManagement) GetPluginDiagnostics(client httpclient.HTTPClientInterface, basePath string) (*models.PluginDiagnostics, error) {
	path, err := url.JoinPath(basePath, mgmtDiagnosticsPath)
	if err != nil {
		return nil, fmt.Errorf("error building diagnostics path: %w", err)
	}

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching plugin diagnostics: %w", err)
	}

	var diagnostics models.PluginDiagnostics

	if err := json.Unmarshal(responseData, &diagnostics); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &diagnostics, nil
}

func (a CatalogManagement) ValidatePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceConfigPayload) (*models.DetailedValidationResult, error) {
	path, err := url.JoinPath(basePath, mgmtSourcesPath, sourceId+":validate")
	if err != nil {
		return nil, fmt.Errorf("error building validate source path: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	responseData, err := client.POST(path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error validating source: %w", err)
	}

	var result models.DetailedValidationResult

	if err := json.Unmarshal(responseData, &result); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &result, nil
}

func (a CatalogManagement) GetPluginSourceRevisions(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RevisionList, error) {
	path, err := url.JoinPath(basePath, mgmtSourcesPath, sourceId, mgmtRevisionsPath)
	if err != nil {
		return nil, fmt.Errorf("error building revisions path: %w", err)
	}

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching source revisions: %w", err)
	}

	var revisionList models.RevisionList

	if err := json.Unmarshal(responseData, &revisionList); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &revisionList, nil
}

func (a CatalogManagement) RollbackPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.RollbackRequest) (*models.RollbackResult, error) {
	path, err := url.JoinPath(basePath, mgmtSourcesPath, sourceId+":rollback")
	if err != nil {
		return nil, fmt.Errorf("error building rollback path: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	responseData, err := client.POST(path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error rolling back source: %w", err)
	}

	var result models.RollbackResult

	if err := json.Unmarshal(responseData, &result); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &result, nil
}

func (a CatalogManagement) ResolvePluginBasePath(client httpclient.HTTPClientInterface, pluginName string) (string, error) {
	responseData, err := client.GET(pluginsPath)
	if err != nil {
		return "", fmt.Errorf("error fetching plugins: %w", err)
	}

	var pluginList models.CatalogPluginList

	if err := json.Unmarshal(responseData, &pluginList); err != nil {
		return "", fmt.Errorf("error decoding plugins response: %w", err)
	}

	for _, plugin := range pluginList.Plugins {
		if plugin.Name == pluginName {
			return plugin.BasePath, nil
		}
	}

	return "", fmt.Errorf("plugin not found: %s", pluginName)
}
