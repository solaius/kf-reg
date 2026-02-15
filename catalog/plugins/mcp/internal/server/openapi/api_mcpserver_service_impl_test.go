package openapi

import (
	"testing"

	"github.com/kubeflow/model-registry/catalog/plugins/mcp/internal/db/models"
	sharedmodels "github.com/kubeflow/model-registry/internal/db/models"
)

func TestConvertToOpenAPIModel(t *testing.T) {
	name := "test-server"
	extID := "ext-123"
	id := int32(42)
	var createTime int64 = 1000
	var updateTime int64 = 2000
	serverUrl := "https://mcp.example.com/test"
	transportType := "stdio"
	toolCount := int32(5)
	resourceCount := int32(3)
	promptCount := int32(2)
	description := "A test MCP server"

	entity := &models.McpServerImpl{
		ID:     &id,
		TypeID: new(int32),
		Attributes: &models.McpServerAttributes{
			Name:                     &name,
			ExternalID:               &extID,
			CreateTimeSinceEpoch:     &createTime,
			LastUpdateTimeSinceEpoch: &updateTime,
		},
	}

	props := []sharedmodels.Properties{
		sharedmodels.NewStringProperty("description", description, false),
		sharedmodels.NewStringProperty("serverUrl", serverUrl, false),
		sharedmodels.NewStringProperty("transportType", transportType, false),
		sharedmodels.NewIntProperty("toolCount", toolCount, false),
		sharedmodels.NewIntProperty("resourceCount", resourceCount, false),
		sharedmodels.NewIntProperty("promptCount", promptCount, false),
	}
	entity.Properties = &props

	customProps := []sharedmodels.Properties{
		sharedmodels.NewStringProperty("category", "development", true),
	}
	entity.CustomProperties = &customProps

	result := convertToOpenAPIModel(entity)

	if result.Name != name {
		t.Errorf("expected name %q, got %q", name, result.Name)
	}
	if result.ExternalId != extID {
		t.Errorf("expected externalId %q, got %q", extID, result.ExternalId)
	}
	if result.Id != "42" {
		t.Errorf("expected id '42', got %q", result.Id)
	}
	if result.CreateTimeSinceEpoch != "1000" {
		t.Errorf("expected createTimeSinceEpoch '1000', got %q", result.CreateTimeSinceEpoch)
	}
	if result.LastUpdateTimeSinceEpoch != "2000" {
		t.Errorf("expected lastUpdateTimeSinceEpoch '2000', got %q", result.LastUpdateTimeSinceEpoch)
	}
	if result.Description != description {
		t.Errorf("expected description %q, got %q", description, result.Description)
	}
	if result.ServerUrl != serverUrl {
		t.Errorf("expected serverUrl %q, got %q", serverUrl, result.ServerUrl)
	}
	if result.TransportType != transportType {
		t.Errorf("expected transportType %q, got %q", transportType, result.TransportType)
	}
	if result.ToolCount == nil || *result.ToolCount != toolCount {
		t.Errorf("expected toolCount %d, got %v", toolCount, result.ToolCount)
	}
	if result.ResourceCount == nil || *result.ResourceCount != resourceCount {
		t.Errorf("expected resourceCount %d, got %v", resourceCount, result.ResourceCount)
	}
	if result.PromptCount == nil || *result.PromptCount != promptCount {
		t.Errorf("expected promptCount %d, got %v", promptCount, result.PromptCount)
	}
	if result.CustomProperties == nil {
		t.Fatal("expected custom properties")
	}
	if v, ok := result.CustomProperties["category"].(string); !ok || v != "development" {
		t.Errorf("expected custom property 'category' = 'development', got %v", result.CustomProperties["category"])
	}
}

func TestConvertToOpenAPIModelMinimal(t *testing.T) {
	name := "minimal-server"
	entity := models.NewMcpServer(&models.McpServerAttributes{
		Name: &name,
	})

	result := convertToOpenAPIModel(entity)

	if result.Name != name {
		t.Errorf("expected name %q, got %q", name, result.Name)
	}
	if result.ServerUrl != "" {
		t.Errorf("expected empty serverUrl, got %q", result.ServerUrl)
	}
	if result.ToolCount != nil {
		t.Errorf("expected nil toolCount, got %v", result.ToolCount)
	}
}
