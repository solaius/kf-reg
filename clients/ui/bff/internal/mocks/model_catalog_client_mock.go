package mocks

import (
	"encoding/json"
	"fmt"
	"io"
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
	// Stateful plugin source storage keyed by plugin name ("mcp" or "model")
	pluginSources map[string][]models.SourceInfo
}

func NewModelCatalogClientMock(logger *slog.Logger) (*ModelCatalogClientMock, error) {
	return &ModelCatalogClientMock{
		pluginSources: make(map[string][]models.SourceInfo),
	}, nil
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

func (m *ModelCatalogClientMock) pluginKeyFromBasePath(basePath string) string {
	if strings.Contains(basePath, "mcp") {
		return "mcp"
	}
	return "model"
}

func (m *ModelCatalogClientMock) initPluginSources(key string) {
	if _, ok := m.pluginSources[key]; ok {
		return
	}
	if key == "mcp" {
		m.pluginSources[key] = []models.SourceInfo{
			{
				Id:      "mcp-yaml-source",
				Name:    "MCP YAML Source",
				Type:    "yaml",
				Enabled: true,
				Status:  models.SourceStatus{State: "available", EntityCount: 7},
				Properties: map[string]interface{}{
					"content": "servers:\n  - name: kubernetes-mcp-server\n    description: MCP server for Kubernetes cluster management\n    serverUrl: stdio://kubernetes-mcp-server\n    transportType: stdio\n    deploymentMode: local\n    image: quay.io/kubeflow/kubernetes-mcp-server:latest\n  - name: openshift-mcp-server\n    description: MCP server for OpenShift Container Platform\n    serverUrl: stdio://openshift-mcp-server\n    transportType: stdio\n    deploymentMode: local\n    image: quay.io/kubeflow/openshift-mcp-server:latest\n",
				},
			},
			{
				Id:      "community-servers",
				Name:    "Community Servers",
				Type:    "yaml",
				Enabled: false,
				Status:  models.SourceStatus{State: "disabled", EntityCount: 0},
				Properties: map[string]interface{}{
					"content": "servers:\n  - name: community-server-1\n    description: Community contributed server\n",
				},
			},
		}
	} else {
		// IDs must match the model catalog settings source config IDs
		// so that "Manage source" navigates correctly to /model-catalog-settings/manage-source/:id
		m.pluginSources[key] = []models.SourceInfo{
			{
				Id:      "bella_ai_validated_models",
				Name:    "Bella AI validated",
				Type:    "yaml",
				Enabled: true,
				Status:  models.SourceStatus{State: "available", EntityCount: 12},
			},
			{
				Id:      "custom_yaml_models",
				Name:    "Custom yaml",
				Type:    "yaml",
				Enabled: true,
				Status:  models.SourceStatus{State: "available", EntityCount: 5},
			},
			{
				Id:      "dora_ai_models",
				Name:    "Dora AI",
				Type:    "yaml",
				Enabled: true,
				Status:  models.SourceStatus{State: "available", EntityCount: 5},
			},
			{
				Id:      "hugging_face_source",
				Name:    "Hugging face source",
				Type:    "huggingface",
				Enabled: true,
				Status:  models.SourceStatus{State: "loading", EntityCount: 0},
			},
			{
				Id:      "sample_source_models",
				Name:    "Sample source",
				Type:    "yaml",
				Enabled: false,
				Status:  models.SourceStatus{State: "disabled", EntityCount: 0},
			},
		}
	}
}

func (m *ModelCatalogClientMock) GetPluginSources(client httpclient.HTTPClientInterface, basePath string) (*models.SourceInfoList, error) {
	key := m.pluginKeyFromBasePath(basePath)
	m.initPluginSources(key)

	sources := m.pluginSources[key]
	return &models.SourceInfoList{
		Sources: sources,
		Count:   len(sources),
	}, nil
}

func (m *ModelCatalogClientMock) ValidatePluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.ValidationResult, error) {
	return &models.ValidationResult{
		Valid: true,
	}, nil
}

