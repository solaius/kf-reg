package openapi

import (
	"errors"
	"fmt"
	"net/http"
)

// ParsingError indicates that an error has occurred when parsing request parameters.
type ParsingError struct {
	Param string
	Err   error
}

func (e *ParsingError) Unwrap() error {
	return e.Err
}

func (e *ParsingError) Error() string {
	if e.Param == "" {
		return e.Err.Error()
	}
	return e.Param + ": " + e.Err.Error()
}

// RequiredError indicates that a required parameter is missing.
type RequiredError struct {
	Field string
}

func (e *RequiredError) Error() string {
	return fmt.Sprintf("required field '%s' is zero value.", e.Field)
}

// ErrorHandler defines the required method for handling errors.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error, result *ImplResponse)

// DefaultErrorHandler defines the default logic on how to handle errors from the controller.
func DefaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error, result *ImplResponse) {
	var parsingErr *ParsingError
	if ok := errors.As(err, &parsingErr); ok {
		code := http.StatusBadRequest
		_ = EncodeJSONResponse(map[string]string{"error": err.Error()}, &code, w)
		return
	}

	var requiredErr *RequiredError
	if ok := errors.As(err, &requiredErr); ok {
		code := http.StatusUnprocessableEntity
		_ = EncodeJSONResponse(map[string]string{"error": err.Error()}, &code, w)
		return
	}

	_ = EncodeJSONResponse(result.Body, &result.Code, w)
}
