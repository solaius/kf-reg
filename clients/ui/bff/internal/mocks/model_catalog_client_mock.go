package mocks

import (
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/kubeflow/model-registry/ui/bff/internal/models"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
	"github.com/stretchr/testify/mock"
)

type ModelCatalogClientMock struct {
	mock.Mock
}

func NewModelCatalogClientMock(logger *slog.Logger) (*ModelCatalogClientMock, error) {
	return &ModelCatalogClientMock{}, nil
}

func (m *ModelCatalogClientMock) GetAllCatalogModelsAcrossSources(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogModelList, error) {
	allModels := GetCatalogModelMocks()
	var filteredModels []models.CatalogModel

	sourceId := pageValues.Get("source")
	sourceLabel := pageValues.Get("sourceLabel")
	query := pageValues.Get("q")

	if sourceId != "" {
		for _, model := range allModels {
			if model.SourceId != nil && *model.SourceId == sourceId {
				filteredModels = append(filteredModels, model)
			}
		}
	} else if sourceLabel != "" {
		allSources := GetCatalogSourceMocks()
		var matchingSourceIds []string

		if sourceLabel == "null" {
			for _, source := range allSources {
				if len(source.Labels) == 0 {
					matchingSourceIds = append(matchingSourceIds, source.Id)
				}
			}
		} else {
			for _, source := range allSources {
				for _, label := range source.Labels {
					if label == sourceLabel {
						matchingSourceIds = append(matchingSourceIds, source.Id)
						break
					}
				}
			}
		}

		for _, model := range allModels {
			if model.SourceId != nil {
				for _, sid := range matchingSourceIds {
					if *model.SourceId == sid {
						filteredModels = append(filteredModels, model)
						break
					}
				}
			}
		}
	} else {
		filteredModels = allModels
	}

	if query != "" {
		var queryFilteredModels []models.CatalogModel
		queryLower := strings.ToLower(query)

		for _, model := range filteredModels {
			matchFound := false

			// Check name
			if strings.Contains(strings.ToLower(model.Name), queryLower) {
				matchFound = true
			}

			// Check description
			if !matchFound && model.Description != nil && strings.Contains(strings.ToLower(*model.Description), queryLower) {
				matchFound = true
			}

			// Check provider
			if !matchFound && model.Provider != nil && strings.Contains(strings.ToLower(*model.Provider), queryLower) {
				matchFound = true
			}

			if matchFound {
				queryFilteredModels = append(queryFilteredModels, model)
			}
		}

		filteredModels = queryFilteredModels
	}

	pageSizeStr := pageValues.Get("pageSize")
	pageSize := 10 // default
	if pageSizeStr != "" {
		if parsed, err := strconv.Atoi(pageSizeStr); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	pageTokenStr := pageValues.Get("nextPageToken")
	startIndex := 0
	if pageTokenStr != "" {
		if parsed, err := strconv.Atoi(pageTokenStr); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}

	totalSize := len(filteredModels)
	endIndex := startIndex + pageSize
	if endIndex > totalSize {
		endIndex = totalSize
	}

	var pagedModels []models.CatalogModel
	if startIndex < totalSize {
		pagedModels = filteredModels[startIndex:endIndex]
	} else {
		pagedModels = []models.CatalogModel{}
	}

	var nextPageToken string
	if endIndex < totalSize {
		nextPageToken = strconv.Itoa(endIndex)
	}

	size := len(pagedModels)
	if size > math.MaxInt32 {
		size = math.MaxInt32
	}
	ps := pageSize
	if ps > math.MaxInt32 {
		ps = math.MaxInt32
	}

	catalogModelList := models.CatalogModelList{
		Items:         pagedModels,
		Size:          int32(size),
		PageSize:      int32(ps),
		NextPageToken: nextPageToken,
	}

	return &catalogModelList, nil

}

func (m *ModelCatalogClientMock) GetCatalogSourceModel(client httpclient.HTTPClientInterface, sourceId string, modelName string) (*models.CatalogModel, error) {
	allModels := GetCatalogModelMocks()

	decodedModelName, err := url.QueryUnescape(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modelName: %w", err)
	}

	decodedModelName = strings.TrimPrefix(decodedModelName, "/")

	for _, model := range allModels {
		if model.SourceId != nil && *model.SourceId == sourceId && model.Name == decodedModelName {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("catalog model not found for sourceId: %s, modelName: %s", sourceId, decodedModelName)
}

func (m *ModelCatalogClientMock) GetAllCatalogSources(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogSourceList, error) {
	allMockSources := GetCatalogSourceListMock()
	var filteredMockSources []models.CatalogSource

	name := pageValues.Get("name")

	if name != "" {
		nameFilterLower := strings.ToLower(name)
		for _, source := range allMockSources.Items {
			if strings.ToLower(source.Id) == nameFilterLower || strings.ToLower(source.Name) == nameFilterLower {
				filteredMockSources = append(filteredMockSources, source)
			}
		}
	} else {
		filteredMockSources = allMockSources.Items
	}
	catalogSourceList := models.CatalogSourceList{
		Items:         filteredMockSources,
		PageSize:      int32(10),
		NextPageToken: "",
		Size:          int32(len(filteredMockSources)),
	}

	return &catalogSourceList, nil
}

func (m *ModelCatalogClientMock) GetCatalogSourceModelArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	var allMockModelArtifacts models.CatalogModelArtifactList

	if sourceId == "sample-source" && (modelName == "repo1%2Fgranite-8b-code-instruct" || modelName == "repo1%2Fgranite-8b-code-instruct-quantized.w4a16") {
		performanceArtifacts := GetCatalogPerformanceMetricsArtifactListMock(4)
		accuracyArtifacts := GetCatalogAccuracyMetricsArtifactListMock()
		modelArtifacts := GetCatalogModelArtifactListMock()
		combinedItems := append(performanceArtifacts.Items, accuracyArtifacts.Items...)
		combinedItems = append(combinedItems, modelArtifacts.Items...)
		allMockModelArtifacts = models.CatalogModelArtifactList{
			Items:         combinedItems,
			Size:          int32(len(combinedItems)),
			PageSize:      performanceArtifacts.PageSize,
			NextPageToken: "",
		}
	} else if sourceId == "sample-source" && modelName == "repo1%2Fgranite-7b-instruct" {
		accuracyArtifacts := GetCatalogAccuracyMetricsArtifactListMock()
		modelArtifacts := GetCatalogModelArtifactListMock()
		combinedItems := append(accuracyArtifacts.Items, modelArtifacts.Items...)
		allMockModelArtifacts = models.CatalogModelArtifactList{
			Items:         combinedItems,
			Size:          int32(len(combinedItems)),
			PageSize:      accuracyArtifacts.PageSize,
			NextPageToken: "",
		}
	} else if sourceId == "sample-source" && (modelName == "repo1%2Fgranite-3b-code-base") {
		allMockModelArtifacts = GetCatalogModelArtifactListMock()
	} else {
		allMockModelArtifacts = GetCatalogModelArtifactListMock()
	}

	return &allMockModelArtifacts, nil
}

func (m *ModelCatalogClientMock) GetCatalogModelPerformanceArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	allMockModelPerformanceArtifacts := GetCatalogPerformanceMetricsArtifactListMock(4)
	return &allMockModelPerformanceArtifacts, nil

}

func (m *ModelCatalogClientMock) GetCatalogFilterOptions(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	filterOptions := GetFilterOptionsListMock()

	return &filterOptions, nil
}

func (m *ModelCatalogClientMock) GetAllCatalogPlugins(client httpclient.HTTPClientInterface) (*models.CatalogPluginList, error) {
	pluginList := GetCatalogPluginListMock()
	return &pluginList, nil
}

func (m *ModelCatalogClientMock) GetPluginSources(client httpclient.HTTPClientInterface, basePath string) (*models.SourceInfoList, error) {
	return &models.SourceInfoList{
		Sources: []models.SourceInfo{
			{
				Id:      "source-1",
				Name:    "Test Source",
				Type:    "yaml",
				Enabled: true,
				Status:  "ready",
			},
		},
		Count: 1,
	}, nil
}

func (m *ModelCatalogClientMock) ValidatePluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.ValidationResult, error) {
	return &models.ValidationResult{
		Valid:   true,
		Message: "configuration is valid",
	}, nil
}

