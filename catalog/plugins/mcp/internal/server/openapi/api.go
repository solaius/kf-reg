package openapi

import (
	"context"
	"net/http"
)

// DefaultAPIRouter defines the required methods for binding the api requests to a responses.
type DefaultAPIRouter interface {
	ListMcpServers(http.ResponseWriter, *http.Request)
	GetMcpServer(http.ResponseWriter, *http.Request)
}

// DefaultAPIServicer defines the api actions for the default service.
type DefaultAPIServicer interface {
	ListMcpServers(ctx context.Context, pageSize int32, pageToken string, q string, filterQuery string, orderBy string, sortOrder string) (ImplResponse, error)
	GetMcpServer(ctx context.Context, name string) (ImplResponse, error)
}
