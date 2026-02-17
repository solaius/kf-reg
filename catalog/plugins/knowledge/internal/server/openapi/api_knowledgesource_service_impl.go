package openapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/models"
	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/service"
)

// KnowledgeSourceCatalogServiceAPIService implements the business logic for the KnowledgeSource Catalog API.
type KnowledgeSourceCatalogServiceAPIService struct {
	services service.Services
}

// NewKnowledgeSourceCatalogServiceAPIService creates a new service instance.
func NewKnowledgeSourceCatalogServiceAPIService(services service.Services) *KnowledgeSourceCatalogServiceAPIService {
	return &KnowledgeSourceCatalogServiceAPIService{
		services: services,
	}
}

// Ensure we implement the DefaultAPIServicer interface.
var _ DefaultAPIServicer = &KnowledgeSourceCatalogServiceAPIService{}

// ListKnowledgeSources implements DefaultAPIServicer.ListKnowledgeSources
func (s *KnowledgeSourceCatalogServiceAPIService) ListKnowledgeSources(ctx context.Context, pageSize int32, pageToken string, q string, filterQuery string, orderBy string, sortOrder string) (ImplResponse, error) {
	listOptions := models.KnowledgeSourceListOptions{
		Query: &q,
	}
	listOptions.PageSize = &pageSize
	listOptions.NextPageToken = &pageToken
	listOptions.FilterQuery = &filterQuery
	listOptions.OrderBy = &orderBy
	listOptions.SortOrder = &sortOrder

	result, err := s.services.KnowledgeSourceRepository.List(listOptions)
	if err != nil {
		return Response(http.StatusInternalServerError, nil), err
	}

	items := make([]KnowledgeSource, len(result.Items))
	for i, item := range result.Items {
		items[i] = convertToOpenAPIModel(item)
	}

	response := KnowledgeSourceList{
		Items:         items,
		NextPageToken: result.NextPageToken,
		Size:          int32(len(items)),
	}

	return Response(http.StatusOK, response), nil
}

// GetKnowledgeSource implements DefaultAPIServicer.GetKnowledgeSource
func (s *KnowledgeSourceCatalogServiceAPIService) GetKnowledgeSource(ctx context.Context, name string) (ImplResponse, error) {
	entity, err := s.services.KnowledgeSourceRepository.GetByName(name)
	if err != nil {
		if errors.Is(err, service.ErrKnowledgeSourceNotFound) {
			return Response(http.StatusNotFound, nil), err
		}
		return Response(http.StatusInternalServerError, nil), err
	}

	return Response(http.StatusOK, convertToOpenAPIModel(entity)), nil
}

// convertToOpenAPIModel converts a database entity to the OpenAPI model type.
func convertToOpenAPIModel(entity models.KnowledgeSource) KnowledgeSource {
	attrs := entity.GetAttributes()
	result := KnowledgeSource{}

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
			case "sourceType":
				if prop.StringValue != nil {
					result.SourceType = *prop.StringValue
				}
			case "location":
				if prop.StringValue != nil {
					result.Location = *prop.StringValue
				}
			case "contentType":
				if prop.StringValue != nil {
					result.ContentType = *prop.StringValue
				}
			case "provider":
				if prop.StringValue != nil {
					result.Provider = *prop.StringValue
				}
			case "status":
				if prop.StringValue != nil {
					result.Status = *prop.StringValue
				}
			case "documentCount":
				if prop.IntValue != nil {
					v := *prop.IntValue
					result.DocumentCount = &v
				}
			case "vectorDimensions":
				if prop.IntValue != nil {
					v := *prop.IntValue
					result.VectorDimensions = &v
				}
			case "indexType":
				if prop.StringValue != nil {
					result.IndexType = *prop.StringValue
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