func (m *ModelCatalogClientMock) ApplyPluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.SourceInfo, error) {
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	return &models.SourceInfo{
		Id:      payload.Id,
		Name:    payload.Name,
		Type:    payload.Type,
		Enabled: enabled,
		Status:  "ready",
	}, nil
}

func (m *ModelCatalogClientMock) EnablePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceEnableRequest) (*models.SourceInfo, error) {
	return &models.SourceInfo{
		Id:      sourceId,
		Name:    "Test Source",
		Type:    "yaml",
		Enabled: payload.Enabled,
		Status:  "ready",
	}, nil
}

func (m *ModelCatalogClientMock) DeletePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) error {
	return nil
}

func (m *ModelCatalogClientMock) RefreshPlugin(client httpclient.HTTPClientInterface, basePath string) (*models.RefreshResult, error) {
	return &models.RefreshResult{
		Status:  "completed",
		Message: "all sources refreshed",
	}, nil
}

func (m *ModelCatalogClientMock) RefreshPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RefreshResult, error) {
	return &models.RefreshResult{
		Status:   "completed",
		Message:  "source refreshed",
		SourceId: sourceId,
	}, nil
}

func (m *ModelCatalogClientMock) GetPluginDiagnostics(client httpclient.HTTPClientInterface, basePath string) (*models.PluginDiagnostics, error) {
	return &models.PluginDiagnostics{
		PluginName:  "model",
		Healthy:     true,
		Version:     "v1alpha1",
		SourceCount: 1,
	}, nil
}

