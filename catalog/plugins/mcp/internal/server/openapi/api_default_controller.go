package openapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DefaultAPIController binds http requests to the api service and writes the service results to the http response.
type DefaultAPIController struct {
	service      DefaultAPIServicer
	errorHandler ErrorHandler
}

// DefaultAPIOption for how the controller is set up.
type DefaultAPIOption func(*DefaultAPIController)

// WithDefaultAPIErrorHandler inject ErrorHandler into controller.
func WithDefaultAPIErrorHandler(h ErrorHandler) DefaultAPIOption {
	return func(c *DefaultAPIController) {
		c.errorHandler = h
	}
}

// NewDefaultAPIController creates a default api controller.
func NewDefaultAPIController(s DefaultAPIServicer, opts ...DefaultAPIOption) *DefaultAPIController {
	controller := &DefaultAPIController{
		service:      s,
		errorHandler: DefaultErrorHandler,
	}

	for _, opt := range opts {
		opt(controller)
	}

	return controller
}

// Routes returns all the api routes for the DefaultAPIController.
func (c *DefaultAPIController) Routes() Routes {
	return Routes{
		"ListMcpServers": {
			"ListMcpServers",
			strings.ToUpper("Get"),
			"/api/mcp_catalog/v1alpha1/mcpservers",
			c.ListMcpServers,
		},
		"GetMcpServer": {
			"GetMcpServer",
			strings.ToUpper("Get"),
			"/api/mcp_catalog/v1alpha1/mcpservers/{name}",
			c.GetMcpServer,
		},
	}
}

// OrderedRoutes returns all the api routes in a deterministic order.
func (c *DefaultAPIController) OrderedRoutes() []Route {
	return []Route{
		{
			"ListMcpServers",
			strings.ToUpper("Get"),
			"/api/mcp_catalog/v1alpha1/mcpservers",
			c.ListMcpServers,
		},
		{
			"GetMcpServer",
			strings.ToUpper("Get"),
			"/api/mcp_catalog/v1alpha1/mcpservers/{name}",
			c.GetMcpServer,
		},
	}
}

// ListMcpServers - List McpServer entities.
func (c *DefaultAPIController) ListMcpServers(w http.ResponseWriter, r *http.Request) {
	query, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		c.errorHandler(w, r, &ParsingError{Err: err}, nil)
		return
	}

	var pageSizeParam int32 = 20
	if query.Has("pageSize") {
		parsed, err := parseInt32(query.Get("pageSize"))
		if err != nil {
			c.errorHandler(w, r, &ParsingError{Param: "pageSize", Err: err}, nil)
			return
		}
		pageSizeParam = parsed
	}

	var pageTokenParam string
	if query.Has("pageToken") {
		pageTokenParam = query.Get("pageToken")
	}

	var qParam string
	if query.Has("q") {
		qParam = query.Get("q")
	}

	var filterQueryParam string
	if query.Has("filterQuery") {
		filterQueryParam = query.Get("filterQuery")
	}

	var orderByParam string
	if query.Has("orderBy") {
		orderByParam = query.Get("orderBy")
	}

	var sortOrderParam string = "ASC"
	if query.Has("sortOrder") {
		sortOrderParam = query.Get("sortOrder")
	}

	result, err := c.service.ListMcpServers(r.Context(), pageSizeParam, pageTokenParam, qParam, filterQueryParam, orderByParam, sortOrderParam)
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	_ = EncodeJSONResponse(result.Body, &result.Code, w)
}

// GetMcpServer - Get a McpServer by name.
func (c *DefaultAPIController) GetMcpServer(w http.ResponseWriter, r *http.Request) {
	nameParam := chi.URLParam(r, "name")
	if nameParam == "" {
		c.errorHandler(w, r, &RequiredError{"name"}, nil)
		return
	}

	result, err := c.service.GetMcpServer(r.Context(), nameParam)
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	_ = EncodeJSONResponse(result.Body, &result.Code, w)
}
