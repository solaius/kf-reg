package openapi

import (
	"net/http"
)

// Route defines the parameters for an api endpoint.
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is a map of defined api endpoints.
type Routes map[string]Route

// Router defines the required methods for retrieving api routes.
type Router interface {
	Routes() Routes
	OrderedRoutes() []Route
}