func (m *ModelCatalogClientMock) ResolvePluginBasePath(client httpclient.HTTPClientInterface, pluginName string) (string, error) {
	pluginList := GetCatalogPluginListMock()
	for _, plugin := range pluginList.Plugins {
		if plugin.Name == pluginName {
			return plugin.BasePath, nil
		}
	}
	return "", fmt.Errorf("plugin not found: %s", pluginName)
}

func getMcpServersMockData() []models.McpServer {
	return []models.McpServer{
		{
			ID:                  "1",
			Name:                "kubernetes-mcp-server",
			Description:         "MCP server for Kubernetes cluster management. Provides tools for pod lifecycle, deployment scaling, service discovery, and cluster diagnostics.",
			ServerUrl:           "stdio://kubernetes-mcp-server",
			TransportType:       "stdio",
			DeploymentMode:      "local",
			Image:               "quay.io/kubeflow/kubernetes-mcp-server:latest",
			SupportedTransports: "stdio,http",
			License:             "Apache-2.0",
			Verified:            true,
			Certified:           true,
			Provider:            "Red Hat",
			Category:            "Red Hat",
			ToolCount:           12,
			ResourceCount:       8,
			PromptCount:         3,
		},
		{
			ID:                  "2",
			Name:                "openshift-mcp-server",
			Description:         "MCP server for OpenShift Container Platform. Extends Kubernetes tools with OpenShift-specific resources: Routes, DeploymentConfigs, BuildConfigs, and ImageStreams.",
			ServerUrl:           "stdio://openshift-mcp-server",
			TransportType:       "stdio",
			DeploymentMode:      "local",
			Image:               "quay.io/openshift/mcp-server:latest",
			SupportedTransports: "stdio,http",
			License:             "Apache-2.0",
			Verified:            true,
			Certified:           true,
			Provider:            "Red Hat",
			Category:            "Red Hat",
			ToolCount:           18,
			ResourceCount:       12,
			PromptCount:         4,
		},
		{
			ID:                  "3",
			Name:                "ansible-mcp-server",
			Description:         "MCP server for Ansible Automation Platform. Provides tools for playbook execution, inventory management, role discovery, and collection browsing.",
			ServerUrl:           "stdio://ansible-mcp-server",
			TransportType:       "stdio",
			DeploymentMode:      "local",
			Image:               "quay.io/ansible/mcp-server:latest",
			SupportedTransports: "stdio",
			License:             "Apache-2.0",
			Verified:            true,
			Certified:           true,
			Provider:            "Red Hat",
			Category:            "Red Hat",
			ToolCount:           10,
			ResourceCount:       6,
			PromptCount:         5,
		},
		{
			ID:                  "4",
			Name:                "postgres-mcp-server",
			Description:         "MCP server for PostgreSQL database operations. Provides tools for query execution, schema inspection, migration management, and performance analysis.",
			ServerUrl:           "stdio://postgres-mcp-server",
			TransportType:       "stdio",
			DeploymentMode:      "local",
			Image:               "quay.io/crunchy/postgres-mcp:latest",
			SupportedTransports: "stdio",
			License:             "PostgreSQL",
			Verified:            true,
			Provider:            "Crunchy Data",
			Category:            "Database",
			ToolCount:           8,
			ResourceCount:       5,
			PromptCount:         2,
		},
		{
			ID:                  "5",
			Name:                "github-mcp-server",
			Description:         "MCP server for GitHub API integration. Provides tools for repository management, pull request workflows, issue tracking, and Actions automation.",
			ServerUrl:           "https://api.github.com/mcp",
			TransportType:       "http",
			DeploymentMode:      "remote",
			Endpoint:            "https://api.github.com/mcp",
			SupportedTransports: "http,sse",
			License:             "MIT",
			Verified:            true,
			Provider:            "GitHub",
			Category:            "DevOps",
			ToolCount:           15,
			ResourceCount:       10,
			PromptCount:         3,
		},
		{
			ID:                  "6",
			Name:                "slack-mcp-server",
			Description:         "MCP server for Slack workspace integration. Provides tools for messaging, channel management, user lookup, and workflow automation.",
			ServerUrl:           "https://slack.com/api/mcp",
			TransportType:       "http",
			DeploymentMode:      "remote",
			Endpoint:            "https://slack.com/api/mcp",
			SupportedTransports: "http",
			License:             "MIT",
			Provider:            "Slack",
			Category:            "Communication",
			ToolCount:           9,
			ResourceCount:       4,
			PromptCount:         2,
		},
		{
			ID:                  "7",
			Name:                "jira-mcp-server",
			Description:         "MCP server for Jira project management. Provides tools for issue CRUD, sprint management, board queries, and workflow transitions.",
			ServerUrl:           "https://jira.atlassian.com/mcp",
			TransportType:       "http",
			DeploymentMode:      "remote",
			Endpoint:            "https://jira.atlassian.com/mcp",
			SupportedTransports: "http",
			License:             "Apache-2.0",
			Provider:            "Atlassian",
			Category:            "DevOps",
			ToolCount:           11,
			ResourceCount:       7,
			PromptCount:         3,
		},
	}
}