func (m *ModelCatalogClientMock) ApplyPluginSourceConfig(client httpclient.HTTPClientInterface, basePath string, payload models.SourceConfigPayload) (*models.SourceInfo, error) {
	key := m.pluginKeyFromBasePath(basePath)
	m.initPluginSources(key)

	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	state := "available"
	if !enabled {
		state = "disabled"
	}

	newSource := models.SourceInfo{
		Id:         payload.Id,
		Name:       payload.Name,
		Type:       payload.Type,
		Enabled:    enabled,
		Status:     models.SourceStatus{State: state, EntityCount: 0},
		Properties: payload.Properties,
	}

	// Update existing or append new
	found := false
	for i, s := range m.pluginSources[key] {
		if s.Id == payload.Id {
			m.pluginSources[key][i] = newSource
			found = true
			break
		}
	}
	if !found {
		m.pluginSources[key] = append(m.pluginSources[key], newSource)
	}

	return &newSource, nil
}

func (m *ModelCatalogClientMock) EnablePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceEnableRequest) (*models.SourceInfo, error) {
	key := m.pluginKeyFromBasePath(basePath)
	m.initPluginSources(key)

	for i, s := range m.pluginSources[key] {
		if s.Id == sourceId {
			m.pluginSources[key][i].Enabled = payload.Enabled
			if payload.Enabled {
				m.pluginSources[key][i].Status.State = "available"
			} else {
				m.pluginSources[key][i].Status.State = "disabled"
			}
			return &m.pluginSources[key][i], nil
		}
	}

	// Source not found, return a generic response
	state := "available"
	if !payload.Enabled {
		state = "disabled"
	}
	return &models.SourceInfo{
		Id:      sourceId,
		Name:    sourceId,
		Type:    "yaml",
		Enabled: payload.Enabled,
		Status:  models.SourceStatus{State: state, EntityCount: 0},
	}, nil
}

func (m *ModelCatalogClientMock) DeletePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) error {
	key := m.pluginKeyFromBasePath(basePath)
	m.initPluginSources(key)

	for i, s := range m.pluginSources[key] {
		if s.Id == sourceId {
			m.pluginSources[key] = append(m.pluginSources[key][:i], m.pluginSources[key][i+1:]...)
			break
		}
	}
	return nil
}

func (m *ModelCatalogClientMock) RefreshPlugin(client httpclient.HTTPClientInterface, basePath string) (*models.RefreshResult, error) {
	return &models.RefreshResult{
		EntitiesLoaded:  12,
		EntitiesRemoved: 0,
		Duration:        150,
	}, nil
}

func (m *ModelCatalogClientMock) RefreshPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RefreshResult, error) {
	return &models.RefreshResult{
		SourceId:        sourceId,
		EntitiesLoaded:  5,
		EntitiesRemoved: 0,
		Duration:        80,
	}, nil
}

func (m *ModelCatalogClientMock) GetPluginDiagnostics(client httpclient.HTTPClientInterface, basePath string) (*models.PluginDiagnostics, error) {
	key := m.pluginKeyFromBasePath(basePath)
	m.initPluginSources(key)

	var sourceDiags []models.SourceDiagnostic
	for _, s := range m.pluginSources[key] {
		sourceDiags = append(sourceDiags, models.SourceDiagnostic{
			Id:          s.Id,
			Name:        s.Name,
			State:       s.Status.State,
			EntityCount: s.Status.EntityCount,
		})
	}

	return &models.PluginDiagnostics{
		PluginName: key,
		Sources:    sourceDiags,
	}, nil
}

func (m *ModelCatalogClientMock) ValidatePluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.SourceConfigPayload) (*models.DetailedValidationResult, error) {
	return &models.DetailedValidationResult{
		Valid: true,
		Warnings: []models.ValidationError{
			{Field: "properties.password", Message: "Property key 'password' may contain sensitive data - consider using a SecretRef instead"},
		},
		LayerResults: []models.LayerValidationResult{
			{Layer: "yaml_parse", Valid: true},
			{Layer: "strict_fields", Valid: true},
			{Layer: "semantic", Valid: true},
			{Layer: "security_warnings", Valid: true, Errors: []models.ValidationError{
				{Field: "properties.password", Message: "Property key 'password' may contain sensitive data - consider using a SecretRef instead"},
			}},
		},
	}, nil
}

