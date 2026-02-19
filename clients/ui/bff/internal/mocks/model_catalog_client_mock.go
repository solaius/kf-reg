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

// CatalogGovernanceInterface mock implementations.

func (m *ModelCatalogClientMock) GetGovernance(_ httpclient.HTTPClientInterface, plugin, kind, name string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"assetRef":   map[string]string{"plugin": plugin, "kind": kind, "name": name},
		"governance": map[string]interface{}{"lifecycle": map[string]string{"state": "draft"}},
	}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) PatchGovernance(_ httpclient.HTTPClientInterface, plugin, kind, name string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{
		"assetRef":   map[string]string{"plugin": plugin, "kind": kind, "name": name},
		"governance": map[string]interface{}{"lifecycle": map[string]string{"state": "draft"}},
	}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) GetGovernanceHistory(_ httpclient.HTTPClientInterface, _, _, _ string, _ url.Values) (json.RawMessage, error) {
	result := map[string]interface{}{"events": []interface{}{}, "totalSize": 0}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) PostGovernanceAction(_ httpclient.HTTPClientInterface, _, _, _, _ string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{"action": "lifecycle.setState", "status": "completed"}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) ListVersions(_ httpclient.HTTPClientInterface, _, _, _ string, _ url.Values) (json.RawMessage, error) {
	result := map[string]interface{}{"versions": []interface{}{}, "totalSize": 0}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) CreateVersion(_ httpclient.HTTPClientInterface, _, _, _ string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{"versionId": "v1.0:mock", "versionLabel": "v1.0"}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) ListBindings(_ httpclient.HTTPClientInterface, _, _, _ string) (json.RawMessage, error) {
	result := map[string]interface{}{"bindings": []interface{}{}}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) SetBinding(_ httpclient.HTTPClientInterface, _, _, _, _ string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{"environment": "dev", "versionId": "v1.0:mock"}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) ListApprovals(_ httpclient.HTTPClientInterface, _ url.Values) (json.RawMessage, error) {
	result := map[string]interface{}{"requests": []interface{}{}, "totalSize": 0}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) GetApproval(_ httpclient.HTTPClientInterface, id string) (json.RawMessage, error) {
	result := map[string]interface{}{"id": id, "status": "pending"}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) PostApprovalDecision(_ httpclient.HTTPClientInterface, _ string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{"decision": "approved", "status": "pending"}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) CancelApproval(_ httpclient.HTTPClientInterface, id string, _ io.Reader) (json.RawMessage, error) {
	result := map[string]interface{}{"status": "canceled", "requestId": id}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}

func (m *ModelCatalogClientMock) ListPolicies(_ httpclient.HTTPClientInterface) (json.RawMessage, error) {
	result := map[string]interface{}{"policies": []interface{}{}}
	data, _ := json.Marshal(result)
	return json.RawMessage(data), nil
}