func (m *ModelCatalogClientMock) GetMcpServers(client httpclient.HTTPClientInterface, basePath string) (*models.McpServerList, error) {
	servers := getMcpServersMockData()
	return &models.McpServerList{
		Items: servers,
		Size:  len(servers),
	}, nil
}

func (m *ModelCatalogClientMock) GetMcpServer(client httpclient.HTTPClientInterface, basePath string, name string) (*models.McpServer, error) {
	for _, s := range getMcpServersMockData() {
		if s.Name == name {
			return &s, nil
		}
	}
	return &models.McpServer{
		ID:   "unknown",
		Name: name,
	}, nil
}

func (m *ModelCatalogClientMock) CreateCatalogSourcePreview(client httpclient.HTTPClientInterface, sourcePreviewPayload models.CatalogSourcePreviewRequest, pageValues url.Values) (*models.CatalogSourcePreviewResult, error) {
	filterStatus := pageValues.Get("filterStatus")
	if filterStatus == "" {
		filterStatus = "all"
	}

	pageSize := 20
	if ps := pageValues.Get("pageSize"); ps != "" {
		_, _ = fmt.Sscanf(ps, "%d", &pageSize)
	}

	nextPageToken := pageValues.Get("nextPageToken")

	catalogSourcePreview := CreateCatalogSourcePreviewMockWithFilter(filterStatus, pageSize, nextPageToken)

	return &catalogSourcePreview, nil
}
