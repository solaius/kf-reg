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
		"ListKnowledgeSources": {
			"ListKnowledgeSources",
			strings.ToUpper("Get"),
			"/api/knowledge_catalog/v1alpha1/knowledgesources",
			c.ListKnowledgeSources,
		},
		"GetKnowledgeSource": {
			"GetKnowledgeSource",
			strings.ToUpper("Get"),
			"/api/knowledge_catalog/v1alpha1/knowledgesources/{name}",
			c.GetKnowledgeSource,
		},
	}
}

// OrderedRoutes returns all the api routes in a deterministic order.
func (c *DefaultAPIController) OrderedRoutes() []Route {
	return []Route{
		{
			"ListKnowledgeSources",
			strings.ToUpper("Get"),
			"/api/knowledge_catalog/v1alpha1/knowledgesources",
			c.ListKnowledgeSources,
		},
		{
			"GetKnowledgeSource",
			strings.ToUpper("Get"),
			"/api/knowledge_catalog/v1alpha1/knowledgesources/{name}",
			c.GetKnowledgeSource,
		},
	}
}

// ListKnowledgeSources - List KnowledgeSource entities.
func (c *DefaultAPIController) ListKnowledgeSources(w http.ResponseWriter, r *http.Request) {
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

	result, err := c.service.ListKnowledgeSources(r.Context(), pageSizeParam, pageTokenParam, qParam, filterQueryParam, orderByParam, sortOrderParam)
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	_ = EncodeJSONResponse(result.Body, &result.Code, w)
}

// GetKnowledgeSource - Get a KnowledgeSource by name.
func (c *DefaultAPIController) GetKnowledgeSource(w http.ResponseWriter, r *http.Request) {
	nameParam := chi.URLParam(r, "name")
	if nameParam == "" {
		c.errorHandler(w, r, &RequiredError{"name"}, nil)
		return
	}

	result, err := c.service.GetKnowledgeSource(r.Context(), nameParam)
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	_ = EncodeJSONResponse(result.Body, &result.Code, w)
}
