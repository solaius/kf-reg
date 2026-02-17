package openapi

import (
	"context"
	"net/http"
)

// DefaultAPIRouter defines the required methods for binding the api requests to responses.
type DefaultAPIRouter interface {
	ListKnowledgeSources(http.ResponseWriter, *http.Request)
	GetKnowledgeSource(http.ResponseWriter, *http.Request)
}

// DefaultAPIServicer defines the api actions for the default service.
type DefaultAPIServicer interface {
	ListKnowledgeSources(ctx context.Context, pageSize int32, pageToken string, q string, filterQuery string, orderBy string, sortOrder string) (ImplResponse, error)
	GetKnowledgeSource(ctx context.Context, name string) (ImplResponse, error)
}