func (m *ModelCatalogClientMock) GetPluginSourceRevisions(client httpclient.HTTPClientInterface, basePath string, sourceId string) (*models.RevisionList, error) {
	return &models.RevisionList{
		Revisions: []models.ConfigRevision{
			{Version: "abc123", Timestamp: "2025-12-15T10:00:00Z", Size: 1024},
			{Version: "def456", Timestamp: "2025-12-14T09:00:00Z", Size: 980},
		},
		Count: 2,
	}, nil
}

func (m *ModelCatalogClientMock) RollbackPluginSource(client httpclient.HTTPClientInterface, basePath string, sourceId string, payload models.RollbackRequest) (*models.RollbackResult, error) {
	return &models.RollbackResult{
		Status:  "rolled_back",
		Version: payload.Version,
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
			SourceLabel:         "MCP YAML Source",
			ToolCount:           12,
			ResourceCount:       8,
			PromptCount:         3,
			Version:             "1.2.0",
			SourceUrl:           "https://github.com/kubeflow/kubernetes-mcp-server",
			LastModified:        "2025-12-15",
			Tags:                []string{"kubernetes", "containers", "orchestration"},
			Readme:              "# Kubernetes MCP Server\n\nProvides tools for managing Kubernetes clusters through the Model Context Protocol.\n\n## Installation\n\n```bash\ndocker pull quay.io/kubeflow/kubernetes-mcp-server:latest\n```\n\n## Usage\n\nConfigure your MCP client to connect via stdio transport.\n\n## Available Tools\n\n- **list_pods**: List pods in a namespace\n- **get_pod**: Get details of a specific pod\n- **create_deployment**: Create a new deployment\n- **delete_pod**: Delete a pod\n- **scale_deployment**: Scale a deployment",
			Tools: []models.McpTool{
				{
					Name:        "list_pods",
					Description: "List all pods in a specified namespace with optional label filtering. Returns pod names, status, restart count, and age.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "namespace", Type: "string", Description: "Kubernetes namespace to list pods from. Defaults to 'default'.", Required: false},
						{Name: "labelSelector", Type: "string", Description: "Label selector to filter pods (e.g. 'app=nginx').", Required: false},
					},
				},
				{
					Name:        "get_pod",
					Description: "Get detailed information about a specific pod including its status, containers, volumes, and events.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name of the pod to retrieve.", Required: true},
						{Name: "namespace", Type: "string", Description: "Namespace of the pod.", Required: true},
					},
				},
				{
					Name:        "create_deployment",
					Description: "Create a new Kubernetes deployment with the specified container image, replica count, and configuration.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name for the new deployment.", Required: true},
						{Name: "namespace", Type: "string", Description: "Target namespace for the deployment.", Required: true},
						{Name: "image", Type: "string", Description: "Container image to deploy (e.g. 'nginx:latest').", Required: true},
						{Name: "replicas", Type: "integer", Description: "Number of pod replicas. Defaults to 1.", Required: false},
					},
				},
				{
					Name:        "delete_pod",
					Description: "Delete a pod from the cluster. Use force option to skip graceful termination.",
					AccessType:  "destructive",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name of the pod to delete.", Required: true},
						{Name: "namespace", Type: "string", Description: "Namespace of the pod.", Required: true},
						{Name: "force", Type: "boolean", Description: "Force immediate deletion without graceful shutdown.", Required: false},
					},
				},
				{
					Name:        "scale_deployment",
					Description: "Scale an existing deployment to the specified number of replicas.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name of the deployment to scale.", Required: true},
						{Name: "namespace", Type: "string", Description: "Namespace of the deployment.", Required: true},
						{Name: "replicas", Type: "integer", Description: "Desired number of replicas.", Required: true},
					},
				},
			},
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
			SourceLabel:         "MCP YAML Source",
			ToolCount:           18,
			ResourceCount:       12,
			PromptCount:         4,
			Version:             "2.0.1",
			SourceUrl:           "https://github.com/openshift/mcp-server",
			LastModified:        "2025-11-20",
			Tags:                []string{"openshift", "kubernetes", "routes", "builds"},
			Readme:              "# OpenShift MCP Server\n\nExtends Kubernetes with OpenShift-specific resources including Routes, BuildConfigs, DeploymentConfigs, and ImageStreams.\n\n## Installation\n\n```bash\ndocker pull quay.io/openshift/mcp-server:latest\n```\n\n## Features\n\n- Manage OpenShift Routes for external access\n- Trigger and monitor builds via BuildConfigs\n- Work with DeploymentConfigs and ImageStreams\n\n## Available Tools\n\n- **list_routes**: List all routes in a namespace\n- **create_route**: Expose a service via a route\n- **get_buildconfig**: Inspect build configurations\n- **start_build**: Trigger a new build\n- **delete_route**: Remove an existing route",
			Tools: []models.McpTool{
				{
					Name:        "list_routes",
					Description: "List all OpenShift routes in the current or specified namespace.",
					AccessType:  "read_only",
				},
				{
					Name:        "create_route",
					Description: "Create a new OpenShift route to expose a service externally.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name for the new route.", Required: true},
						{Name: "service", Type: "string", Description: "Target service to expose.", Required: true},
						{Name: "hostname", Type: "string", Description: "Custom hostname for the route.", Required: false},
					},
				},
				{
					Name:        "get_buildconfig",
					Description: "Get details of an OpenShift BuildConfig including triggers, source, and strategy.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name of the BuildConfig.", Required: true},
						{Name: "namespace", Type: "string", Description: "Namespace of the BuildConfig.", Required: true},
					},
				},
				{
					Name:        "start_build",
					Description: "Trigger a new build from an existing BuildConfig.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "buildconfig", Type: "string", Description: "Name of the BuildConfig to trigger.", Required: true},
						{Name: "namespace", Type: "string", Description: "Namespace of the BuildConfig.", Required: true},
					},
				},
				{
					Name:        "delete_route",
					Description: "Delete an OpenShift route.",
					AccessType:  "destructive",
					Parameters: []models.McpToolParameter{
						{Name: "name", Type: "string", Description: "Name of the route to delete.", Required: true},
						{Name: "namespace", Type: "string", Description: "Namespace of the route.", Required: true},
					},
				},
			},
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
			SourceLabel:         "MCP YAML Source",
			ToolCount:           10,
			ResourceCount:       6,
			PromptCount:         5,
			Version:             "1.0.0",
			SourceUrl:           "https://github.com/ansible/mcp-server",
			Tags:                []string{"ansible", "automation", "playbooks"},
			Readme:              "# Ansible MCP Server\n\nProvides tools for Ansible Automation Platform integration.\n\n## Available Tools\n\n- **run_playbook**: Execute an Ansible playbook\n- **list_roles**: List available Ansible roles\n- **list_collections**: Browse Ansible collections",
			Tools: []models.McpTool{
				{
					Name:        "run_playbook",
					Description: "Execute an Ansible playbook against the specified inventory.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "playbook", Type: "string", Description: "Path to the Ansible playbook file.", Required: true},
						{Name: "inventory", Type: "string", Description: "Path to the inventory file or comma-separated host list.", Required: true},
						{Name: "extra_vars", Type: "string", Description: "Extra variables as JSON string.", Required: false},
					},
				},
				{
					Name:        "list_roles",
					Description: "List all available Ansible roles in the configured roles path.",
					AccessType:  "read_only",
				},
				{
					Name:        "list_collections",
					Description: "List installed Ansible collections, optionally filtered by namespace.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "namespace", Type: "string", Description: "Filter by collection namespace (e.g. 'ansible.builtin').", Required: false},
					},
				},
			},
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
			SourceLabel:         "MCP YAML Source",
			ToolCount:           8,
			ResourceCount:       5,
			PromptCount:         2,
			Version:             "0.9.0",
			Tags:                []string{"postgresql", "database", "sql"},
			Readme:              "# PostgreSQL MCP Server\n\nProvides tools for PostgreSQL database management and query execution.\n\n## Available Tools\n\n- **execute_query**: Run SQL queries\n- **list_tables**: List database tables\n- **describe_table**: Get table schema details\n- **create_index**: Create database indexes",
			Tools: []models.McpTool{
				{
					Name:        "execute_query",
					Description: "Execute a SQL query against a PostgreSQL database and return the results.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "query", Type: "string", Description: "SQL query to execute.", Required: true},
						{Name: "database", Type: "string", Description: "Target database name. Uses default if not specified.", Required: false},
					},
				},
				{
					Name:        "list_tables",
					Description: "List all tables in the specified schema.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "schema", Type: "string", Description: "Database schema to list tables from. Defaults to 'public'.", Required: false},
					},
				},
				{
					Name:        "describe_table",
					Description: "Get the column definitions, constraints, and indexes for a table.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "table", Type: "string", Description: "Name of the table to describe.", Required: true},
						{Name: "schema", Type: "string", Description: "Schema containing the table. Defaults to 'public'.", Required: false},
					},
				},
				{
					Name:        "create_index",
					Description: "Create a new index on a table for the specified columns.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "table", Type: "string", Description: "Table to create the index on.", Required: true},
						{Name: "columns", Type: "string", Description: "Comma-separated list of columns to index.", Required: true},
						{Name: "name", Type: "string", Description: "Name for the index. Auto-generated if not provided.", Required: false},
					},
				},
			},
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
			SourceLabel:         "MCP YAML Source",
			ToolCount:           15,
			ResourceCount:       10,
			PromptCount:         3,
			Version:             "3.1.0",
			SourceUrl:           "https://github.com/github/mcp-server",
			Tags:                []string{"github", "git", "ci-cd"},
			Readme:              "# GitHub MCP Server\n\nIntegrate with GitHub's API through the Model Context Protocol. Manage repositories, issues, pull requests, and more.\n\n## Authentication\n\nRequires a GitHub personal access token with appropriate scopes.\n\n## Available Tools\n\n- **list_repos**: List repositories for a user or organization\n- **create_issue**: Create a new issue in a repository\n- **list_prs**: List pull requests with status filtering\n- **merge_pr**: Merge a pull request with configurable merge strategy",
			Tools: []models.McpTool{
				{
					Name:        "list_repos",
					Description: "List repositories for the authenticated user or a specified organization.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "org", Type: "string", Description: "Organization name. Lists user repos if not specified.", Required: false},
						{Name: "visibility", Type: "string", Description: "Filter by visibility: 'public', 'private', or 'all'.", Required: false},
					},
				},
				{
					Name:        "create_issue",
					Description: "Create a new issue in a GitHub repository.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "repo", Type: "string", Description: "Repository in 'owner/repo' format.", Required: true},
						{Name: "title", Type: "string", Description: "Issue title.", Required: true},
						{Name: "body", Type: "string", Description: "Issue body in Markdown.", Required: false},
						{Name: "labels", Type: "string", Description: "Comma-separated list of label names.", Required: false},
					},
				},
				{
					Name:        "list_prs",
					Description: "List pull requests for a repository with optional state filtering.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "repo", Type: "string", Description: "Repository in 'owner/repo' format.", Required: true},
						{Name: "state", Type: "string", Description: "Filter by state: 'open', 'closed', or 'all'. Defaults to 'open'.", Required: false},
					},
				},
				{
					Name:        "merge_pr",
					Description: "Merge a pull request using the specified merge method.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "repo", Type: "string", Description: "Repository in 'owner/repo' format.", Required: true},
						{Name: "pr_number", Type: "integer", Description: "Pull request number to merge.", Required: true},
						{Name: "method", Type: "string", Description: "Merge method: 'merge', 'squash', or 'rebase'. Defaults to 'merge'.", Required: false},
					},
				},
			},
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
			SourceLabel:         "Community Servers",
			ToolCount:           9,
			ResourceCount:       4,
			PromptCount:         2,
			Version:             "2.3.0",
			Tags:                []string{"slack", "messaging", "communication"},
			Readme:              "# Slack MCP Server\n\nIntegrate with Slack workspaces through MCP. Send messages, manage channels, and search conversations.\n\n## Available Tools\n\n- **send_message**: Send messages to channels or threads\n- **list_channels**: List workspace channels\n- **search_messages**: Search message history",
			Tools: []models.McpTool{
				{
					Name:        "send_message",
					Description: "Send a message to a Slack channel or thread.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "channel", Type: "string", Description: "Channel ID or name to send the message to.", Required: true},
						{Name: "text", Type: "string", Description: "Message text content.", Required: true},
						{Name: "thread_ts", Type: "string", Description: "Thread timestamp to reply in a thread.", Required: false},
					},
				},
				{
					Name:        "list_channels",
					Description: "List channels in the Slack workspace.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "types", Type: "string", Description: "Comma-separated channel types: 'public_channel', 'private_channel'.", Required: false},
					},
				},
				{
					Name:        "search_messages",
					Description: "Search for messages in the workspace matching a query.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "query", Type: "string", Description: "Search query string.", Required: true},
						{Name: "count", Type: "integer", Description: "Maximum number of results to return.", Required: false},
					},
				},
			},
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
			SourceLabel:         "Community Servers",
			ToolCount:           11,
			ResourceCount:       7,
			PromptCount:         3,
			Version:             "1.5.0",
			SourceUrl:           "https://github.com/atlassian/jira-mcp-server",
			Tags:                []string{"jira", "project-management", "atlassian"},
			Readme:              "# Jira MCP Server\n\nManage Jira projects, issues, and workflows through MCP.\n\n## Available Tools\n\n- **create_issue**: Create new Jira issues\n- **get_issue**: Retrieve issue details\n- **transition_issue**: Move issues through workflow states\n- **search_issues**: Search with JQL",
			Tools: []models.McpTool{
				{
					Name:        "create_issue",
					Description: "Create a new issue in a Jira project.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "project", Type: "string", Description: "Project key (e.g. 'PROJ').", Required: true},
						{Name: "summary", Type: "string", Description: "Issue summary/title.", Required: true},
						{Name: "type", Type: "string", Description: "Issue type: 'Bug', 'Story', 'Task', 'Epic'.", Required: true},
						{Name: "description", Type: "string", Description: "Detailed issue description.", Required: false},
					},
				},
				{
					Name:        "get_issue",
					Description: "Get detailed information about a Jira issue.",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "key", Type: "string", Description: "Issue key (e.g. 'PROJ-123').", Required: true},
					},
				},
				{
					Name:        "transition_issue",
					Description: "Move a Jira issue to a new workflow state.",
					AccessType:  "read_write",
					Parameters: []models.McpToolParameter{
						{Name: "key", Type: "string", Description: "Issue key (e.g. 'PROJ-123').", Required: true},
						{Name: "transition", Type: "string", Description: "Target transition name (e.g. 'In Progress', 'Done').", Required: true},
					},
				},
				{
					Name:        "search_issues",
					Description: "Search for issues using JQL (Jira Query Language).",
					AccessType:  "read_only",
					Parameters: []models.McpToolParameter{
						{Name: "jql", Type: "string", Description: "JQL query string.", Required: true},
						{Name: "maxResults", Type: "integer", Description: "Maximum number of results to return.", Required: false},
					},
				},
			},
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

