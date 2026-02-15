package openapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kubeflow/model-registry/catalog/plugins/mcp/internal/db/models"
	"github.com/kubeflow/model-registry/catalog/plugins/mcp/internal/db/service"
)

// McpServerCatalogServiceAPIService implements the business logic for the McpServer Catalog API.
type McpServerCatalogServiceAPIService struct {
	services service.Services
}

// NewMcpServerCatalogServiceAPIService creates a new service instance.
func NewMcpServerCatalogServiceAPIService(services service.Services) *McpServerCatalogServiceAPIService {
	return &McpServerCatalogServiceAPIService{
		services: services,
	}
}

// Ensure we implement the DefaultAPIServicer interface.
var _ DefaultAPIServicer = &McpServerCatalogServiceAPIService{}

// ListMcpServers implements DefaultAPIServicer.ListMcpServers
// This returns a paginated list of McpServer entities.
func (s *McpServerCatalogServiceAPIService) ListMcpServers(ctx context.Context, pageSize int32, pageToken string, q string, filterQuery string, orderBy string, sortOrder string) (ImplResponse, error) {
	listOptions := models.McpServerListOptions{
		Query: &q,
	}
	listOptions.PageSize = &pageSize
	listOptions.NextPageToken = &pageToken
	listOptions.FilterQuery = &filterQuery
	listOptions.OrderBy = &orderBy
	listOptions.SortOrder = &sortOrder

	result, err := s.services.McpServerRepository.List(listOptions)
	if err != nil {
		return Response(http.StatusInternalServerError, nil), err
	}

	// Convert to OpenAPI model types
	items := make([]McpServer, len(result.Items))
	for i, item := range result.Items {
		items[i] = convertToOpenAPIModel(item)
	}

	response := McpServerList{
		Items:         items,
		NextPageToken: result.NextPageToken,
		Size:          int32(len(items)),
	}

	return Response(http.StatusOK, response), nil
}

// GetMcpServer implements DefaultAPIServicer.GetMcpServer
// This returns a single McpServer by name.
func (s *McpServerCatalogServiceAPIService) GetMcpServer(ctx context.Context, name string) (ImplResponse, error) {
	entity, err := s.services.McpServerRepository.GetByName(name)
	if err != nil {
		if errors.Is(err, service.ErrMcpServerNotFound) {
			return Response(http.StatusNotFound, nil), err
		}
		return Response(http.StatusInternalServerError, nil), err
	}

	return Response(http.StatusOK, convertToOpenAPIModel(entity)), nil
}

// convertToOpenAPIModel converts a database entity to the OpenAPI model type.
func convertToOpenAPIModel(entity models.McpServer) McpServer {
	attrs := entity.GetAttributes()
	result := McpServer{}

	if attrs.Name != nil {
		result.Name = *attrs.Name
	}
	if attrs.ExternalID != nil {
		result.ExternalId = *attrs.ExternalID
	}
	if attrs.CreateTimeSinceEpoch != nil {
		result.CreateTimeSinceEpoch = fmt.Sprintf("%d", *attrs.CreateTimeSinceEpoch)
	}
	if attrs.LastUpdateTimeSinceEpoch != nil {
		result.LastUpdateTimeSinceEpoch = fmt.Sprintf("%d", *attrs.LastUpdateTimeSinceEpoch)
	}
	if entity.GetID() != nil {
		result.Id = fmt.Sprintf("%d", *entity.GetID())
	}

	// Extract properties
	if entity.GetProperties() != nil {
		for _, prop := range *entity.GetProperties() {
			switch prop.Name {
			case "description":
				if prop.StringValue != nil {
					result.Description = *prop.StringValue
				}
			case "serverUrl":
				if prop.StringValue != nil {
					result.ServerUrl = *prop.StringValue
				}
			case "transportType":
				if prop.StringValue != nil {
					result.TransportType = *prop.StringValue
				}
			case "toolCount":
				if prop.IntValue != nil {
					v := *prop.IntValue
					result.ToolCount = &v
				}
			case "resourceCount":
				if prop.IntValue != nil {
					v := *prop.IntValue
					result.ResourceCount = &v
				}
			case "promptCount":
				if prop.IntValue != nil {
					v := *prop.IntValue
					result.PromptCount = &v
				}
			}
		}
	}

	// Extract custom properties
	if entity.GetCustomProperties() != nil {
		customProps := make(map[string]interface{})
		for _, prop := range *entity.GetCustomProperties() {
			switch {
			case prop.StringValue != nil:
				customProps[prop.Name] = *prop.StringValue
			case prop.IntValue != nil:
				customProps[prop.Name] = *prop.IntValue
			case prop.DoubleValue != nil:
				customProps[prop.Name] = *prop.DoubleValue
			case prop.BoolValue != nil:
				customProps[prop.Name] = *prop.BoolValue
			}
		}
		if len(customProps) > 0 {
			result.CustomProperties = customProps
		}
	}

	return result
}