func (m *ModelCatalogClientMock) GetPluginCapabilities(client httpclient.HTTPClientInterface, pluginName string) (json.RawMessage, error) {
	caps := map[string]interface{}{
		"pluginName": pluginName,
		"version":    "v2",
		"entityTypes": []map[string]interface{}{
			{
				"name":        "mcpservers",
				"displayName": "MCP Servers",
				"plural":      "mcpservers",
			},
		},
		"sourceTypes": []map[string]interface{}{
			{
				"name":        "yaml",
				"displayName": "YAML File",
			},
		},
		"actions": []map[string]interface{}{
			{
				"name":        "deploy",
				"displayName": "Deploy",
				"scope":       "entity",
			},
		},
	}

	data, err := json.Marshal(caps)
	if err != nil {
		return nil, fmt.Errorf("error marshaling mock capabilities: %w", err)
	}

	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) GetCatalogEntityList(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, queryParams url.Values) (json.RawMessage, error) {
	result := map[string]interface{}{
		"items":         []interface{}{},
		"size":          0,
		"pageSize":      10,
		"nextPageToken": "",
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling mock entity list: %w", err)
	}

	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) GetCatalogEntity(client httpclient.HTTPClientInterface, pluginName string, entityPlural string, entityName string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"name":       entityName,
		"pluginName": pluginName,
		"entityType": entityPlural,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling mock entity: %w", err)
	}

	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) PostCatalogEntityAction(_ httpclient.HTTPClientInterface, pluginName string, entityPlural string, entityName string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("action executed on %s/%s/%s", pluginName, entityPlural, entityName),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling mock action result: %w", err)
	}

	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) PostCatalogSourceAction(_ httpclient.HTTPClientInterface, pluginName string, sourceId string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("action executed on %s source %s", pluginName, sourceId),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling mock source action result: %w", err)
	}

	return json.RawMessage(data), nil
}
